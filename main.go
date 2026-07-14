// Command voxgo is accent-friendly voice tooling for Linux on Wayland:
// system-wide dictation that types into the focused window, and a
// speech-to-speech chat assistant ("Irene"), both powered by the OpenAI
// Realtime API over WebSocket.
//
// The binary is self-contained and exposes five entry points: a background
// daemon controlled over a unix socket, a web dashboard, a terminal chat
// mode, and thin client subcommands (toggle/start/stop/status) intended to
// be bound to compositor hotkeys.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mbergo/voxgo/internal/config"
	"github.com/mbergo/voxgo/internal/daemon"
)

const usage = `voxgo — accent-friendly dictation & voice chat for Linux (Wayland)

Usage:
  voxgo daemon           run the dictation daemon (background service)
  voxgo web [addr]       web dashboard (default: 127.0.0.1:7853) —
                         start/stop dictation & chat, pick voice, edit persona
  voxgo chat [voice]     talk with Irene in the terminal (default voice: shimmer)
  voxgo say [-v voice] [text|-]   speak text aloud via OpenAI TTS (stdin with -)
  voxgo toggle           start/stop dictation — bind this to a hotkey
  voxgo start            start dictation
  voxgo stop             stop dictation
  voxgo status           print daemon state (idle | listening)

Configuration (~/.config/voxgo/env, KEY=VALUE; real env vars win):
  OPENAI_API_KEY   required — your OpenAI API key
  VOXGO_SINK       PipeWire sink for chat audio output (see: pactl list sinks short)
  VOXGO_VOICE      default chat voice (shimmer, marin, cedar, alloy, ...)
  VOXGO_PROMPT     override the built-in Irene persona
  VOXGO_DEBUG      set to any value to log raw Realtime API events

Examples:
  voxgo daemon &                 # once per login
  voxgo toggle                   # speak; text is typed into the focused window
  voxgo chat marin               # voice chat using the marin voice
  voxgo web 0.0.0.0:8080         # dashboard reachable from another device
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "daemon":
		runDaemon()
	case "web":
		addr := "127.0.0.1:7853"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		runWeb(addr)
	case "chat":
		voice := "shimmer"
		if len(os.Args) > 2 {
			voice = os.Args[2]
		}
		runChat(voice)
	case "say":
		runSay(os.Args[2:])
	case "start", "stop", "toggle", "status":
		sendCommand(os.Args[1])
	default:
		fmt.Print(usage)
		os.Exit(2)
	}
}

func runDaemon() {
	cfg := config.Export()
	apiKey := cfg["OPENAI_API_KEY"]
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set (env or ~/.config/voxgo/env)")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	if err := daemon.New(apiKey).Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func sendCommand(cmd string) {
	conn, err := net.Dial("unix", daemon.SocketPath())
	if err != nil {
		log.Fatalf("daemon not running? (%v)", err)
	}
	defer conn.Close()
	fmt.Fprintln(conn, cmd)
	buf := make([]byte, 256)
	n, _ := conn.Read(buf)
	fmt.Print(string(buf[:n]))
}
