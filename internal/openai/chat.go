package openai

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

const chatURL = "wss://api.openai.com/v1/realtime?model=gpt-4o-realtime-preview"

// ConnectChat opens a full speech-to-speech conversation session with the
// given voice (e.g. "sage").
func ConnectChat(apiKey, voice string) (*Session, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+apiKey)
	header.Set("OpenAI-Beta", "realtime=v1")

	conn, resp, err := websocket.DefaultDialer.Dial(chatURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("dial: %w (status %d: %s)", err, resp.StatusCode, body)
		}
		return nil, fmt.Errorf("dial: %w", err)
	}

	s := &Session{conn: conn}
	cfg := map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"modalities":          []string{"audio", "text"},
			"voice":               voice,
			"input_audio_format":  "pcm16",
			"output_audio_format": "pcm16",
			"turn_detection": map[string]any{
				"type":                "server_vad",
				"threshold":           0.6, // higher: ignores speaker bleed/echo
				"prefix_padding_ms":   300,
				"silence_duration_ms": 700, // patient with pauses in accented speech
			},
			"input_audio_noise_reduction": map[string]any{
				"type": "near_field",
			},
			"instructions": "You are a friendly voice assistant. The user may speak accented English; understand them naturally and reply concisely.",
		},
	}
	if err := conn.WriteJSON(cfg); err != nil {
		conn.Close()
		return nil, err
	}
	return s, nil
}
