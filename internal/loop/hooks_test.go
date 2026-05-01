package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUserPromptSubmitCreatesTimeLoopFile(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	now := fixedTime()

	result, err := HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    `[[CODEX_LOOP name="release-stress-qa" min="6h"]]` + "\nRun the QA task.",
	}, now)
	if err != nil {
		t.Fatalf("handle user prompt submit: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil hook result, got %#v", result)
	}

	loopFiles, err := IterLoopRecords(paths)
	if err != nil {
		t.Fatalf("iterate loops: %v", err)
	}
	if len(loopFiles) != 1 {
		t.Fatalf("expected 1 loop file, got %d", len(loopFiles))
	}
	record := loopFiles[0].Record
	if record.Status != StatusActive {
		t.Fatalf("expected active status, got %q", record.Status)
	}
	if record.Name != "release-stress-qa" {
		t.Fatalf("unexpected name %q", record.Name)
	}
	if record.TaskPrompt != "Run the QA task." {
		t.Fatalf("unexpected task prompt %q", record.TaskPrompt)
	}
	if record.MinDurationSeconds == nil || *record.MinDurationSeconds != 21600 {
		t.Fatalf("unexpected min duration %#v", record.MinDurationSeconds)
	}
	if record.LimitMode != LimitModeTime {
		t.Fatalf("unexpected limit mode %q", record.LimitMode)
	}
	if record.WorkspaceRoot != repoRoot {
		t.Fatalf("expected workspace root %q, got %q", repoRoot, record.WorkspaceRoot)
	}
}

func TestUserPromptSubmitSupersedesPreviousLoopForSession(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()

	_, err := HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    `[[CODEX_LOOP name="first-run" min="1h"]]` + "\nFirst task.",
	}, start)
	if err != nil {
		t.Fatalf("first activation: %v", err)
	}
	_, err = HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    `[[CODEX_LOOP name="second-run" rounds="2"]]` + "\nSecond task.",
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("second activation: %v", err)
	}

	loopFiles, err := IterLoopRecords(paths)
	if err != nil {
		t.Fatalf("iterate loops: %v", err)
	}
	if len(loopFiles) != 2 {
		t.Fatalf("expected 2 loop files, got %d", len(loopFiles))
	}
	statuses := map[string]string{}
	for _, loopFile := range loopFiles {
		statuses[loopFile.Record.Name] = loopFile.Record.Status
	}
	if statuses["first-run"] != StatusSuperseded {
		t.Fatalf("expected first-run superseded, got %q", statuses["first-run"])
	}
	if statuses["second-run"] != StatusActive {
		t.Fatalf("expected second-run active, got %q", statuses["second-run"])
	}
}

