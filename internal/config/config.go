package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Load reads ~/.config/voxgo/env (KEY=VALUE lines) and returns the merged
// config, with real environment variables taking precedence.
func Load() map[string]string {
	cfg := map[string]string{}

	home, err := os.UserHomeDir()
	if err == nil {
		f, err := os.Open(filepath.Join(home, ".config", "voxgo", "env"))
		if err == nil {
			defer f.Close()
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				k, v, ok := strings.Cut(line, "=")
				if !ok {
					continue
				}
				cfg[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
			}
		}
	}

	for _, key := range []string{"OPENAI_API_KEY", "VOXGO_MODEL", "VOXGO_PROMPT", "VOXGO_SINK", "VOXGO_VOICE"} {
		if v := os.Getenv(key); v != "" {
			cfg[key] = v
		}
	}
	return cfg
}
