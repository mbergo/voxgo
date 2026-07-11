package typer

import (
	"os/exec"
)

// Type injects text into the currently focused window (Wayland, via wtype).
func Type(text string) error {
	return exec.Command("wtype", text).Run()
}
