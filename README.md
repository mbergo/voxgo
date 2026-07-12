# voxgo 🎙

[![CI](https://github.com/mbergo/voxgo/actions/workflows/ci.yml/badge.svg)](https://github.com/mbergo/voxgo/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mbergo/voxgo)](https://goreportcard.com/report/github.com/mbergo/voxgo)
[![License: MIT](https://img.shields.io/badge/License-MIT-orange.svg)](LICENSE)

**Accent-friendly, system-wide voice tooling for Linux (Wayland).**
Dictate into any window in *your* accent, or talk to Irene — a sharp-tongued
speech-to-speech assistant — all from one static Go binary powered by the
OpenAI Realtime API.

---

## Why

Local speech models keep mishearing accented English — to the point of having
to fake an American accent just to dictate. `gpt-4o-transcribe` doesn't have
that problem. voxgo streams your microphone straight to it over WebSocket and
types the result into whatever window has focus, system-wide.

Born as a rewrite of a week-long Python/FastAPI project. This version took an
afternoon and has no server, no Python, no dependencies beyond one WebSocket
library.

## What you get

| Command | What it does |
|---|---|
| `voxgo daemon` | Background dictation service, controlled via unix socket |
| `voxgo toggle` | Flip dictation on/off — **bind this to a hotkey** |
| `voxgo chat [voice]` | Speech-to-speech conversation with Irene in the terminal |
| `voxgo web [addr]` | Web dashboard: toggles, voice picker, persona editor, live transcript |
| `voxgo status` | `idle` or `listening` |

## Architecture

```
                 ┌────────────────────────── voxgo (single binary) ───────────────────────────┐
                 │                                                                             │
  🎤 mic ──► pw-record ──► PCM16 ──► WebSocket ──► OpenAI Realtime API (server VAD)            │
                 │                                        │                                    │
                 │                          transcription │ speech                             │
                 │                                        ▼                                    │
  focused window ◄── wtype ◄──────────────── transcripts  +  audio ────► pw-cat ──► 🔊 speakers│
                 │                                                                             │
                 │   unix socket ◄── voxgo toggle (hotkey)      HTTP+SSE ◄── dashboard         │
                 └─────────────────────────────────────────────────────────────────────────────┘
```

- **Dictation** uses a transcription-only Realtime session (`gpt-4o-transcribe`)
  with server-side voice activity detection — you just talk, it types.
- **Chat** uses a full `gpt-realtime` session: your speech in, Irene's voice out,
  with echo-resistant VAD tuning so she doesn't interrupt herself.
- **Everything reconnects** automatically with exponential backoff.

## Install

```bash
# Runtime dependencies (Debian/Ubuntu)
sudo apt install wtype pipewire-utils libnotify-bin

# voxgo itself
go install github.com/mbergo/voxgo@latest
# …or grab a binary from the releases page
```

## Configure

Create `~/.config/voxgo/env`:

```ini
OPENAI_API_KEY=sk-...

# Optional:
VOXGO_SINK=alsa_output.pci-0000_00_1f.3.analog-stereo  # pactl list sinks short
VOXGO_VOICE=shimmer                                     # marin, cedar, alloy, ...
VOXGO_PROMPT=You are …                                  # replace Irene's persona
```

Real environment variables override the file, so `VOXGO_VOICE=marin voxgo chat`
works for one-offs. `chmod 600` the file — it holds your API key.

## Use

### Dictation

```bash
voxgo daemon &        # once per login (or install contrib/voxgo.service)
voxgo toggle          # speak — text appears in the focused window
voxgo toggle          # stop
```

Bind the toggle to a key:

```ini
# Hyprland
bind = SUPER, D, exec, voxgo toggle

# Sway
bindsym $mod+d exec voxgo toggle
```

### Chat with Irene

```bash
voxgo chat            # default voice: shimmer
voxgo chat marin      # any Realtime voice
```

Irene is brilliant, sarcastic, and perpetually three steps ahead — in the
spirit of a certain woman from Sherlock Holmes. Override her with
`VOXGO_PROMPT` if you prefer something friendlier. (Why would you?)

### Dashboard

```bash
voxgo web             # http://127.0.0.1:7853
```

Start/stop dictation and chat, switch voices, edit the persona (persisted to
your config), and watch live transcripts — all from the browser.

## Troubleshooting

| Symptom | Fix |
|---|---|
| No audio from Irene | Set `VOXGO_SINK` — your default sink may be a mic's headphone jack. `pactl list sinks short` |
| Nothing transcribed | Check the mic: `pw-record --format s16 --rate 24000 --channels 1 - \| wc -c` for a few seconds |
| Text not typed | wtype must be installed; works on Wayland compositors with the virtual-keyboard protocol |
| API errors | Run with `VOXGO_DEBUG=1` to see every Realtime event |

## Development

```bash
make build    # static binary
make test     # unit tests
make vet      # static analysis
```

CI runs vet, build, and tests on every push. Contributions welcome.

## License

[MIT](LICENSE)
