package audio

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Player plays raw PCM16 24kHz mono audio (OpenAI Realtime output format).
type Player struct {
	ctx   context.Context
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

// NewPlayer starts a pw-cat playback pipe. VOXGO_SINK selects the output
// device; otherwise the system default sink is used.
func NewPlayer(ctx context.Context) (*Player, error) {
	cmd, stdin, err := startPipe(ctx)
	if err != nil {
		return nil, err
	}
	return &Player{ctx: ctx, cmd: cmd, stdin: stdin}, nil
}

func startPipe(ctx context.Context) (*exec.Cmd, io.WriteCloser, error) {
	cmd := exec.CommandContext(ctx, "pw-cat", "--playback", "--raw",
		"--format", "s16",
		"--rate", "24000",
		"--channels", "1",
		"-",
	)
	if sink := os.Getenv("VOXGO_SINK"); sink != "" {
		cmd.Env = append(os.Environ(), "PIPEWIRE_NODE="+sink)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pw-cat stdin: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("starting pw-cat: %w", err)
	}
	return cmd, stdin, nil
}

// Write plays a chunk of PCM16 audio.
func (p *Player) Write(pcm []byte) (int, error) {
	return p.stdin.Write(pcm)
}

// Flush drops all queued audio by restarting the playback pipe. Used for
// barge-in: when the user talks over the assistant, buffered speech is cut
// off immediately instead of playing to the end.
func (p *Player) Flush() error {
	p.stdin.Close()
	_ = p.cmd.Process.Kill()
	_ = p.cmd.Wait()
	cmd, stdin, err := startPipe(p.ctx)
	if err != nil {
		return err
	}
	p.cmd, p.stdin = cmd, stdin
	return nil
}

// Close stops playback.
func (p *Player) Close() error {
	p.stdin.Close()
	return p.cmd.Wait()
}
