package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withTempHome points $HOME at a temp dir so Load reads a scratch config.
func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeConfig(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "voxgo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadParsesKeyValueLines(t *testing.T) {
	home := withTempHome(t)
	writeConfig(t, home, "OPENAI_API_KEY=sk-test\nVOXGO_SINK=my-sink\n")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("VOXGO_SINK", "")

	cfg := Load()
	if cfg["OPENAI_API_KEY"] != "sk-test" {
		t.Errorf("OPENAI_API_KEY = %q, want sk-test", cfg["OPENAI_API_KEY"])
	}
	if cfg["VOXGO_SINK"] != "my-sink" {
		t.Errorf("VOXGO_SINK = %q, want my-sink", cfg["VOXGO_SINK"])
	}
}

func TestLoadSkipsCommentsAndBlanks(t *testing.T) {
	home := withTempHome(t)
	writeConfig(t, home, "# a comment\n\nOPENAI_API_KEY=sk-test\nnot-a-pair\n")

	cfg := Load()
	if len(cfg) != 1 {
		t.Errorf("expected 1 key, got %d: %v", len(cfg), cfg)
	}
}

func TestLoadStripsQuotes(t *testing.T) {
	home := withTempHome(t)
	writeConfig(t, home, "VOXGO_PROMPT=\"be nice\"\nVOXGO_MODEL='gpt-realtime'\n")

	cfg := Load()
	if cfg["VOXGO_PROMPT"] != "be nice" {
		t.Errorf("VOXGO_PROMPT = %q, want unquoted", cfg["VOXGO_PROMPT"])
	}
	if cfg["VOXGO_MODEL"] != "gpt-realtime" {
		t.Errorf("VOXGO_MODEL = %q, want unquoted", cfg["VOXGO_MODEL"])
	}
}

func TestEnvOverridesFile(t *testing.T) {
	home := withTempHome(t)
	writeConfig(t, home, "OPENAI_API_KEY=sk-from-file\n")
	t.Setenv("OPENAI_API_KEY", "sk-from-env")

	cfg := Load()
	if cfg["OPENAI_API_KEY"] != "sk-from-env" {
		t.Errorf("env should win, got %q", cfg["OPENAI_API_KEY"])
	}
}

func TestLoadMissingFile(t *testing.T) {
	withTempHome(t)
	t.Setenv("OPENAI_API_KEY", "")

	cfg := Load() // must not panic or error
	if v := cfg["OPENAI_API_KEY"]; v != "" {
		t.Errorf("expected empty config, got OPENAI_API_KEY=%q", v)
	}
}
