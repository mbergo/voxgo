package web

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mbergo/voxgo/internal/chat"
	"github.com/mbergo/voxgo/internal/dictate"
)

//go:embed index.html
var content embed.FS

type transcriptEvent struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Server is the voxgo web dashboard.
type Server struct {
	apiKey string

	mu         sync.Mutex
	dictCancel context.CancelFunc
	chatCancel context.CancelFunc
	voice      string
	subs       map[chan transcriptEvent]struct{}
}

// New creates a dashboard server.
func New(apiKey string) *Server {
	return &Server{
		apiKey: apiKey,
		voice:  "shimmer",
		subs:   map[chan transcriptEvent]struct{}{},
	}
}

// ListenAndServe runs the dashboard on addr until ctx is done.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(content))
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("POST /api/dictation/toggle", s.handleDictToggle)
	mux.HandleFunc("POST /api/chat/toggle", s.handleChatToggle)
	mux.HandleFunc("POST /api/prompt", s.handlePrompt)
	mux.HandleFunc("GET /api/events", s.handleEvents)

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	log.Printf("voxgo dashboard on http://%s", addr)
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) broadcast(ev transcriptEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (s *Server) state() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"dictating": s.dictCancel != nil,
		"chatting":  s.chatCancel != nil,
		"voice":     s.voice,
		"prompt":    os.Getenv("VOXGO_PROMPT"),
	}
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.state())
}

func (s *Server) handleDictToggle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.dictCancel != nil {
		s.dictCancel()
		s.dictCancel = nil
		s.mu.Unlock()
		writeJSON(w, s.state())
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.dictCancel = cancel
	s.mu.Unlock()

	go func() {
		err := dictate.Run(ctx, s.apiKey, func(text string) {
			s.broadcast(transcriptEvent{Kind: "dictation", Text: text})
		})
		if err != nil {
			log.Printf("dictation: %v", err)
		}
		s.mu.Lock()
		if s.dictCancel != nil {
			s.dictCancel()
			s.dictCancel = nil
		}
		s.mu.Unlock()
	}()
	writeJSON(w, s.state())
}

func (s *Server) handleChatToggle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Voice string `json:"voice"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	if s.chatCancel != nil {
		s.chatCancel()
		s.chatCancel = nil
		s.mu.Unlock()
		writeJSON(w, s.state())
		return
	}
	if req.Voice != "" {
		s.voice = req.Voice
	}
	voice := s.voice
	ctx, cancel := context.WithCancel(context.Background())
	s.chatCancel = cancel
	s.mu.Unlock()

	go func() {
		err := chat.Run(ctx, s.apiKey, voice, func(ev chat.Event) {
			s.broadcast(transcriptEvent{Kind: ev.Kind, Text: ev.Text})
		})
		if err != nil {
			log.Printf("chat: %v", err)
			s.broadcast(transcriptEvent{Kind: "dictation", Text: "chat error: " + err.Error()})
		}
		s.mu.Lock()
		if s.chatCancel != nil {
			s.chatCancel()
			s.chatCancel = nil
		}
		s.mu.Unlock()
	}()
	writeJSON(w, s.state())
}

func (s *Server) handlePrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	os.Setenv("VOXGO_PROMPT", req.Prompt)
	if err := saveConfigValue("VOXGO_PROMPT", req.Prompt); err != nil {
		log.Printf("saving prompt: %v", err)
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	ch := make(chan transcriptEvent, 16)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			fl.Flush()
		}
	}
}

// saveConfigValue upserts KEY=VALUE in ~/.config/voxgo/env, single-line value.
func saveConfigValue(key, value string) error {
	value = strings.ReplaceAll(value, "\n", " ")
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".config", "voxgo", "env")

	var lines []string
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
				continue
			}
			lines = append(lines, line)
		}
		f.Close()
	}
	lines = append(lines, key+"="+value)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
