// Times the npm parse-torrent-title package over the golden corpus.
// In-process timing: V8 startup and corpus loading are excluded.
//
//   npm install
//   node bench_ptt.js ../../parser/testdata/golden.json [passes]

const fs = require("fs");
const ptt = require("parse-torrent-title");

const corpusPath = process.argv[2];
const passes = parseInt(process.argv[3] || "3", 10);
const titles = JSON.parse(fs.readFileSync(corpusPath, "utf8")).map((e) => e.title);

for (const t of titles) ptt.parse(t); // warmup: let V8 optimize

const start = process.hrtime.bigint();
for (let p = 0; p < passes; p++) {
  for (const t of titles) ptt.parse(t);
}
const elapsedNs = Number(process.hrtime.bigint() - start);

const perTitleUs = elapsedNs / (passes * titles.length) / 1000;
console.log(`parse-torrent-title (npm)\t${titles.length} titles x ${passes}\t${perTitleUs.toFixed(1)} us/title`);
