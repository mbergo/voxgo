package audio

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Capture starts recording from the default mic via pw-record and returns
// a reader of raw PCM16 little-endian, 16kHz, mono — the format the
// OpenAI Realtime API expects.
func Capture(ctx context.Context) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "pw-record",
		"--format", "s16",
		"--rate", "16000",
		"--channels", "1",
		"-", // write to stdout
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pw-record stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting pw-record: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = cmd.Wait()
	}()
	return stdout, nil
}
