package jhin

import "strconv"

type ver struct {
	v     string
	i     int
	major string
	minor string
	patch string
}

var v = ver{
	// x-release-please-start-version
	v: "0.3.1",
	// x-release-please-end
	// x-release-please-start-major
	major: "0",
	// x-release-please-end
	// x-release-please-start-minor
	minor: "3",
	// x-release-please-end
	// x-release-please-start-patch
	patch: "1",
	// x-release-please-end
}

func digits(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}

func (v ver) Int() int {
	if v.i != 0 {
		return v.i
	}
	major, _ := strconv.Atoi(digits(v.major))
	minor, _ := strconv.Atoi(digits(v.minor))
	patch, _ := strconv.Atoi(digits(v.patch))
	v.i = major*1000000 + minor*1000 + patch
	return v.i
}

func (v ver) String() string {
	return v.v
}

func Version() ver {
	return v
}
