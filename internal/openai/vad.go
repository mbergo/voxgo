package openai

import (
	"os"
	"strconv"
)

// vadSettings builds the server VAD turn-detection config. Defaults differ
// per mode (dictation is snappier than chat), but both can be overridden:
//
//	VOXGO_VAD_THRESHOLD   0.0-1.0, higher = needs louder speech to trigger
//	VOXGO_VAD_SILENCE_MS  how long you can pause before the turn ends
func vadSettings(defThreshold float64, defSilenceMS int) map[string]any {
	threshold := defThreshold
	if v := os.Getenv("VOXGO_VAD_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 1 {
			threshold = f
		}
	}
	silence := defSilenceMS
	if v := os.Getenv("VOXGO_VAD_SILENCE_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			silence = n
		}
	}
	return map[string]any{
		"type":                "server_vad",
		"threshold":           threshold,
		"prefix_padding_ms":   300,
		"silence_duration_ms": silence,
	}
}
