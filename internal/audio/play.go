package audio

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Player plays raw PCM16 24kHz mono audio (OpenAI Realtime output format).
type Player struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

// NewPlayer starts a pw-cat playback pipe.
func NewPlayer(ctx context.Context) (*Player, error) {
	cmd := exec.CommandContext(ctx, "pw-cat", "--playback",
		"--format", "s16",
		"--rate", "24000",
		"--channels", "1",
		"-",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("pw-cat stdin: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting pw-cat: %w", err)
	}
	return &Player{cmd: cmd, stdin: stdin}, nil
}

// Write plays a chunk of PCM16 audio.
func (p *Player) Write(pcm []byte) (int, error) {
	return p.stdin.Write(pcm)
}

// Close stops playback.
func (p *Player) Close() error {
	p.stdin.Close()
	return p.cmd.Wait()
}
