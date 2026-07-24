package rank

// The attribute registry: every rankable/filterable property of a release is
// declared exactly once here. Scoring and fetch-filtering both iterate the
// same detection results, so adding an attribute means adding one constant,
// one detection line, and default policy values — nothing else.

import (
	"strings"

	"github.com/dreulavelle/jhin/parser"
)

// Attr identifies a single rankable attribute of a release.
type Attr string

const (
	// sources / qualities
	AttrWeb      Attr = "web"
	AttrWebDL    Attr = "webdl"
	AttrWebMux   Attr = "webmux"
	AttrBluRay   Attr = "bluray"
	AttrDVD      Attr = "dvd"
	AttrHDTV     Attr = "hdtv"
	AttrVHS      Attr = "vhs"
	AttrRemux    Attr = "remux"
	AttrWebRip   Attr = "webrip"
	AttrWebDLRip Attr = "webdlrip"
	AttrUHDRip   Attr = "uhdrip"
	AttrHDRip    Attr = "hdrip"
	AttrDVDRip   Attr = "dvdrip"
	AttrBDRip    Attr = "bdrip"
	AttrBRRip    Attr = "brrip"
	AttrVHSRip   Attr = "vhsrip"
	AttrPPVRip   Attr = "ppvrip"
	AttrSATRip   Attr = "satrip"
	AttrTVRip    Attr = "tvrip"

	// trash sources
	AttrCam      Attr = "cam"
	AttrTeleCine Attr = "telecine"
	AttrTeleSync Attr = "telesync"
	AttrScreener Attr = "screener"
	AttrR5       Attr = "r5"
	AttrPDTV     Attr = "pdtv"

	// codecs
	AttrAVC  Attr = "avc"
	AttrHEVC Attr = "hevc"
	AttrAV1  Attr = "av1"
	AttrXvid Attr = "xvid"
	AttrMPEG Attr = "mpeg"

	// hdr / depth
	AttrDolbyVision Attr = "dolby_vision"
	AttrHDR         Attr = "hdr"
	AttrHDR10Plus   Attr = "hdr10plus"
	AttrSDR         Attr = "sdr"
	Attr10Bit       Attr = "10bit"

	// audio
	AttrAAC              Attr = "aac"
	AttrAtmos            Attr = "atmos"
	AttrDolbyDigital     Attr = "dolby_digital"
	AttrDolbyDigitalPlus Attr = "dolby_digital_plus"
	AttrDTSLossy         Attr = "dts_lossy"
	AttrDTSLossless      Attr = "dts_lossless"
	AttrFLAC             Attr = "flac"
	AttrOPUS             Attr = "opus"
	AttrPCM              Attr = "pcm"
	AttrMP3              Attr = "mp3"
	AttrTrueHD           Attr = "truehd"
	AttrCleanAudio       Attr = "clean_audio"

	// channels
	AttrSurround Attr = "surround"
	AttrStereo   Attr = "stereo"
	AttrMono     Attr = "mono"

	// extras
	Attr3D          Attr = "3d"
	AttrConverted   Attr = "converted"
	AttrDocumentary Attr = "documentary"
	AttrDubbed      Attr = "dubbed"
	AttrEdition     Attr = "edition"
	AttrHardcoded   Attr = "hardcoded"
	AttrNetwork     Attr = "network"
	AttrProper      Attr = "proper"
	AttrRepack      Attr = "repack"
	AttrRetail      Attr = "retail"
	AttrSubbed      Attr = "subbed"
	AttrUpscaled    Attr = "upscaled"
	AttrScene       Attr = "scene"
	AttrUncensored  Attr = "uncensored"
	AttrSite        Attr = "site"
	AttrSize        Attr = "size"
)

