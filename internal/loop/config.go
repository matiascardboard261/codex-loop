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
`

type RuntimeConfig struct {
	OptionalSkillName         string `toml:"optional_skill_name"`
	OptionalSkillPath         string `toml:"optional_skill_path"`
	ExtraContinuationGuidance string `toml:"extra_continuation_guidance"`
}

type OptionalContinuationConfig struct {
	SkillName     string
	SkillPath     string
	ExtraGuidance string
}

func LoadRuntimeConfig(paths Paths) RuntimeConfig {
	cfg := RuntimeConfig{}
	path := paths.RuntimeConfigPath()
	if _, err := os.Stat(path); err != nil {
		return cfg
	}
	if _, err := toml.DecodeFile(path, &cfg); err == nil {
		return cfg
	}
	return parseRuntimeConfigFallback(path)
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
		return RuntimeConfig{}
	}
	cfg := RuntimeConfig{}
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
	return cfg
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
