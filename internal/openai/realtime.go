// Package openai is a minimal WebSocket client for the OpenAI Realtime API
// (GA protocol). It supports two session shapes: transcription-only
// (Connect, used for dictation) and full speech-to-speech conversation
// (ConnectChat, used for the Irene assistant). Only the handful of event
// fields voxgo consumes are modeled; everything else is ignored.
//
// Set VOXGO_DEBUG to any non-empty value to log every raw server event,
// which is invaluable when OpenAI evolves the protocol.
package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

const realtimeURL = "wss://api.openai.com/v1/realtime?intent=transcription"

// Event is a minimal Realtime API server event.
type Event struct {
	Type       string `json:"type"`
	Delta      string `json:"delta,omitempty"`
	Transcript string `json:"transcript,omitempty"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Session is a live transcription session over WebSocket.
type Session struct {
	conn *websocket.Conn
}

// Connect opens a Realtime transcription session (GA API).
func Connect(apiKey string) (*Session, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+apiKey)

	conn, resp, err := websocket.DefaultDialer.Dial(realtimeURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("dial: %w (status %d: %s)", err, resp.StatusCode, body)
		}
		return nil, fmt.Errorf("dial: %w", err)
	}

	s := &Session{conn: conn}
	if err := s.configure(); err != nil {
		conn.Close()
		return nil, err
	}
	return s, nil
}

// configure sets up the GA transcription session: model, VAD, noise reduction.
func (s *Session) configure() error {
	cfg := map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"type": "transcription",
			"audio": map[string]any{
				"input": map[string]any{
					"format": map[string]any{
						"type": "audio/pcm",
						"rate": 24000,
					},
					"transcription": map[string]any{
						"model":  "gpt-4o-transcribe",
						"prompt": "Transcribe accented English accurately.",
					},
					// 1s default: room to breathe mid-sentence when
					// English isn't your first language.
					"turn_detection": vadSettings(0.5, 1000),
					"noise_reduction": map[string]any{
						"type": "near_field",
					},
				},
			},
		},
	}
	return s.conn.WriteJSON(cfg)
}

// CancelResponse asks the server to stop generating the in-flight response.
// Used for barge-in when the user talks over the assistant.
func (s *Session) CancelResponse() error {
	return s.conn.WriteJSON(map[string]any{"type": "response.cancel"})
}

// SendAudio streams a chunk of PCM16 audio to the session.
func (s *Session) SendAudio(pcm []byte) error {
	return s.conn.WriteJSON(map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(pcm),
	})
}

// Read returns the next server event.
func (s *Session) Read() (*Event, error) {
	_, msg, err := s.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if os.Getenv("VOXGO_DEBUG") != "" {
		log.Printf("[debug] %s", truncate(string(msg), 300))
	}
	var ev Event
	if err := json.Unmarshal(msg, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// Close shuts down the WebSocket.
func (s *Session) Close() error {
	return s.conn.Close()
}
