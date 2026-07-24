Corpus: 1156 titles (parser/testdata/golden.json)

| Field | jhin | torrent-name-parser | go-ptn | go-parse-torrent-name |
|---|---|---|---|---|
| title (1137) | 100.0% | 65.5% | 58.0% | 64.2% |
| year (372) | 100.0% | 95.7% | 96.8% | 96.8% |
| seasons (313) | 100.0% | 67.7% | 40.3% | 40.3% |
| episodes (356) | 100.0% | 39.9% | 32.9% | 49.2% |
| resolution (497) | 100.0% | 95.2% | 89.3% | 89.3% |
| source (531) | 100.0% | 79.5% | 70.4% | 70.4% |
| codec (429) | 100.0% | 97.4% | 82.8% | 82.8% |
| group (352) | 100.0% | 81.5% | 50.6% | 39.8% |
| container (307) | 100.0% | 87.0% | 85.3% | 98.4% |
| **overall** | **100.0%** | **78.8%** | **67.4%** | **70.1%** |

| | jhin | torrent-name-parser | go-ptn | go-parse-torrent-name | 
|---|---|---|---|---|
| spurious extractions | 0 | 992 | 260 | 293 |
| parse errors/panics | 0 | 0 | 13 | 13 |
