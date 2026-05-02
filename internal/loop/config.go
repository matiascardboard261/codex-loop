package loop

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const DefaultRuntimeConfig = `# Optional continuation guidance appended to every automatic continuation.
# Leave these blank if you want generic behavior only.
optional_skill_name = ""
optional_skill_path = ""
extra_continuation_guidance = ""

# Managed Codex lifecycle hook settings. Re-run codex-loop install after changing these.
[hooks]
stop_timeout_seconds = 2700

# Goal-loop confirmation settings.
[goal]
confirm_model = "gpt-5.5"
confirm_reasoning_effort = "high"
confirm_command = "codex exec --cd $WORKSPACE_ROOT --ephemeral --yolo --output-last-message $CONFIRM_OUTPUT_PATH $MODEL_ARGV $REASONING_ARGV --skip-git-repo-check -"
timeout_seconds = 2400
interpret_model = "gpt-5.4-mini"
interpret_reasoning_effort = "low"
interpret_timeout_seconds = 120
max_output_bytes = 12000

# Optional command executed inside codex-loop before each automatic continuation.
[pre_loop_continue]
command = ""
cwd = "session_cwd"
timeout_seconds = 60
max_output_bytes = 12000
`

type RuntimeConfig struct {
	OptionalSkillName         string                `toml:"optional_skill_name"`
	OptionalSkillPath         string                `toml:"optional_skill_path"`
	ExtraContinuationGuidance string                `toml:"extra_continuation_guidance"`
	Hooks                     HooksConfig           `toml:"hooks"`
	Goal                      GoalConfig            `toml:"goal"`
	PreLoopContinue           PreLoopContinueConfig `toml:"pre_loop_continue"`
}

type HooksConfig struct {
	StopTimeoutSeconds int `toml:"stop_timeout_seconds"`
}

type GoalConfig struct {
	ConfirmModel             string `toml:"confirm_model"`
	ConfirmReasoningEffort   string `toml:"confirm_reasoning_effort"`
	ConfirmCommand           string `toml:"confirm_command"`
	TimeoutSeconds           int    `toml:"timeout_seconds"`
	InterpretModel           string `toml:"interpret_model"`
	InterpretReasoningEffort string `toml:"interpret_reasoning_effort"`
	InterpretTimeoutSeconds  int    `toml:"interpret_timeout_seconds"`
	MaxOutputBytes           int    `toml:"max_output_bytes"`
}

