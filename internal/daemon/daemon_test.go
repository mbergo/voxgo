package daemon

import (
	"strings"
	"testing"
)

func TestSocketPathUsesXDGRuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1234")
	if got := SocketPath(); got != "/run/user/1234/voxgo.sock" {
		t.Errorf("SocketPath() = %q", got)
	}
}

func TestSocketPathFallsBackToTmp(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	got := SocketPath()
	if !strings.HasSuffix(got, "/voxgo.sock") {
		t.Errorf("SocketPath() = %q, want */voxgo.sock", got)
	}
}
