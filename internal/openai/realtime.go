package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// Connect opens a Realtime transcription session.
func Connect(apiKey string) (*Session, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+apiKey)
	header.Set("OpenAI-Beta", "realtime=v1")

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

// configure sets up the transcription session: model, VAD, language.
func (s *Session) configure() error {
	cfg := map[string]any{
		"type": "transcription_session.update",
		"session": map[string]any{
			"input_audio_format": "pcm16",
			"input_audio_transcription": map[string]any{
				"model":  "gpt-4o-transcribe",
				"prompt": "Transcribe accented English accurately.",
			},
			"turn_detection": map[string]any{
				"type":              "server_vad",
				"threshold":         0.5,
				"silence_duration_ms": 500,
			},
			"input_audio_noise_reduction": map[string]any{
				"type": "near_field",
			},
		},
	}
	return s.conn.WriteJSON(cfg)
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
	var ev Event
	if err := json.Unmarshal(msg, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

// Close shuts down the WebSocket.
func (s *Session) Close() error {
	return s.conn.Close()
}