// qualityAttrs maps the parser's Quality values onto attributes.
var qualityAttrs = map[string]Attr{
	"WEB":          AttrWeb,
	"WEB-DL":       AttrWebDL,
	"WEBMux":       AttrWebMux,
	"BluRay":       AttrBluRay,
	"DVD":          AttrDVD,
	"HDTV":         AttrHDTV,
	"HDTVRip":      AttrTVRip,
	"VHS":          AttrVHS,
	"REMUX":        AttrRemux,
	"BluRay REMUX": AttrRemux,
	"WEBRip":       AttrWebRip,
	"WEB-DLRip":    AttrWebDLRip,
	"UHDRip":       AttrUHDRip,
	"HDRip":        AttrHDRip,
	"DVDRip":       AttrDVDRip,
	"BDRip":        AttrBDRip,
	"BRRip":        AttrBRRip,
	"VHSRip":       AttrVHSRip,
	"PPVRip":       AttrPPVRip,
	"SATRip":       AttrSATRip,
	"TVRip":        AttrTVRip,
	"CAM":          AttrCam,
	"TeleCine":     AttrTeleCine,
	"TeleSync":     AttrTeleSync,
	"SCR":          AttrScreener,
	"R5":           AttrR5,
	"PDTV":         AttrPDTV,
}

// trashQualityAttrs are the sources considered trash by the hard trash veto.
var trashQualityAttrs = map[Attr]bool{
	AttrCam: true, AttrTeleCine: true, AttrTeleSync: true,
	AttrScreener: true, AttrR5: true, AttrPDTV: true,
}

var codecAttrs = map[string]Attr{
	"avc":  AttrAVC,
	"hevc": AttrHEVC,
	"av1":  AttrAV1,
	"xvid": AttrXvid,
	"mpeg": AttrMPEG,
}

var hdrAttrs = map[string]Attr{
	"DV":     AttrDolbyVision,
	"HDR":    AttrHDR,
	"HDR10+": AttrHDR10Plus,
	"SDR":    AttrSDR,
}

var audioAttrs = map[string]Attr{
	"AAC":                AttrAAC,
	"Atmos":              AttrAtmos,
	"Dolby Digital":      AttrDolbyDigital,
	"Dolby Digital Plus": AttrDolbyDigitalPlus,
	"DTS Lossy":          AttrDTSLossy,
	"DTS Lossless":       AttrDTSLossless,
	"FLAC":               AttrFLAC,
	"OPUS":               AttrOPUS,
	"PCM":                AttrPCM,
	"MP3":                AttrMP3,
	"TrueHD":             AttrTrueHD,
	"HQ Clean Audio":     AttrCleanAudio,
}

var channelAttrs = map[string]Attr{
	"5.1":    AttrSurround,
	"7.1":    AttrSurround,
	"2.0":    AttrStereo,
	"stereo": AttrStereo,
	"mono":   AttrMono,
}

// Attributes returns every rank attribute detected in a parse result —
// exported so apps can drive formatting or their own logic from the same
// detection scoring and filtering use.
func Attributes(d *parser.Result) []Attr {
	return attributes(d)
}

// attributes is the single source of truth consumed by both scoring and
// fetch filtering.
func attributes(d *parser.Result) []Attr {
	attrs := make([]Attr, 0, 12)

	if a, ok := qualityAttrs[d.Quality]; ok {
		attrs = append(attrs, a)
	}
	if a, ok := codecAttrs[strings.ToLower(d.Codec)]; ok {
		attrs = append(attrs, a)
	}
	for _, h := range d.HDR {
		if a, ok := hdrAttrs[h]; ok {
			attrs = append(attrs, a)
		}
	}
	if d.BitDepth != "" {
		attrs = append(attrs, Attr10Bit)
	}
	for _, a := range d.Audio {
		if attr, ok := audioAttrs[a]; ok {
			attrs = append(attrs, attr)
		}
	}
	for _, c := range d.Channels {
		if attr, ok := channelAttrs[c]; ok {
			attrs = append(attrs, attr)
		}
	}

	flag := func(cond bool, a Attr) {
		if cond {
			attrs = append(attrs, a)
		}
	}
	flag(d.ThreeD, Attr3D)
	flag(d.Convert, AttrConverted)
	flag(d.Documentary, AttrDocumentary)
	flag(d.Dubbed, AttrDubbed)
	flag(d.Edition != "", AttrEdition)
	flag(d.Hardcoded, AttrHardcoded)
	flag(d.Network != "", AttrNetwork)
	flag(d.Proper, AttrProper)
	flag(d.Repack, AttrRepack)
	flag(d.Retail, AttrRetail)
	flag(d.Subbed, AttrSubbed)
	flag(d.Upscaled, AttrUpscaled)
	flag(d.Scene, AttrScene)
	flag(d.Uncensored, AttrUncensored)
	flag(d.Site != "", AttrSite)
	flag(d.Size != "", AttrSize)

	return attrs
}
