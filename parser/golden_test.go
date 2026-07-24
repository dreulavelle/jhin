package parser

// Golden corpus test: testdata/golden.json was produced by running Python
// PTT v1.6.16 over every title in its test suite. It is the accuracy
// contract: jhin must keep producing byte-identical fields.

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

type goldenCase struct {
	Title  string         `json:"title"`
	Result map[string]any `json:"result"`
}

func normalizeGolden(v any) any {
	switch t := v.(type) {
	case float64:
		return fmt.Sprintf("%v", int(t))
	case []any:
		out := make([]string, len(t))
		for i, e := range t {
			out[i] = fmt.Sprintf("%v", e)
		}
		return out
	default:
		return v
	}
}

func TestGoldenCorpus(t *testing.T) {
	blob, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []goldenCase
	if err := json.Unmarshal(blob, &cases); err != nil {
		t.Fatal(err)
	}

	mismatchedTitles := 0
	fieldMismatches := map[string]int{}

	for _, c := range cases {
		r := Parse(c.Title)
		if r.Error() != nil {
			t.Errorf("parse error on %q: %v", c.Title, r.Error())
			mismatchedTitles++
			continue
		}
		goBlob, _ := json.Marshal(r)
		var got map[string]any
		_ = json.Unmarshal(goBlob, &got)

		bad := false
		// python-present keys must match
		for k, pv := range c.Result {
			gv, ok := got[k]
			if !ok {
				continue // field not represented in Go (should not happen)
			}
			npv, ngv := normalizeGolden(pv), normalizeGolden(gv)
			// python omits zero values only sometimes; both empty-ish is fine
			if !reflect.DeepEqual(npv, ngv) {
				fieldMismatches[k]++
				if !bad {
					bad = true
					mismatchedTitles++
				}
				if fieldMismatches[k] <= 3 {
					t.Logf("MISMATCH %q field=%s python=%#v go=%#v", c.Title, k, npv, ngv)
				}
			}
		}
		// Go-matched fields that python did not match at all
		for k, gv := range got {
			if _, ok := c.Result[k]; ok {
				continue
			}
			switch tv := gv.(type) {
			case bool:
				if tv {
					fieldMismatches[k]++
					if !bad {
						bad = true
						mismatchedTitles++
					}
					if fieldMismatches[k] <= 3 {
						t.Logf("EXTRA %q field=%s go=%#v (python: unset)", c.Title, k, gv)
					}
				}
			case string:
				if tv != "" {
					fieldMismatches[k]++
					if !bad {
						bad = true
						mismatchedTitles++
					}
					if fieldMismatches[k] <= 3 {
						t.Logf("EXTRA %q field=%s go=%#v (python: unset)", c.Title, k, gv)
					}
				}
			case []any:
				if len(tv) > 0 {
					fieldMismatches[k]++
					if !bad {
						bad = true
						mismatchedTitles++
					}
					if fieldMismatches[k] <= 3 {
						t.Logf("EXTRA %q field=%s go=%#v (python: unset)", c.Title, k, gv)
					}
				}
			}
		}
	}

	if mismatchedTitles > 0 {
		t.Errorf("golden corpus: %d/%d titles diverge; per-field: %v", mismatchedTitles, len(cases), fieldMismatches)
	}
}
