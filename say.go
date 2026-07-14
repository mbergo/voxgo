package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/mbergo/voxgo/internal/audio"
	"github.com/mbergo/voxgo/internal/config"
)

// runSay converts text to speech via the OpenAI TTS API and plays it through
// the configured sink. Text comes from the CLI args, or stdin when args are
// empty or "-". This gives any program — or an AI assistant driving a shell —
// a voice.
func runSay(args []string) {
	cfg := config.Load()
	apiKey := cfg["OPENAI_API_KEY"]
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set (env or ~/.config/voxgo/env)")
	}
	if sink := cfg["VOXGO_SINK"]; sink != "" {
		os.Setenv("VOXGO_SINK", sink)
	}
	voice := cfg["VOXGO_SAY_VOICE"]
	if voice == "" {
		voice = "marin" // female, distinct from Irene's shimmer
	}
	if len(args) >= 2 && args[0] == "-v" {
		voice = args[1]
		args = args[2:]
	}

	var text string
	if len(args) == 0 || (len(args) == 1 && args[0] == "-") {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		text = string(data)
	} else {
		for i, a := range args {
			if i > 0 {
				text += " "
			}
			text += a
		}
	}
	if text == "" {
		log.Fatal("nothing to say")
	}

	body, _ := json.Marshal(map[string]any{
		"model":           "gpt-4o-mini-tts",
		"voice":           voice,
		"input":           text,
		"response_format": "pcm", // 24kHz s16le mono — pw-cat's native diet
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/audio/speech", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		log.Fatalf("TTS API: %s: %s", resp.Status, msg)
	}

	player, err := audio.NewPlayer(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(player, resp.Body); err != nil {
		log.Fatal(err)
	}
	if err := player.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "playback:", err)
	}
}
