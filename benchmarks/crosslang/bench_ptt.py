# Times Python PTT (pypi: parsett) over the golden corpus.
# In-process timing: interpreter startup and corpus loading are excluded,
# so the number is the library's parse cost, comparable to Go ns/op.
#
#   uv venv && uv pip install parsett
#   .venv/bin/python bench_ptt.py ../../parser/testdata/golden.json [passes]

import json
import sys
import time

import PTT


def main() -> None:
    corpus_path = sys.argv[1]
    passes = int(sys.argv[2]) if len(sys.argv) > 2 else 3
    with open(corpus_path, encoding="utf-8") as f:
        titles = [e["title"] for e in json.load(f)]

    for t in titles:  # warmup
        PTT.parse_title(t)

    start = time.perf_counter()
    for _ in range(passes):
        for t in titles:
            PTT.parse_title(t)
    elapsed = time.perf_counter() - start

    per_title_us = elapsed / (passes * len(titles)) * 1e6
    print(
        f"PTT (parsett)\t{len(titles)} titles x {passes}\t{per_title_us:.1f} us/title"
    )


if __name__ == "__main__":
    main()
