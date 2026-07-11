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

const usage = `voxgo — accent-friendly dictation for Linux (Wayland)

Usage:
  voxgo daemon    run the background daemon
  voxgo chat      voice conversation mode (speaks back, default voice: shimmer)
  voxgo toggle    start/stop listening (bind this to a hotkey)
  voxgo start     start listening
  voxgo stop      stop listening
  voxgo status    show daemon state
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "daemon":
		runDaemon()
	case "chat":
		voice := "shimmer"
		if len(os.Args) > 2 {
			voice = os.Args[2]
		}
		runChat(voice)
	case "start", "stop", "toggle", "status":
		sendCommand(os.Args[1])
	default:
		fmt.Print(usage)
		os.Exit(2)
	}
}

func runDaemon() {
	cfg := config.Load()
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
