package media

import "testing"

const probeFixture = `{
  "streams": [
    {"index": 0, "codec_type": "video", "codec_name": "mpeg2video",
     "width": 720, "height": 576, "avg_frame_rate": "25/1", "r_frame_rate": "25/1"},
    {"index": 1, "codec_type": "audio", "codec_name": "mp2"}
  ],
  "format": {"duration": "3661.250000"}
}`

func TestParseProbe(t *testing.T) {
	info, err := parseProbe([]byte(probeFixture))
	if err != nil {
		t.Fatal(err)
	}
	if info.Duration != 3661.25 {
		t.Errorf("duration = %v", info.Duration)
	}
	if info.Width != 720 || info.Height != 576 {
		t.Errorf("dimensions = %dx%d", info.Width, info.Height)
	}
	if info.FPS != 25 {
		t.Errorf("fps = %v", info.FPS)
	}
	if info.VideoCodec != "mpeg2video" || info.AudioCodec != "mp2" {
		t.Errorf("codecs = %s/%s", info.VideoCodec, info.AudioCodec)
	}
}

func TestParseProbeNoVideo(t *testing.T) {
	if _, err := parseProbe([]byte(`{"streams":[{"codec_type":"audio"}],"format":{}}`)); err == nil {
		t.Error("expected error for missing video stream")
	}
}

func TestParseRate(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{"30000/1001", 30000.0 / 1001.0},
		{"25/1", 25},
		{"0/0", 0},
		{"29.97", 29.97},
		{"", 0},
		{"garbage", 0},
	} {
		if got := parseRate(tc.in); got != tc.want {
			t.Errorf("parseRate(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
