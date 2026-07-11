# voxgo 🎙

Accent-friendly, system-wide dictation for Linux (Wayland), powered by the
OpenAI Realtime API over WebSocket. Speak naturally — in *your* accent — and
accurate text is typed into whatever window has focus.

## Why

Local speech models keep mishearing accented English. `gpt-4o-transcribe`
doesn't. voxgo streams your mic straight to it and types the result, anywhere.

## How it works

```
pw-record (mic, PCM16 16kHz) ──► WebSocket ──► OpenAI Realtime (server VAD)
                                                        │
   focused window ◄── wtype ◄── transcript deltas ◄─────┘
```

- **Daemon + hotkey control** — `voxgo daemon` runs in the background;
  `voxgo toggle` flips listening on/off via a unix socket, so you can bind it
  to any compositor keybinding.
- **Auto-reconnect** with exponential backoff if the WebSocket drops.
- **Desktop notifications** on state changes.
- Single static Go binary. No Python, no FastAPI, no fuss.

## Install

```bash
sudo apt install wtype pipewire-utils   # runtime deps
go install github.com/mbergo/voxgo@latest
```

## Setup

Put your key in `~/.config/voxgo/env`:

```
OPENAI_API_KEY=sk-...
```

## Usage

```bash
voxgo daemon &        # once, e.g. in your session autostart
voxgo toggle          # start talking; run again to stop
voxgo status
```

### Hotkey (Hyprland example)

```
bind = SUPER, D, exec, voxgo toggle
```

### Hotkey (Sway example)

```
bindsym $mod+d exec voxgo toggle
```

## License

MIT