type PreLoopContinueConfig struct {
	Command        string `toml:"command"`
	CWD            string `toml:"cwd"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
	MaxOutputBytes int    `toml:"max_output_bytes"`
}

type OptionalContinuationConfig struct {
	SkillName     string
	SkillPath     string
	ExtraGuidance string
}

func LoadRuntimeConfig(paths Paths) RuntimeConfig {
	cfg := defaultRuntimeConfig()
	path := paths.RuntimeConfigPath()
	if _, err := os.Stat(path); err != nil {
		return cfg
	}
	if metadata, err := toml.DecodeFile(path, &cfg); err == nil {
		normalized := normalizeRuntimeConfig(cfg)
		if metadata.IsDefined("goal", "confirm_model") && strings.TrimSpace(cfg.Goal.ConfirmModel) == "" {
			normalized.Goal.ConfirmModel = ""
		}
		if metadata.IsDefined("goal", "confirm_reasoning_effort") && strings.TrimSpace(cfg.Goal.ConfirmReasoningEffort) == "" {
			normalized.Goal.ConfirmReasoningEffort = ""
		}
		if metadata.IsDefined("goal", "interpret_model") && strings.TrimSpace(cfg.Goal.InterpretModel) == "" {
			normalized.Goal.InterpretModel = ""
		}
		if metadata.IsDefined("goal", "interpret_reasoning_effort") && strings.TrimSpace(cfg.Goal.InterpretReasoningEffort) == "" {
			normalized.Goal.InterpretReasoningEffort = ""
		}
		return normalized
	}
	return parseRuntimeConfigFallback(path)
}

func defaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Hooks: HooksConfig{
			StopTimeoutSeconds: DefaultStopHookTimeoutSeconds,
		},
		Goal: GoalConfig{
			ConfirmModel:             DefaultGoalConfirmModel,
			ConfirmReasoningEffort:   DefaultGoalConfirmReasoningEffort,
			ConfirmCommand:           DefaultGoalConfirmCommand(),
			TimeoutSeconds:           DefaultGoalTimeoutSeconds,
			InterpretModel:           DefaultGoalInterpretModel,
			InterpretReasoningEffort: DefaultGoalInterpretReasoningEffort,
			InterpretTimeoutSeconds:  DefaultGoalInterpretTimeoutSeconds,
			MaxOutputBytes:           DefaultGoalMaxOutputBytes,
		},
		PreLoopContinue: PreLoopContinueConfig{
			CWD:            PreLoopContinueCWDSession,
			TimeoutSeconds: DefaultPreLoopContinueTimeoutSeconds,
			MaxOutputBytes: DefaultPreLoopContinueMaxOutputBytes,
		},
	}
}

func normalizeRuntimeConfig(cfg RuntimeConfig) RuntimeConfig {
	if cfg.Hooks.StopTimeoutSeconds <= 0 {
		cfg.Hooks.StopTimeoutSeconds = DefaultStopHookTimeoutSeconds
	}
	if strings.TrimSpace(cfg.Goal.ConfirmModel) == "" {
		cfg.Goal.ConfirmModel = DefaultGoalConfirmModel
	}
	if strings.TrimSpace(cfg.Goal.ConfirmReasoningEffort) == "" || !ValidReasoningEffort(cfg.Goal.ConfirmReasoningEffort) {
		cfg.Goal.ConfirmReasoningEffort = DefaultGoalConfirmReasoningEffort
	}
	if strings.TrimSpace(cfg.Goal.ConfirmCommand) == "" {
		cfg.Goal.ConfirmCommand = DefaultGoalConfirmCommand()
	}
	if cfg.Goal.TimeoutSeconds <= 0 {
		cfg.Goal.TimeoutSeconds = DefaultGoalTimeoutSeconds
	}
	if strings.TrimSpace(cfg.Goal.InterpretModel) == "" {
		cfg.Goal.InterpretModel = DefaultGoalInterpretModel
	}
	if strings.TrimSpace(cfg.Goal.InterpretReasoningEffort) == "" || !ValidReasoningEffort(cfg.Goal.InterpretReasoningEffort) {
		cfg.Goal.InterpretReasoningEffort = DefaultGoalInterpretReasoningEffort
	}
	if cfg.Goal.InterpretTimeoutSeconds <= 0 {
		cfg.Goal.InterpretTimeoutSeconds = DefaultGoalInterpretTimeoutSeconds
	}
	maxGoalBudget := cfg.Hooks.StopTimeoutSeconds - GoalHookTimeoutGraceSeconds
	if maxGoalBudget <= 1 {
		cfg.Goal.TimeoutSeconds = 1
		cfg.Goal.InterpretTimeoutSeconds = 1
	} else {
		if cfg.Goal.TimeoutSeconds >= maxGoalBudget {
			cfg.Goal.TimeoutSeconds = maxGoalBudget - 1
		}
		maxInterpretTimeout := maxGoalBudget - cfg.Goal.TimeoutSeconds
		if maxInterpretTimeout < 1 {
			maxInterpretTimeout = 1
		}
		if cfg.Goal.InterpretTimeoutSeconds > maxInterpretTimeout {
			cfg.Goal.InterpretTimeoutSeconds = maxInterpretTimeout
		}
	}
	if cfg.Goal.MaxOutputBytes <= 0 {
		cfg.Goal.MaxOutputBytes = DefaultGoalMaxOutputBytes
	}
	if strings.TrimSpace(cfg.PreLoopContinue.CWD) == "" {
		cfg.PreLoopContinue.CWD = PreLoopContinueCWDSession
	}
	if cfg.PreLoopContinue.TimeoutSeconds <= 0 {
		cfg.PreLoopContinue.TimeoutSeconds = DefaultPreLoopContinueTimeoutSeconds
	}
	if cfg.PreLoopContinue.MaxOutputBytes <= 0 {
		cfg.PreLoopContinue.MaxOutputBytes = DefaultPreLoopContinueMaxOutputBytes
	}
	return cfg
}

func ResolveOptionalContinuationConfig(paths Paths, workspaceRoot string) OptionalContinuationConfig {
	cfg := LoadRuntimeConfig(paths)
	skillName := strings.TrimSpace(cfg.OptionalSkillName)
	skillPathText := strings.TrimSpace(cfg.OptionalSkillPath)
	extraGuidance := strings.TrimSpace(cfg.ExtraContinuationGuidance)

	resolvedSkillPath := ""
	if skillName != "" && skillPathText != "" {
		candidate := filepath.Clean(skillPathText)
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(workspaceRoot, candidate)
		}
		absCandidate, err := filepath.Abs(candidate)
		if err == nil {
			if info, statErr := os.Stat(absCandidate); statErr == nil && info.IsDir() {
				absCandidate = filepath.Join(absCandidate, "SKILL.md")
			}
			if pathIsInside(workspaceRoot, absCandidate) {
				if info, statErr := os.Stat(absCandidate); statErr == nil && !info.IsDir() {
					resolvedSkillPath = absCandidate
				}
			}
		}
		if resolvedSkillPath == "" {
			skillName = ""
		}
	}

	return OptionalContinuationConfig{
		SkillName:     skillName,
		SkillPath:     resolvedSkillPath,
		ExtraGuidance: extraGuidance,
	}
}

func parseRuntimeConfigFallback(path string) RuntimeConfig {
	content, err := os.ReadFile(path)
	if err != nil {
		return defaultRuntimeConfig()
	}
	cfg := defaultRuntimeConfig()
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, rawValue, _ := strings.Cut(line, "=")
		value, err := strconv.Unquote(strings.TrimSpace(rawValue))
		if err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "optional_skill_name":
			cfg.OptionalSkillName = value
		case "optional_skill_path":
			cfg.OptionalSkillPath = value
		case "extra_continuation_guidance":
			cfg.ExtraContinuationGuidance = value
		}
	}
	return normalizeRuntimeConfig(cfg)
}

func pathIsInside(root string, candidate string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absCandidate)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && !filepath.IsAbs(rel))
}
