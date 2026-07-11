package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mbergo/voxgo/internal/audio"
	"github.com/mbergo/voxgo/internal/config"
	"github.com/mbergo/voxgo/internal/openai"
)

// runChat is a full speech-to-speech conversation using the system default
// mic and speakers.
func runChat(voice string) {
	cfg := config.Load()
	apiKey := cfg["OPENAI_API_KEY"]
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set (env or ~/.config/voxgo/env)")
	}
	if sink := cfg["VOXGO_SINK"]; sink != "" {
		os.Setenv("VOXGO_SINK", sink)
	}
	if p := cfg["VOXGO_PROMPT"]; p != "" {
		os.Setenv("VOXGO_PROMPT", p)
	}
	if v := cfg["VOXGO_VOICE"]; v != "" && voice == "shimmer" {
		voice = v
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nbye 👋")
		cancel()
	}()

	sess, err := openai.ConnectChat(apiKey, voice)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer sess.Close()

	mic, err := audio.CaptureRate(ctx, 24000)
	if err != nil {
		log.Fatalf("mic: %v", err)
	}
	defer mic.Close()

	speaker, err := audio.NewPlayer(ctx)
	if err != nil {
		log.Fatalf("speaker: %v", err)
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

	fmt.Printf("🗣  voxgo chat — voice %q, speak naturally. Ctrl-C to quit.\n", voice)

	for {
		ev, err := sess.Read()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Fatalf("read: %v", err)
		}
		switch ev.Type {
		case "response.output_audio.delta", "response.audio.delta":
			pcm, err := base64.StdEncoding.DecodeString(ev.Delta)
			if err == nil {
				_, _ = speaker.Write(pcm)
			}
		case "response.output_audio_transcript.done", "response.audio_transcript.done":
			if ev.Transcript != "" {
				fmt.Printf("🤖 %s\n", ev.Transcript)
			}
		case "conversation.item.input_audio_transcription.completed":
			if ev.Transcript != "" {
				fmt.Printf("🧑 %s\n", ev.Transcript)
			}
		case "error":
			if ev.Error != nil {
				log.Printf("api error: %s", ev.Error.Message)
			}
		}
	}
}
