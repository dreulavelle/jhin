package main

import (
	"encoding/json"
	"strings"
	"testing"

	jhin "github.com/dreulavelle/jhin"
)

func TestMarshalSetOmitsUnsetFields(t *testing.T) {
	r := jhin.Parse("The.Matrix.1999.1080p.BluRay.x264")
	blob, err := marshalSet(r, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(blob), `{"codec":"avc","quality":"BluRay","resolution":"1080p","title":"The Matrix","year":"1999"}`; got != want {
		t.Errorf("marshalSet =\n %s\nwant\n %s", got, want)
	}
}

// Whatever the short form emits must agree with the full object key for key;
// the flag chooses how much to show, never what the values are.
func TestMarshalSetAgreesWithFull(t *testing.T) {
	for _, title := range []string{
		"The.Matrix.1999.1080p.BluRay.x264",
		"Dr..STONE.(2019)-S04E16-074-1080p-WEB-DL-GROUP",
		"[SubsPlease] Anime - 12 (1080p) [ABCD1234].mkv",
		"Movie.Name.2020.DE.DL.1080p.BluRay.x264-GROUP",
		"",
	} {
		r := jhin.Parse(title)
		blob, err := marshalSet(r, false)
		if err != nil {
			t.Fatal(err)
		}
		var short, full map[string]any
		if err := json.Unmarshal(blob, &short); err != nil {
			t.Fatalf("%q: short output is not valid JSON: %v", title, err)
		}
		fullBlob, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(fullBlob, &full); err != nil {
			t.Fatal(err)
		}
		for k, v := range short {
			fv, ok := full[k]
			if !ok {
				t.Errorf("%q: short emitted %q, absent from the full object", title, k)
				continue
			}
			if a, b := jsonOf(t, v), jsonOf(t, fv); a != b {
				t.Errorf("%q: field %q is %s short, %s full", title, k, a, b)
			}
		}
		for k, v := range full {
			if _, ok := short[k]; ok {
				continue
			}
			switch tv := v.(type) {
			case nil:
			case bool:
				if tv {
					t.Errorf("%q: dropped %q, which was set to true", title, k)
				}
			case string:
				if tv != "" {
					t.Errorf("%q: dropped %q, which was set to %q", title, k, tv)
				}
			case []any:
				if len(tv) > 0 {
					t.Errorf("%q: dropped %q, which held %v", title, k, tv)
				}
			}
		}
	}
}

func TestMarshalSetEmptyAndPretty(t *testing.T) {
	blob, err := marshalSet(jhin.Parse(""), false)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) != "{}" {
		t.Errorf("empty title = %s, want {}", blob)
	}

	blob, err = marshalSet(jhin.Parse("The.Matrix.1999.1080p"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blob), "\n  \"resolution\": \"1080p\"") {
		t.Errorf("pretty output is not indented:\n%s", blob)
	}
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
