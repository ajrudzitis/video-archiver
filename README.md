# video-archiver

A terminal UI for splitting digitized home movies into titled segments.

Browse a directory of video files, mark in/out points with a live frame
preview, give each segment a title, and let ffmpeg re-encode them into
individual files.

## Requirements

- Go 1.26+
- `ffmpeg` and `ffprobe` on PATH (e.g. `nix-shell -p ffmpeg`)

## Usage

```
video-archiver [dir|file] [-preview auto|kitty|halfblock|off]
```

Work in progress. See milestones in git history.
