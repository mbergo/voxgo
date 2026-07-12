package dictate

import (
	"context"
	"errors"

	"github.com/mbergo/voxgo/internal/audio"
	"github.com/mbergo/voxgo/internal/openai"
	"github.com/mbergo/voxgo/internal/typer"
)

// Run performs one dictation session: mic → OpenAI transcription → keyboard.
// Each completed transcript is also delivered to onText (may be nil).
func Run(ctx context.Context, apiKey string, onText func(string)) error {
	sess, err := openai.Connect(apiKey)
	if err != nil {
		return err
	}
	defer sess.Close()

	mic, err := audio.CaptureRate(ctx, 24000)
	if err != nil {
		return err
	}
	defer mic.Close()

	go func() {
		buf := make([]byte, 4800) // 100ms @ 24kHz
		for {
			n, err := mic.Read(buf)
			if n > 0 {
				if err := sess.SendAudio(buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		<-ctx.Done()
		sess.Close()
	}()

	for {
		ev, err := sess.Read()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		switch ev.Type {
		case "conversation.item.input_audio_transcription.completed":
			if ev.Transcript == "" {
				continue
			}
			if onText != nil {
				onText(ev.Transcript)
			}
			_ = typer.Type(ev.Transcript + " ")
		case "error":
			if ev.Error != nil {
				return errors.New(ev.Error.Message)
			}
		}
	}
}
