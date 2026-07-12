package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mbergo/voxgo/internal/config"
	"github.com/mbergo/voxgo/internal/web"
)

// runWeb serves the control dashboard.
func runWeb(addr string) {
	cfg := config.Load()
	apiKey := cfg["OPENAI_API_KEY"]
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set (env or ~/.config/voxgo/env)")
	}
	if sink := cfg["VOXGO_SINK"]; sink != "" {
		os.Setenv("VOXGO_SINK", sink)
	}
	if p := cfg["VOXGO_PROMPT"]; p != "" {
		os.Setenv("VOXGO_PROMPT", p)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	if err := web.New(apiKey).ListenAndServe(ctx, addr); err != nil {
		log.Fatal(err)
	}
}
