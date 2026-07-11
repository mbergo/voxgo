package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mbergo/voxgo/internal/audio"
	"github.com/mbergo/voxgo/internal/openai"
	"github.com/mbergo/voxgo/internal/typer"
)

// SocketPath returns the control socket path.
func SocketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "voxgo.sock")
}

// Daemon owns the dictation state machine.
type Daemon struct {
	apiKey string

	mu        sync.Mutex
	listening bool
	cancel    context.CancelFunc
}

// New creates a daemon.
func New(apiKey string) *Daemon {
	return &Daemon{apiKey: apiKey}
}

// Run serves the control socket until ctx is done.
func (d *Daemon) Run(ctx context.Context) error {
	sock := SocketPath()
	_ = os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("listen %s: %w", sock, err)
	}
	defer ln.Close()
	defer os.Remove(sock)

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	log.Printf("voxgo daemon ready — control socket %s", sock)
	notify("voxgo", "daemon ready")

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go d.handle(ctx, conn)
	}
}

func (d *Daemon) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	cmd := strings.TrimSpace(string(buf[:n]))

	var reply string
	switch cmd {
	case "start":
		reply = d.start(ctx)
	case "stop":
		reply = d.stop()
	case "toggle":
		d.mu.Lock()
		listening := d.listening
		d.mu.Unlock()
		if listening {
			reply = d.stop()
		} else {
			reply = d.start(ctx)
		}
	case "status":
		d.mu.Lock()
		if d.listening {
			reply = "listening"
		} else {
			reply = "idle"
		}
		d.mu.Unlock()
	default:
		reply = "unknown command: " + cmd
	}
	fmt.Fprintln(conn, reply)
}

func (d *Daemon) start(parent context.Context) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.listening {
		return "already listening"
	}
	ctx, cancel := context.WithCancel(parent)
	d.cancel = cancel
	d.listening = true
	go d.dictate(ctx)
	notify("voxgo 🎙", "listening")
	return "listening"
}

func (d *Daemon) stop() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.listening {
		return "already idle"
	}
	d.cancel()
	d.listening = false
	notify("voxgo", "stopped")
	return "stopped"
}

// dictate runs one listening episode: mic → OpenAI → keyboard,
// reconnecting with backoff until ctx is cancelled.
func (d *Daemon) dictate(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		err := d.session(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("session error: %v — reconnecting in %s", err, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}
}

func (d *Daemon) session(ctx context.Context) error {
	sess, err := openai.Connect(d.apiKey)
	if err != nil {
		return err
	}
	defer sess.Close()

	mic, err := audio.CaptureRate(ctx, 24000)
	if err != nil {
		return err
	}
	defer mic.Close()

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

	for {
		ev, err := sess.Read()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		switch ev.Type {
		case "conversation.item.input_audio_transcription.completed":
			if ev.Transcript == "" {
				continue
			}
			log.Printf("» %s", ev.Transcript)
			if err := typer.Type(ev.Transcript + " "); err != nil {
				log.Printf("wtype: %v", err)
			}
		case "error":
			if ev.Error != nil {
				return errors.New(ev.Error.Message)
			}
		}
	}
}

func notify(title, body string) {
	_ = exec.Command("notify-send", "-a", "voxgo", "-t", "1500", title, body).Run()
}
