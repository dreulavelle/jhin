package parser

import "testing"

func TestResolutionHeight(t *testing.T) {
	tests := map[string]string{
		"720x480p":   "480p",
		"1440x1080p": "1080p",
		"1916x1080p": "1080p",
		"3840x2160p": "2160p",
		"2160p":      "2160p",
		"1080p":      "1080p",
		"4k":         "4k",
		"":           "",
		// Not a WxH pair: nothing to reduce.
		"x264":  "x264",
		"720x":  "720x",
		"x1080": "x1080",
	}

	for in, want := range tests {
		if got := ResolutionHeight(in); got != want {
			t.Errorf("ResolutionHeight(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeResolutionReducesFullDimensions(t *testing.T) {
	tests := map[string]string{
		"720x480p":   "480p",
		"3840x2160p": "4k",
		"2560x1440p": "2k",
		"2160p":      "4k",
		"1440p":      "2k",
		"1080p":      "1080p",
	}

	for in, want := range tests {
		if got := normalizeResolution(in); got != want {
			t.Errorf("normalizeResolution(%q) = %q, want %q", in, got, want)
		}
	}
}
