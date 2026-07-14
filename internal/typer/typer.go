// Package typer injects text into the currently focused window. It shells
// out to wtype, which speaks the Wayland virtual-keyboard protocol, so
// dictation works in any application — terminals, browsers, editors —
// exactly as if the text had been typed on a physical keyboard.
package typer

import (
	"os/exec"
)

// Type injects text into the currently focused window (Wayland, via wtype).
func Type(text string) error {
	return exec.Command("wtype", text).Run()
}

// PressEnter sends a Return keypress to the focused window, used by the
// auto-send dictation mode (VOXGO_ENTER) to submit chat messages.
func PressEnter() error {
	return exec.Command("wtype", "-k", "Return").Run()
}
