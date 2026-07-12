package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateReflectsIdleServer(t *testing.T) {
	s := New("sk-test")
	rec := httptest.NewRecorder()
	s.handleState(rec, httptest.NewRequest(http.MethodGet, "/api/state", nil))

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["dictating"] != false || got["chatting"] != false {
		t.Errorf("fresh server should be idle: %v", got)
	}
	if got["voice"] != "shimmer" {
		t.Errorf("default voice = %v, want shimmer", got["voice"])
	}
}

func TestHandlePromptPersistsToConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".config", "voxgo"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := New("sk-test")
	body := strings.NewReader(`{"prompt":"be sassy"}`)
	rec := httptest.NewRecorder()
	s.handlePrompt(rec, httptest.NewRequest(http.MethodPost, "/api/prompt", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if os.Getenv("VOXGO_PROMPT") != "be sassy" {
		t.Errorf("VOXGO_PROMPT env not updated")
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "voxgo", "env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "VOXGO_PROMPT=be sassy") {
		t.Errorf("config file missing prompt: %q", data)
	}
}

func TestSaveConfigValueUpserts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "voxgo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "env")
	if err := os.WriteFile(path, []byte("OPENAI_API_KEY=sk-keep\nVOXGO_PROMPT=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := saveConfigValue("VOXGO_PROMPT", "new value"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	if !strings.Contains(text, "OPENAI_API_KEY=sk-keep") {
		t.Errorf("existing key lost: %q", text)
	}
	if !strings.Contains(text, "VOXGO_PROMPT=new value") || strings.Contains(text, "VOXGO_PROMPT=old") {
		t.Errorf("prompt not replaced: %q", text)
	}
}

func TestSaveConfigValueFlattensNewlines(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".config", "voxgo"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := saveConfigValue("VOXGO_PROMPT", "line one\nline two"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".config", "voxgo", "env"))
	if strings.Contains(string(data), "line one\nline two") {
		t.Errorf("newlines must be flattened for the single-line format: %q", data)
	}
	if !strings.Contains(string(data), "VOXGO_PROMPT=line one line two") {
		t.Errorf("flattened value missing: %q", data)
	}
}

func TestBroadcastReachesSubscribers(t *testing.T) {
	s := New("sk-test")
	ch := make(chan transcriptEvent, 1)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()

	s.broadcast(transcriptEvent{Kind: "user", Text: "hi"})
	select {
	case ev := <-ch:
		if ev.Text != "hi" {
			t.Errorf("got %+v", ev)
		}
	default:
		t.Error("subscriber did not receive event")
	}
}

func TestBroadcastSkipsFullSubscribers(t *testing.T) {
	s := New("sk-test")
	full := make(chan transcriptEvent) // unbuffered, nobody reading
	s.mu.Lock()
	s.subs[full] = struct{}{}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.broadcast(transcriptEvent{Kind: "user", Text: "hi"}) // must not block
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broadcast blocked on a slow subscriber")
	}
}
