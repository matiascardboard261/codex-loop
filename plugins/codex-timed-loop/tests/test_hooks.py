from __future__ import annotations

from datetime import timedelta
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock


HOOKS_DIR = Path(__file__).resolve().parents[1] / "templates" / "hooks"
if str(HOOKS_DIR) not in sys.path:
    sys.path.insert(0, str(HOOKS_DIR))

from loop_common import (  # noqa: E402
    build_loop_record,
    create_loop_path,
    extract_activation,
    parse_duration,
    replace_loop_file,
    runtime_config_path,
    runtime_loops_dir,
    utc_now,
)
from loop_stop import handle_stop  # noqa: E402
from loop_user_prompt_submit import handle_user_prompt_submit  # noqa: E402


class HookTests(unittest.TestCase):
    def write_runtime_config(self, codex_home: Path, text: str) -> None:
        runtime_dir = codex_home / "codex-timed-loop"
        runtime_dir.mkdir(parents=True, exist_ok=True)
        (runtime_dir / "config.toml").write_text(text, encoding="utf-8")

    def test_parse_duration_accepts_aliases(self) -> None:
        cases = {
            "30m": 1800,
            "30min": 1800,
            "1h 30m": 5400,
            "2 hours": 7200,
            "45sec": 45,
            "1D 2H 3MIN 4s": 93784,
        }
        for value, expected in cases.items():
            with self.subTest(value=value):
                self.assertEqual(parse_duration(value), expected)

    def test_extract_activation_requires_exactly_one_limit(self) -> None:
        with self.assertRaisesRegex(ValueError, "exactly one"):
            extract_activation('[[CODEX_LOOP name="qa"]]\nRun QA.')
        with self.assertRaisesRegex(ValueError, "exactly one"):
            extract_activation('[[CODEX_LOOP name="qa" min="30m" rounds="3"]]\nRun QA.')
        with self.assertRaisesRegex(ValueError, "invalid rounds"):
            extract_activation('[[CODEX_LOOP name="qa" rounds="0"]]\nRun QA.')

    def test_user_prompt_submit_creates_time_loop_file(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            repo_root = Path(temp_dir) / "repo"
            repo_root.mkdir(parents=True)
            payload = {
                "session_id": "sess-1",
                "cwd": str(repo_root),
                "prompt": '[[CODEX_LOOP name="release-stress-qa" min="6h"]]\nRun the QA task.',
            }
            now = utc_now()

            with mock.patch.dict(os.environ, {"CODEX_HOME": str(codex_home)}, clear=False):
                result = handle_user_prompt_submit(payload, now=now)
                self.assertIsNone(result)
                loop_files = sorted(runtime_loops_dir(create=False).glob("*.json"))

            self.assertEqual(len(loop_files), 1)
            data = json.loads(loop_files[0].read_text(encoding="utf-8"))
            self.assertEqual(data["status"], "active")
            self.assertEqual(data["name"], "release-stress-qa")
            self.assertEqual(data["task_prompt"], "Run the QA task.")
            self.assertEqual(data["min_duration_seconds"], 21600)
            self.assertEqual(data["limit_mode"], "time")
            self.assertEqual(data["workspace_root"], str(repo_root.resolve()))

    def test_user_prompt_submit_supersedes_previous_loop_for_session(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            repo_root = Path(temp_dir) / "repo"
            repo_root.mkdir(parents=True)
            first_payload = {
                "session_id": "sess-1",
                "cwd": str(repo_root),
                "prompt": '[[CODEX_LOOP name="first-run" min="1h"]]\nFirst task.',
            }
            second_payload = {
                "session_id": "sess-1",
                "cwd": str(repo_root),
                "prompt": '[[CODEX_LOOP name="second-run" rounds="2"]]\nSecond task.',
            }

            with mock.patch.dict(os.environ, {"CODEX_HOME": str(codex_home)}, clear=False):
                handle_user_prompt_submit(first_payload, now=utc_now())
                handle_user_prompt_submit(second_payload, now=utc_now() + timedelta(minutes=1))
                loop_files = sorted(runtime_loops_dir(create=False).glob("*.json"))

            self.assertEqual(len(loop_files), 2)
            data = [json.loads(path.read_text(encoding="utf-8")) for path in loop_files]
            statuses = {item["name"]: item["status"] for item in data}
            self.assertEqual(statuses["first-run"], "superseded")
            self.assertEqual(statuses["second-run"], "active")

    def test_prompt_without_header_is_ignored(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            repo_root = Path(temp_dir) / "repo"
            repo_root.mkdir(parents=True)
            payload = {
                "session_id": "sess-1",
                "cwd": str(repo_root),
                "prompt": "Do the normal task without a loop header.",
            }

            with mock.patch.dict(os.environ, {"CODEX_HOME": str(codex_home)}, clear=False):
                result = handle_user_prompt_submit(payload, now=utc_now())
                loop_files = list(runtime_loops_dir(create=False).glob("*.json"))

            self.assertIsNone(result)
            self.assertEqual(loop_files, [])

    def test_stop_continues_before_deadline_with_optional_guidance(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            repo_root = Path(temp_dir) / "repo"
            skill_dir = repo_root / ".agents" / "skills" / "focused-qa"
            skill_dir.mkdir(parents=True)
            (skill_dir / "SKILL.md").write_text("placeholder", encoding="utf-8")
            start = utc_now()

            with mock.patch.dict(os.environ, {"CODEX_HOME": str(codex_home)}, clear=False):
                self.write_runtime_config(
                    codex_home,
                    'optional_skill_name = "focused-qa"\n'
                    'optional_skill_path = ".agents/skills/focused-qa"\n'
                    'extra_continuation_guidance = "Capture concrete evidence before you stop."\n',
                )
                activation = extract_activation(
                    '[[CODEX_LOOP name="release-stress-qa" min="6h"]]\nRun the QA task.'
                )
                self.assertIsNotNone(activation)
                loop = build_loop_record(
                    session_id="sess-1",
                    cwd=str(repo_root),
                    workspace_root=repo_root.resolve(),
                    activation=activation,
                    now=start,
                )
                path = create_loop_path(loop["slug"], start)
                replace_loop_file(path, loop)

                result = handle_stop(
                    {
                        "session_id": "sess-1",
                        "cwd": str(repo_root),
                        "last_assistant_message": "Task looks complete.",
                    },
                    now=start + timedelta(minutes=30),
                )

            self.assertIsNotNone(result)
            self.assertEqual(result["decision"], "block")
            self.assertIn("Remaining time", result["reason"])
            self.assertIn("focused-qa", result["reason"])
            self.assertIn(str((skill_dir / "SKILL.md").resolve()), result["reason"])
            self.assertIn("Capture concrete evidence before you stop.", result["reason"])
            updated = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(updated["continue_count"], 1)
            self.assertEqual(updated["status"], "active")

    def test_stop_marks_time_loop_completed_after_deadline(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            repo_root = Path(temp_dir) / "repo"
            repo_root.mkdir(parents=True)
            start = utc_now()

            with mock.patch.dict(os.environ, {"CODEX_HOME": str(codex_home)}, clear=False):
                activation = extract_activation(
                    '[[CODEX_LOOP name="release-stress-qa" min="5m"]]\nRun the QA task.'
                )
                self.assertIsNotNone(activation)
                loop = build_loop_record(
                    session_id="sess-1",
                    cwd=str(repo_root),
                    workspace_root=repo_root.resolve(),
                    activation=activation,
                    now=start,
                )
                path = create_loop_path(loop["slug"], start)
                replace_loop_file(path, loop)

                result = handle_stop(
                    {"session_id": "sess-1", "cwd": str(repo_root), "last_assistant_message": "Done."},
                    now=start + timedelta(minutes=6),
                )

            self.assertIsNone(result)
            updated = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(updated["status"], "completed")

    def test_stop_rounds_mode_completes_target_rounds(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            repo_root = Path(temp_dir) / "repo"
            repo_root.mkdir(parents=True)
            start = utc_now()

            with mock.patch.dict(os.environ, {"CODEX_HOME": str(codex_home)}, clear=False):
                activation = extract_activation(
                    '[[CODEX_LOOP name="release-stress-qa" rounds="3"]]\nRun the QA task.'
                )
                self.assertIsNotNone(activation)
                loop = build_loop_record(
                    session_id="sess-1",
                    cwd=str(repo_root),
                    workspace_root=repo_root.resolve(),
                    activation=activation,
                    now=start,
                )
                path = create_loop_path(loop["slug"], start)
                replace_loop_file(path, loop)

                first = handle_stop(
                    {"session_id": "sess-1", "cwd": str(repo_root), "last_assistant_message": "Round one done."},
                    now=start + timedelta(minutes=5),
                )
                second = handle_stop(
                    {"session_id": "sess-1", "cwd": str(repo_root), "last_assistant_message": "Round two done."},
                    now=start + timedelta(minutes=10),
                )
                third = handle_stop(
                    {"session_id": "sess-1", "cwd": str(repo_root), "last_assistant_message": "Round three done."},
                    now=start + timedelta(minutes=15),
                )

            self.assertIsNotNone(first)
            self.assertEqual(first["decision"], "block")
            self.assertIn("Round 2 of 3 begins now.", first["reason"])
            self.assertIsNotNone(second)
            self.assertEqual(second["decision"], "block")
            self.assertIn("Round 3 of 3 begins now.", second["reason"])
            self.assertIsNone(third)
            updated = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(updated["status"], "completed")
            self.assertEqual(updated["completed_rounds"], 3)

    def test_stop_escalates_once_then_cuts_short_in_rounds_mode(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            repo_root = Path(temp_dir) / "repo"
            repo_root.mkdir(parents=True)
            start = utc_now()

            with mock.patch.dict(os.environ, {"CODEX_HOME": str(codex_home)}, clear=False):
                activation = extract_activation(
                    '[[CODEX_LOOP name="release-stress-qa" rounds="5"]]\nRun the QA task.'
                )
                self.assertIsNotNone(activation)
                loop = build_loop_record(
                    session_id="sess-1",
                    cwd=str(repo_root),
                    workspace_root=repo_root.resolve(),
                    activation=activation,
                    now=start,
                )
                loop["rapid_stop_count"] = 2
                loop["last_stop_at"] = loop["started_at"]
                path = create_loop_path(loop["slug"], start)
                replace_loop_file(path, loop)

                escalation = handle_stop(
                    {"session_id": "sess-1", "cwd": str(repo_root), "last_assistant_message": "Stopped again."},
                    now=start + timedelta(seconds=90),
                )
                cut_short = handle_stop(
                    {"session_id": "sess-1", "cwd": str(repo_root), "last_assistant_message": "Stopped quickly again."},
                    now=start + timedelta(seconds=150),
                )

            self.assertIsNotNone(escalation)
            self.assertEqual(escalation["decision"], "block")
            self.assertIn("Broaden the scope materially", escalation["reason"])
            self.assertIsNotNone(cut_short)
            self.assertEqual(cut_short["continue"], False)
            updated = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(updated["status"], "cut_short")
            self.assertEqual(updated["completed_rounds"], 2)


if __name__ == "__main__":
    unittest.main()
