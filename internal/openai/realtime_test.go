package openai

import (
	"encoding/json"
	"testing"
)

func TestEventUnmarshalTranscript(t *testing.T) {
	raw := `{"type":"conversation.item.input_audio_transcription.completed","transcript":"hello there"}`
	var ev Event
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Type != "conversation.item.input_audio_transcription.completed" {
		t.Errorf("Type = %q", ev.Type)
	}
	if ev.Transcript != "hello there" {
		t.Errorf("Transcript = %q, want %q", ev.Transcript, "hello there")
	}
}

func TestEventUnmarshalError(t *testing.T) {
	raw := `{"type":"error","error":{"message":"boom"}}`
	var ev Event
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Error == nil || ev.Error.Message != "boom" {
		t.Errorf("Error = %+v, want message boom", ev.Error)
	}
}

func TestEventUnmarshalAudioDelta(t *testing.T) {
	raw := `{"type":"response.output_audio.delta","delta":"AAAA"}`
	var ev Event
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Delta != "AAAA" {
		t.Errorf("Delta = %q, want AAAA", ev.Delta)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("short string changed: %q", got)
	}
	if got := truncate("hello world", 5); got != "hello…" {
		t.Errorf("truncate = %q, want %q", got, "hello…")
	}
}
