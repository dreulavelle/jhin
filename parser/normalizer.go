package parser

import "strings"

func normalizeAudio(audio []string) []string {
	isChanged := false
	for i := range audio {
		switch audio[i] {
		case "AC3":
			audio[i] = "DD"
			isChanged = true
		case "EAC3":
			audio[i] = "DDP"
			isChanged = true
		}
	}
	if !isChanged {
		return audio
	}
	nAudio := []string{}
	seenMap := map[string]struct{}{}
	for _, item := range audio {
		if _, seen := seenMap[item]; !seen {
			nAudio = append(nAudio, item)
			seenMap[item] = struct{}{}
		}
	}
	return nAudio
}

func normalizeCodec(codec string) string {
	codec = strings.ToLower(codec)
	switch codec {
	case "avc", "h264", "x264":
		return "AVC"
	case "hevc", "h265", "x265":
		return "HEVC"
	case "mpeg2":
		return "MPEG-2"
	case "divx", "dvix":
		return "DivX"
	case "xvid":
		return "Xvid"
	default:
		return codec
	}
}

// ResolutionHeight reduces a WxH resolution to its height. Titles that spell
// out both dimensions parse to the full pair ("720x480p"), which callers
// matching on tiers would otherwise key off the width. Values that are not
// WxH are returned lowercased and unchanged.
func ResolutionHeight(resolution string) string {
	resolution = strings.ToLower(resolution)
	i := strings.LastIndex(resolution, "x")
	if i <= 0 || i+1 >= len(resolution) {
		return resolution
	}
	for _, c := range resolution[:i] {
		if c < '0' || c > '9' {
			return resolution
		}
	}
	return resolution[i+1:]
}

func normalizeResolution(resolution string) string {
	height := ResolutionHeight(resolution)
	switch height {
	case "2160p":
		return "4k"
	case "1440p":
		return "2k"
	default:
		return height
	}
}

func (r *Result) Normalize() *Result {
	if r.Error() != nil {
		return r
	}
	if !r.isNormalized {
		r.Audio = normalizeAudio(r.Audio)
		r.Codec = normalizeCodec(r.Codec)
		r.Resolution = normalizeResolution(r.Resolution)
		r.isNormalized = true
	}
	return r
}
