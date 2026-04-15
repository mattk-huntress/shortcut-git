package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RepoConfig struct {
	ObjectiveID          int               `json:"objective_id"`
	WorkspaceSlug        string            `json:"workspace_slug"`
	APIBase              string            `json:"shortcut_api_base"`
	WorkflowStates       map[string]string `json:"workflow_states"`
	Members              map[string]string `json:"members"`
	Groups               map[string]string `json:"groups"`
	DefaultWorkflowState int               `json:"default_workflow_state_id,omitempty"`
	TeamID               string            `json:"team_id,omitempty"`
	LastFetchAt          *time.Time        `json:"last_fetch_at,omitempty"`
}

func configDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".shortcut-git")
}

func loadRepoConfig(repoRoot string) (*RepoConfig, error) {
	data, err := os.ReadFile(filepath.Join(configDir(repoRoot), "config.json"))
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg RepoConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func saveRepoConfig(repoRoot string, cfg *RepoConfig) error {
	dir := configDir(repoRoot)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}

// parseDotEnv reads a .env file and returns a map of key=value pairs.
// Lines beginning with # and blank lines are ignored. Inline comments are not
// supported. Values may be optionally quoted with single or double quotes.
func parseDotEnv(data []byte) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		result[key] = val
	}
	return result
}

func loadAPIToken() (string, error) {
	if token := os.Getenv("SHORTCUT_API_TOKEN"); token != "" {
		return token, nil
	}

	// Check .env next to the binary first, then fall back to cwd.
	if exe, err := os.Executable(); err == nil {
		if data, err := os.ReadFile(filepath.Join(filepath.Dir(exe), ".env")); err == nil {
			if token := parseDotEnv(data)["SHORTCUT_API_TOKEN"]; token != "" {
				return token, nil
			}
		}
	}

	// Check .env file in the current working directory.
	if data, err := os.ReadFile(".env"); err == nil {
		if token := parseDotEnv(data)["SHORTCUT_API_TOKEN"]; token != "" {
			return token, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return "", fmt.Errorf("SHORTCUT_API_TOKEN not set and reading ~/.claude.json failed: %w", err)
	}
	var cfg struct {
		MCPServers struct {
			Shortcut struct {
				Env struct {
					Token string `json:"SHORTCUT_API_TOKEN"`
				} `json:"env"`
			} `json:"shortcut"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parsing ~/.claude.json: %w", err)
	}
	token := cfg.MCPServers.Shortcut.Env.Token
	if token == "" {
		return "", fmt.Errorf("SHORTCUT_API_TOKEN not found in environment, .env file, or ~/.claude.json at mcpServers.shortcut.env.SHORTCUT_API_TOKEN")
	}
	return token, nil
}

// findDefaultWorkflowState returns the configured default or falls back to
// the lowest numeric state ID (B20: deterministic, not random map order).
func findDefaultWorkflowState(cfg *RepoConfig) int {
	if cfg.DefaultWorkflowState != 0 {
		return cfg.DefaultWorkflowState
	}
	lowest := 0
	for idStr := range cfg.WorkflowStates {
		id := 0
		fmt.Sscanf(idStr, "%d", &id)
		if id != 0 && (lowest == 0 || id < lowest) {
			lowest = id
		}
	}
	return lowest
}
