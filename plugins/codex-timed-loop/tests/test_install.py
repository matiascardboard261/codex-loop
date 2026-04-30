from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


PLUGIN_ROOT = Path(__file__).resolve().parents[1]
INSTALLER = PLUGIN_ROOT / "scripts" / "install.py"
UNINSTALLER = PLUGIN_ROOT / "scripts" / "uninstall.py"
MANAGED_STOP_COMMAND = 'python3 "${CODEX_HOME:-$HOME/.codex}/codex-timed-loop/hooks/loop_stop.py"'


class InstallerTests(unittest.TestCase):
    def run_script(self, script: Path, codex_home: Path) -> subprocess.CompletedProcess[str]:
        env = dict(os.environ)
        env["CODEX_HOME"] = str(codex_home)
        return subprocess.run(
            ["python3", str(script)],
            cwd=str(PLUGIN_ROOT),
            env=env,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_install_creates_managed_files_and_preserves_existing_hooks(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            codex_home.mkdir(parents=True)
            (codex_home / "hooks.json").write_text(
                json.dumps(
                    {
                        "hooks": {
                            "Stop": [
                                {
                                    "hooks": [
                                        {
                                            "type": "command",
                                            "command": "python3 ./custom_stop.py",
                                        }
                                    ]
                                }
                            ]
                        }
                    }
                ),
                encoding="utf-8",
            )
            (codex_home / "config.toml").write_text("[features]\nother_flag = true\n", encoding="utf-8")

            result = self.run_script(INSTALLER, codex_home)
            self.assertEqual(result.returncode, 0, result.stderr)

            hooks_doc = json.loads((codex_home / "hooks.json").read_text(encoding="utf-8"))
            stop_hooks = hooks_doc["hooks"]["Stop"]
            commands = [hook["command"] for group in stop_hooks for hook in group["hooks"]]
            self.assertIn("python3 ./custom_stop.py", commands)
            self.assertIn(MANAGED_STOP_COMMAND, commands)

            runtime_dir = codex_home / "codex-timed-loop"
            self.assertTrue((runtime_dir / "loops").is_dir())
            self.assertTrue((runtime_dir / "hooks" / "loop_user_prompt_submit.py").is_file())
            self.assertTrue((runtime_dir / "config.toml").is_file())

            config_text = (codex_home / "config.toml").read_text(encoding="utf-8")
            self.assertIn("codex_hooks = true", config_text)
            self.assertIn("other_flag = true", config_text)

    def test_install_rejects_inline_hooks_in_config(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            codex_home.mkdir(parents=True)
            (codex_home / "config.toml").write_text("[[hooks.Stop]]\n", encoding="utf-8")

            result = self.run_script(INSTALLER, codex_home)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("already defines inline hooks", result.stderr)

    def test_uninstall_removes_only_managed_artifacts(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            codex_home = Path(temp_dir) / ".codex-home"
            codex_home.mkdir(parents=True)
            (codex_home / "hooks.json").write_text(
                json.dumps(
                    {
                        "hooks": {
                            "Stop": [
                                {
                                    "hooks": [
                                        {
                                            "type": "command",
                                            "command": "python3 ./custom_stop.py",
                                        }
                                    ]
                                }
                            ]
                        }
                    }
                ),
                encoding="utf-8",
            )
            (codex_home / "config.toml").write_text("[features]\nother_flag = true\n", encoding="utf-8")

            install_result = self.run_script(INSTALLER, codex_home)
            self.assertEqual(install_result.returncode, 0, install_result.stderr)

            uninstall_result = self.run_script(UNINSTALLER, codex_home)
            self.assertEqual(uninstall_result.returncode, 0, uninstall_result.stderr)

            hooks_doc = json.loads((codex_home / "hooks.json").read_text(encoding="utf-8"))
            stop_hooks = hooks_doc["hooks"]["Stop"]
            commands = [hook["command"] for group in stop_hooks for hook in group["hooks"]]
            self.assertEqual(commands, ["python3 ./custom_stop.py"])
            self.assertFalse((codex_home / "codex-timed-loop").exists())

            config_text = (codex_home / "config.toml").read_text(encoding="utf-8")
            self.assertIn("codex_hooks = true", config_text)
            self.assertIn("other_flag = true", config_text)


if __name__ == "__main__":
    unittest.main()
