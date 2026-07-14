// Package chat runs a full speech-to-speech conversation: microphone audio
// is streamed to an OpenAI Realtime session and the assistant's spoken reply
// is played through the configured PipeWire sink, while both sides'
// transcripts are surfaced to the caller as events. It is shared by the
// `voxgo chat` CLI and the web dashboard.
package chat

import (
	"context"
	"encoding/base64"
	"errors"
	"os"

	"github.com/mbergo/voxgo/internal/audio"
	"github.com/mbergo/voxgo/internal/openai"
)

// Event is a conversation event surfaced to the caller.
type Event struct {
	Kind string // "user" | "assistant"
	Text string
}

// Run drives one speech-to-speech conversation until ctx is cancelled.
// Events (user and assistant transcripts) are delivered to onEvent.
func Run(ctx context.Context, apiKey, voice string, onEvent func(Event)) error {
	sess, err := openai.ConnectChat(apiKey, voice)
	if err != nil {
		return err
	}
	defer sess.Close()

	mic, err := audio.CaptureRate(ctx, 24000)
	if err != nil {
		return err
	}
	defer mic.Close()

	speaker, err := audio.NewPlayer(ctx)
	if err != nil {
		return err
	}
	defer speaker.Close()

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

	// Barge-in: unless VOXGO_INTERRUPT=0, talking over the assistant cancels
	// its in-flight reply and flushes queued audio so it shuts up right away.
	interrupt := os.Getenv("VOXGO_INTERRUPT") != "0"
	responding := false

	for {
		ev, err := sess.Read()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		switch ev.Type {
		case "input_audio_buffer.speech_started":
			if interrupt && responding {
				_ = sess.CancelResponse()
				_ = speaker.Flush()
				responding = false
			}
		case "response.created":
			responding = true
		case "response.done":
			responding = false
		case "response.output_audio.delta", "response.audio.delta":
			pcm, err := base64.StdEncoding.DecodeString(ev.Delta)
			if err == nil {
				_, _ = speaker.Write(pcm)
			}
		case "response.output_audio_transcript.done", "response.audio_transcript.done":
			if ev.Transcript != "" {
				onEvent(Event{Kind: "assistant", Text: ev.Transcript})
			}
		case "conversation.item.input_audio_transcription.completed":
			if ev.Transcript != "" {
				onEvent(Event{Kind: "user", Text: ev.Transcript})
			}
		case "error":
			if ev.Error != nil {
				return errors.New(ev.Error.Message)
			}
		}
	}
}
