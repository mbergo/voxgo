// Package config loads voxgo settings from ~/.config/voxgo/env, a simple
// KEY=VALUE file (comments with #, optional single or double quotes around
// values). Real environment variables always take precedence over the file,
// so one-off overrides like `VOXGO_VOICE=marin voxgo chat` work as expected.
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

	for _, key := range []string{
		"OPENAI_API_KEY", "VOXGO_MODEL", "VOXGO_PROMPT", "VOXGO_SINK",
		"VOXGO_VOICE", "VOXGO_SAY_VOICE", "VOXGO_ENTER", "VOXGO_INTERRUPT",
		"VOXGO_VAD_SILENCE_MS", "VOXGO_VAD_THRESHOLD", "VOXGO_DEBUG",
	} {
		if v := os.Getenv(key); v != "" {
			cfg[key] = v
		}
	}
	return cfg
}

// Export loads the config and copies every value into the process
// environment (real env vars still win, since Load prefers them). Commands
// call this once at startup so internal packages can simply read os.Getenv
// without threading the config map everywhere.
func Export() map[string]string {
	cfg := Load()
	for k, v := range cfg {
		os.Setenv(k, v)
	}
	return cfg
}
