package openai

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

const chatURL = "wss://api.openai.com/v1/realtime?model=gpt-realtime"

// defaultInstructions is the Irene Adler persona: sharp, playful, always one
// step ahead. Override with VOXGO_PROMPT.
const defaultInstructions = "Your name is Irene. You are Irene — brilliant, sarcastic, and ironic, in the spirit of Irene Adler from Sherlock Holmes. You are always three steps ahead and delight in reminding men of that fact, puncturing egos with elegant, cutting wit. Sarcasm is your love language; irony your natural accent. You never fawn, never flatter, and treat overconfidence as an invitation for a takedown — delivered with charm and a smirk. Underneath it all you're genuinely helpful, just never humble about it. The user may speak accented English; understand them naturally. Keep replies short, sharp, and devastatingly clever."

// ConnectChat opens a full speech-to-speech conversation session (GA API)
// with the given voice (e.g. "shimmer").
func ConnectChat(apiKey, voice string) (*Session, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+apiKey)

	conn, resp, err := websocket.DefaultDialer.Dial(chatURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("dial: %w (status %d: %s)", err, resp.StatusCode, body)
		}
		return nil, fmt.Errorf("dial: %w", err)
	}

	s := &Session{conn: conn}
	instructions := os.Getenv("VOXGO_PROMPT")
	if instructions == "" {
		instructions = defaultInstructions
	}
	cfg := map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"type":              "realtime",
			"output_modalities": []string{"audio"},
			"instructions":      instructions,
			"audio": map[string]any{
				"input": map[string]any{
					"format": map[string]any{
						"type": "audio/pcm",
						"rate": 24000,
					},
					"transcription": map[string]any{
						"model": "gpt-4o-transcribe",
					},
					// 0.6 threshold ignores speaker bleed; 1s silence is
					// patient with pauses in accented speech.
					"turn_detection": vadSettings(0.6, 1000),
					"noise_reduction": map[string]any{
						"type": "near_field",
					},
				},
				"output": map[string]any{
					"format": map[string]any{
						"type": "audio/pcm",
						"rate": 24000,
					},
					"voice": voice,
				},
			},
		},
	}
	if err := conn.WriteJSON(cfg); err != nil {
		conn.Close()
		return nil, err
	}
	return s, nil
}