func TestPromptWithoutHeaderIsIgnored(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}

	result, err := HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    "Do the normal task without a loop header.",
	}, fixedTime())
	if err != nil {
		t.Fatalf("handle prompt: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	loopFiles, err := IterLoopRecords(paths)
	if err != nil {
		t.Fatalf("iterate loops: %v", err)
	}
	if len(loopFiles) != 0 {
		t.Fatalf("expected no loops, got %d", len(loopFiles))
	}
}

func TestStopContinuesBeforeDeadlineWithOptionalGuidance(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	skillDir := filepath.Join(repoRoot, ".agents", "skills", "focused-qa")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("placeholder"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	writeRuntimeConfig(t, paths, `optional_skill_name = "focused-qa"
optional_skill_path = ".agents/skills/focused-qa"
extra_continuation_guidance = "Capture concrete evidence before you stop."
`)

	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" min="6h"]]`+"\nRun the QA task.", start)
	latest := "Task looks complete."
	result, err := HandleStop(paths, StopPayload{
		SessionID:            "sess-1",
		CWD:                  repoRoot,
		LastAssistantMessage: &latest,
	}, start.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result == nil || result["decision"] != "block" {
		t.Fatalf("expected block result, got %#v", result)
	}
	reason, ok := result["reason"].(string)
	if !ok {
		t.Fatalf("expected string reason, got %#v", result["reason"])
	}
	assertContains(t, reason, "Remaining time")
	assertContains(t, reason, "focused-qa")
	assertContains(t, reason, filepath.Join(skillDir, "SKILL.md"))
	assertContains(t, reason, "Capture concrete evidence before you stop.")

	updated := readLoop(t, path)
	if updated.ContinueCount != 1 {
		t.Fatalf("expected continue_count 1, got %d", updated.ContinueCount)
	}
	if updated.Status != StatusActive {
		t.Fatalf("expected active status, got %q", updated.Status)
	}
}

func TestStopMarksTimeLoopCompletedAfterDeadline(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" min="5m"]]`+"\nRun the QA task.", start)
	latest := "Done."

	result, err := HandleStop(paths, StopPayload{
		SessionID:            "sess-1",
		CWD:                  repoRoot,
		LastAssistantMessage: &latest,
	}, start.Add(6*time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %q", updated.Status)
	}
}

func TestStopRoundsModeCompletesTargetRounds(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="3"]]`+"\nRun the QA task.", start)

	firstMessage := "Round one done."
	first, err := HandleStop(paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &firstMessage}, start.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("first stop: %v", err)
	}
	secondMessage := "Round two done."
	second, err := HandleStop(paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &secondMessage}, start.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("second stop: %v", err)
	}
	thirdMessage := "Round three done."
	third, err := HandleStop(paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &thirdMessage}, start.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("third stop: %v", err)
	}

	if first == nil || first["decision"] != "block" {
		t.Fatalf("expected first block, got %#v", first)
	}
	assertContains(t, first["reason"].(string), "Round 2 of 3 begins now.")
	if second == nil || second["decision"] != "block" {
		t.Fatalf("expected second block, got %#v", second)
	}
	assertContains(t, second["reason"].(string), "Round 3 of 3 begins now.")
	if third != nil {
		t.Fatalf("expected third nil, got %#v", third)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %q", updated.Status)
	}
	if updated.CompletedRounds != 3 {
		t.Fatalf("expected completed rounds 3, got %d", updated.CompletedRounds)
	}
}

func TestStopEscalatesOnceThenCutsShortInRoundsMode(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="5"]]`+"\nRun the QA task.", start)
	record := readLoop(t, path)
	record.RapidStopCount = 2
	lastStopAt := record.StartedAt
	record.LastStopAt = &lastStopAt
	if err := ReplaceLoopFile(path, record); err != nil {
		t.Fatalf("replace loop: %v", err)
	}

	escalationMessage := "Stopped again."
	escalation, err := HandleStop(paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &escalationMessage}, start.Add(90*time.Second))
	if err != nil {
		t.Fatalf("escalation stop: %v", err)
	}
	cutShortMessage := "Stopped quickly again."
	cutShort, err := HandleStop(paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &cutShortMessage}, start.Add(150*time.Second))
	if err != nil {
		t.Fatalf("cut-short stop: %v", err)
	}

	if escalation == nil || escalation["decision"] != "block" {
		t.Fatalf("expected escalation block, got %#v", escalation)
	}
	assertContains(t, escalation["reason"].(string), "Broaden the scope materially")
	if cutShort == nil || cutShort["continue"] != false {
		t.Fatalf("expected cut-short warning, got %#v", cutShort)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCutShort {
		t.Fatalf("expected cut_short status, got %q", updated.Status)
	}
	if updated.CompletedRounds != 2 {
		t.Fatalf("expected completed rounds 2, got %d", updated.CompletedRounds)
	}
}

func mustPaths(t *testing.T) Paths {
	t.Helper()
	paths, err := NewPaths(filepath.Join(t.TempDir(), ".codex-home"))
	if err != nil {
		t.Fatalf("new paths: %v", err)
	}
	return paths
}

func fixedTime() time.Time {
	return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
}

func writeRuntimeConfig(t *testing.T, paths Paths, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(paths.RuntimeConfigPath()), 0o755); err != nil {
		t.Fatalf("create runtime config dir: %v", err)
	}
	if err := os.WriteFile(paths.RuntimeConfigPath(), []byte(content), 0o644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}
}

func writeLoop(t *testing.T, paths Paths, sessionID string, repoRoot string, prompt string, now time.Time) string {
	t.Helper()
	activation, ok, err := ExtractActivation(prompt)
	if err != nil {
		t.Fatalf("extract activation: %v", err)
	}
	if !ok {
		t.Fatal("expected activation")
	}
	record := BuildLoopRecord(sessionID, repoRoot, repoRoot, activation, now)
	path := CreateLoopPath(paths, record.Slug, now)
	if err := ReplaceLoopFile(path, record); err != nil {
		t.Fatalf("write loop: %v", err)
	}
	return path
}

func readLoop(t *testing.T, path string) LoopRecord {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read loop: %v", err)
	}
	var record LoopRecord
	if err := json.Unmarshal(content, &record); err != nil {
		t.Fatalf("decode loop: %v", err)
	}
	return record
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}
