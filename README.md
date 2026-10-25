<p align="center">
  <img src="docs/logo.png" alt="Video Archiver: film frames in an archive tray" width="180">
</p>

# video-archiver

Split digitized home movies into named clips from your terminal.

Browse videos, preview frames, mark clip boundaries, and export individual
H.264/AAC MP4 files. Segment lists are saved alongside each source video so
you can return to a project later.

## Get started

You need Go 1.26.5 or newer to build, plus `ffmpeg` and `ffprobe` on
your `PATH`. Your ffmpeg build must include the `libx264` and `aac`
encoders; the application checks for both at startup.

On NixOS, open a shell with ffmpeg available:

```sh
nix-shell -p ffmpeg
```

Build from the repository root, then open a directory or video:

```sh
go build -o video-archiver .

./video-archiver
./video-archiver /path/to/videos
./video-archiver /path/to/holiday.mp4
```

### Make your first clip

1. Select a video with the arrow keys and press `enter`. Files with saved
   segment lists have a diamond marker.
2. Find the clip's start with `left` / `right` (one second at a time),
   `,` / `.` (one frame), or `:` (jump to a timestamp). Press `i` to
   mark the start.
3. Move to the end boundary and press `o`. Enter a title to save the segment.
4. Repeat for any other clips. Press `tab` to select, rename, or adjust
   existing segments.
5. Press `x` to export. If output files already exist, choose overwrite,
   skip existing files, or cancel.

Clips go into `<source basename>_segments/` next to the source. For example,
`holiday.mp4` exports into `holiday_segments/`. Press `O` with transport
focused to choose another directory. Relative paths are resolved from the
source video's directory.

Segment edits save automatically. If saving fails, the error stays visible;
retry with `s`. Leaving the editor retries any unsaved changes and stays
in the editor if that retry fails.

## Command-line options

```text
video-archiver [dir|file] [flags]

-preview auto|kitty|halfblock|off   Preview renderer (default: auto)
-version                          Print version and exit
```

Flags work before or after the path. With no path, the browser opens the
current directory. Use `--` before a path that starts with a dash.

```sh
./video-archiver holiday.mp4 -preview halfblock
./video-archiver -preview off /path/to/videos
```

Automatic preview selection uses Kitty graphics on Kitty, WezTerm, and
Ghostty; other terminals use truecolor half-block characters. Choose
`-preview halfblock` or `-preview off` if needed.

## Key reference

Press `h` or `?` on any screen for help. Use `↑`/`↓` to scroll and any other
key to close. In text prompts, use `F1` for help so `h` remains available for typing.

### Browser

| Key | Action |
| --- | --- |
| `↑`/`↓`, `k`/`j` | move cursor |
| `enter`, `l` | open file / enter directory |
| `backspace`, `←` | parent directory |
| `g` / `G`, `home` / `end` | first / last entry |
| `pgup` / `pgdown` | page |
| `r` | reload directory |
| `q` | quit (confirm) |

### Editor: transport

| Key | Action |
| --- | --- |
| `←`/`→` | seek ±1s |
| `shift+←/→`, `{`/`}` | seek ±10s |
| `,` / `.` | step one frame |
| `home` / `end` | jump to start/end |
| `:` | jump to timestamp (`SS.s`, `MM:SS.s`, or `HH:MM:SS.s`) |
| `i` / `o` | set in-point / out-point (`o` opens the title prompt) |
| `O` | set output directory |
| `x` | run the split |
| `s` | retry saving the segment list |
| `tab` | focus segment list |
| `h`, `?`, `F1` | help overlay |
| `esc` | back to browser |
| `q` | quit (confirm) |

### Editor: segment list

| Key | Action |
| --- | --- |
| `↑`/`↓`, `k`/`j` | select segment |
| `enter` | jump preview to segment start |
| `t` / `d` | rename / delete |
| `I` / `O` | set selected segment's start/end at cursor |
| `[` / `]` | nudge start ∓ one frame |
| `{` / `}` | nudge end ∓ one frame |
| `tab`, `esc` | back to transport |

### Jobs

| Key | Action |
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

Output filenames are the sanitized segment titles plus `.mp4`; titles must
be unique per video. Only the marked ranges are exported. The source video
is preserved.

Malformed JSON sidecars are moved to a unique `.corrupt` backup before the
editor starts a fresh segment list.

## Export behavior

Each segment is re-encoded with these settings:

```sh
ffmpeg -ss <start> -i <source> -t <duration> -map 0:v:0 -map '0:a?' \
  -c:v libx264 -preset medium -crf 18 -c:a aac -b:a 192k \
  -movflags +faststart <temporary-output>.mp4
```

Re-encoding allows cuts between keyframes. Jobs run sequentially, with
progress reported by ffmpeg. Press `esc` to cancel the run.

Each clip is encoded to a temporary file in the output directory and moved
to its final name only after encoding succeeds. Failed or cancelled jobs
remove their temporary files and preserve existing outputs. Destinations
that identify the source video are rejected.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
```

Integration tests use ffmpeg to generate sample videos and check probing,
previews, and exports. They skip when the required tools or encoders are
unavailable.
