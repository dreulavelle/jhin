# Changelog

## [0.3.2](https://github.com/dreulavelle/jhin/compare/v0.3.1...v0.3.2) (2026-07-26)


### Performance Improvements

* gate cleanup regexes and derive gates from lookarounds ([#13](https://github.com/dreulavelle/jhin/issues/13)) ([250eefc](https://github.com/dreulavelle/jhin/commit/250eefc1e0a7e711e48219430bf8528bf39ed9ac))

## [0.3.1](https://github.com/dreulavelle/jhin/compare/v0.3.0...v0.3.1) (2026-07-24)


### Performance Improvements

* **parser:** 2.5x faster parsing via Aho-Corasick prefilter ([3035606](https://github.com/dreulavelle/jhin/commit/303560644c7f93351f172e13cc8224a99a403a30))
* **parser:** multi-factor gates and tighter TELE literals ([d753780](https://github.com/dreulavelle/jhin/commit/d7537809eafa5dfacf6f5428e3d0600e02095f5c))

## [0.3.0](https://github.com/dreulavelle/jhin/compare/v0.2.0...v0.3.0) (2026-07-24)


### Features

* **cli:** jhin version command ([3291e2c](https://github.com/dreulavelle/jhin/commit/3291e2ccb4a09dacbb679900ba7b81e641f21586))

## [0.2.0](https://github.com/dreulavelle/jhin/compare/v0.1.2...v0.2.0) (2026-07-24)


### Features

* **rank:** export Attributes; document app-integration pattern ([4b945b5](https://github.com/dreulavelle/jhin/commit/4b945b5b5d6353dd806adbb74a35f87756a26c54))
* **rank:** preference-order sorting, composable sort chains, weighted patterns ([6171271](https://github.com/dreulavelle/jhin/commit/617127175bac171c9265989b0ddf931ea2c617cc))
* **rank:** score explanations, OPUS/PCM attributes, corpus invariants ([bdbd4f3](https://github.com/dreulavelle/jhin/commit/bdbd4f3f4f61d9ab0dec7b9dcead3cd438c2d564))

## [0.1.2](https://github.com/dreulavelle/jhin/compare/v0.1.1...v0.1.2) (2026-07-24)


### Bug Fixes

* **ci:** publish releases with binaries in one step ([f59dd92](https://github.com/dreulavelle/jhin/commit/f59dd9263b1fe92890f91a45abfdf7b79010fa48))

## [0.1.1](https://github.com/dreulavelle/jhin/compare/v0.1.0...v0.1.1) (2026-07-24)


### Bug Fixes

* **ci:** draft releases so binaries attach before publishing ([d4bf34e](https://github.com/dreulavelle/jhin/commit/d4bf34e72aa64d8b5bb85a66ef0cf8923f8d9036))

## 0.1.0 (2026-07-24)


### Features

* jhin rank CLI, rank benchmarks, CI/CD refresh, full getting-started README ([9e3a837](https://github.com/dreulavelle/jhin/commit/9e3a8375c22a511aca46b1721322d28600ea3625))
* **parser:** language translation, parallel batch parse, extraction helpers ([163c162](https://github.com/dreulavelle/jhin/commit/163c162d978df289d04707bf60599bd269cd611d))
* **parser:** sync to PTT 1.8.5 ([458637f](https://github.com/dreulavelle/jhin/commit/458637f5eed3f6f53f122a604271718672b5ccfe))
* rank package — ranking, filtering, and sorting (rank-torrent-name successor) ([592effa](https://github.com/dreulavelle/jhin/commit/592effa74e938a7b25a43bde5be41ad455fab793))
* **rank:** default profile enables 4K through 720p ([36609b5](https://github.com/dreulavelle/jhin/commit/36609b5df50ae8ac0490395217ede61ffea578fd))
* rebrand go-ptt to jhin with a golden-verified parser ([7c3049e](https://github.com/dreulavelle/jhin/commit/7c3049e1f7f8ec97d1733f6316015f2f052c43af))


### Bug Fixes

* harden ranker and range handling per codex review ([6d81a25](https://github.com/dreulavelle/jhin/commit/6d81a2571f3422e201d1175db0cb806ea0823dcc))
* **parser:** cap range expansion, fold prefilter haystack like RE2 ([88dd068](https://github.com/dreulavelle/jhin/commit/88dd068c36c98fa98ecae6b92abd662d413759f7))


### Performance Improvements

* literal prefilter — 2.8x parse throughput, zero accuracy change ([6a5a776](https://github.com/dreulavelle/jhin/commit/6a5a7760caea934076970968b59be0a5ca260546))
* **parser:** allocation-free fast path for cleanup regexes ([1286132](https://github.com/dreulavelle/jhin/commit/12861327bc3fb118221e327ccaae8e2f6dc9d2d4))

## Changelog
