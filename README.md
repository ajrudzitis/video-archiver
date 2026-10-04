# video-archiver

A terminal UI for splitting digitized home movies into titled segments.

Browse a directory of video files, scrub through a file with a live frame
preview, mark in/out points, give each segment a title, and let ffmpeg
re-encode them into individual `.mp4` files. Segment lists are saved
alongside each video, so you can pick up where you left off.

```
┌──────────────────────────────────────────────╮ position 00:12.360 / 01:30.000
│                                              │ ────────●─────────────────────
│              [frame preview]                 │
│                                              │ in  00:10.000  out --:--
│                                              │ segments  (2)
│                                              │  1 00:00.0  00:10.0  grandma arrives
└──────────────────────────────────────────────╯  2 00:10.0  00:25.0  the cake
```

## Requirements

- Go 1.26+ to build
- `ffmpeg` and `ffprobe` on `PATH`, built with `libx264` and `aac` encoders
  (checked at startup). On NixOS: `nix-shell -p ffmpeg`

## Build

```sh
go build -o video-archiver .
```

## Usage

```
video-archiver [dir|file] [flags]

flags:
  -preview auto|kitty|halfblock|off   frame preview renderer (default auto)
  -version                            print version and exit
```

- With no argument, browses the current directory.
- With a directory, browses that directory.
- With a file, opens it directly in the editor.

Preview rendering auto-detects the terminal: Kitty graphics protocol on
Kitty/WezTerm/Ghostty, truecolor half-block character art everywhere else.

### Workflow

1. Pick a video in the browser (`↑`/`↓`, `enter`). Files with saved
   segments are marked `◆`.
2. Scrub to a scene (`←`/`→`, `,`/`.` for single frames, `:` to type a
   timestamp). The preview follows the cursor.
3. Press `i` at the first frame of the segment, `o` at the last, and type
   a title. The segment is saved immediately.
4. Repeat, then press `x` to split. Each segment is re-encoded to
   H.264/AAC and written to the output directory (default:
   `<video>_segments/` next to the source, changeable with `O`).
5. Existing output files prompt for overwrite/skip. `esc` cancels a run
   and kills the encoder.

### Key bindings

Browser:

| key | action |
| --- | --- |
| `↑`/`↓`, `k`/`j` | move cursor |
| `enter`, `l` | open file / enter directory |
| `backspace`, `h` | parent directory |
| `g` / `G`, `home` / `end` | first / last entry |
| `pgup` / `pgdown` | page |
| `r` | reload directory |
| `q` | quit (confirm) |

Editor (transport focus):

| key | action |
| --- | --- |
| `←`/`→` | seek ±1s |
| `shift+←/→`, `{`/`}` | seek ±10s |
| `,` / `.` | step one frame |
| `home` / `end` | jump to start/end |
| `:` | jump to timestamp (`SS.s`, `MM:SS.s`, or `HH:MM:SS.s`) |
| `i` / `o` | set in-point / out-point (`o` opens the title prompt) |
| `O` | set output directory |
| `x` | run the split |
| `s` | save sidecar now (autosaved anyway) |
| `tab` | focus segment list |
| `?` | help overlay |
| `esc` | back to browser |
| `q` | quit (confirm) |

Editor (segment list focus):

| key | action |
| --- | --- |
| `↑`/`↓`, `k`/`j` | select segment |
| `enter` | jump preview to segment start |
| `t` / `d` | rename / delete |
| `I` / `O` | set selected segment's start/end at cursor |
| `[` / `]` | nudge start ∓ one frame |
| `{` / `}` | nudge end ∓ one frame |
| `tab`, `esc` | back to transport |

Jobs:

| key | action |
| --- | --- |
| `o` / `s` / `c` | overwrite / skip existing outputs / cancel (confirm screen) |
| `esc` | cancel run, or back to editor when finished |

## Sidecar files

Segments are stored next to the source as `<video>.segments.json`:

```json
{
  "version": 1,
  "source": "holiday.mp4",
  "duration": 3661.25,
  "output_dir": "/videos/holiday_segments",
  "segments": [
    { "start": 10.5, "end": 72.25, "title": "grandma arrives" }
  ]
}
```

Output file names are the sanitized segment titles plus `.mp4`
(titles must be unique per video). Everything not covered by a segment
is discarded.

## Splitting details

Each segment runs:

```
ffmpeg -ss <start> -i <source> -t <duration> -map 0:v:0 -map 0:a? \
  -c:v libx264 -preset medium -crf 18 -c:a aac -b:a 192k \
  -movflags +faststart <title>.mp4
```

Re-encoding makes cuts frame-accurate (stream-copy would snap to
keyframes). Jobs run sequentially; progress comes from ffmpeg's
`-progress` output. Failed or cancelled outputs are deleted so they
don't linger as partial files.

## Development

```sh
go test ./...     # unit tests; ffmpeg integration tests skip if absent
go vet ./...
```
