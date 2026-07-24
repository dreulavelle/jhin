# Times guessit over the golden corpus. Single pass — guessit is deep
# (rebulk pattern engine) and a full pass already takes tens of seconds.
#
#   uv venv && uv pip install guessit
#   .venv/bin/python bench_guessit.py ../../parser/testdata/golden.json [passes]

import json
import sys
import time

from guessit import guessit


def main() -> None:
    corpus_path = sys.argv[1]
    passes = int(sys.argv[2]) if len(sys.argv) > 2 else 1
    with open(corpus_path, encoding="utf-8") as f:
        titles = [e["title"] for e in json.load(f)]

    for t in titles[:50]:  # warmup: prime rebulk caches
        guessit(t)

    start = time.perf_counter()
    for _ in range(passes):
        for t in titles:
            guessit(t)
    elapsed = time.perf_counter() - start

    per_title_us = elapsed / (passes * len(titles)) * 1e6
    print(f"guessit\t{len(titles)} titles x {passes}\t{per_title_us:.1f} us/title")


if __name__ == "__main__":
    main()
