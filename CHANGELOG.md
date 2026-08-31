# Changelog

## [0.7.0](https://github.com/dreulavelle/jhin/compare/v0.6.0...v0.7.0) (2026-08-31)


### ⚠ BREAKING CHANGES

* **cli:** `jhin parse` no longer emits unset fields by default. Scripts reading a fixed shape want `--long`.
* profiles that relied on Require to exempt releases from other vetoes no longer do so, and profiles whose titles did not match a Require pattern are now rejected rather than fetched.

### Features

* **cli:** add rules check, fields and fmt ([3c48ad5](https://github.com/dreulavelle/jhin/commit/3c48ad58d58937e8ed542b3517bb5aef5a12abdb))
* **cli:** jhin version command ([3291e2c](https://github.com/dreulavelle/jhin/commit/3291e2ccb4a09dacbb679900ba7b81e641f21586))
* **cli:** print only the fields parse actually set ([ae18256](https://github.com/dreulavelle/jhin/commit/ae1825648b80e634511c33567fb1377cd01eb685))
* jhin rank CLI, rank benchmarks, CI/CD refresh, full getting-started README ([9e3a837](https://github.com/dreulavelle/jhin/commit/9e3a8375c22a511aca46b1721322d28600ea3625))
* **parser:** language translation, parallel batch parse, extraction helpers ([163c162](https://github.com/dreulavelle/jhin/commit/163c162d978df289d04707bf60599bd269cd611d))
* **parser:** sync to PTT 1.8.5 ([458637f](https://github.com/dreulavelle/jhin/commit/458637f5eed3f6f53f122a604271718672b5ccfe))
* rank package — ranking, filtering, and sorting (rank-torrent-name successor) ([592effa](https://github.com/dreulavelle/jhin/commit/592effa74e938a7b25a43bde5be41ad455fab793))
* **rank:** default profile enables 4K through 720p ([36609b5](https://github.com/dreulavelle/jhin/commit/36609b5df50ae8ac0490395217ede61ffea578fd))
* **rank:** export Attributes; document app-integration pattern ([4b945b5](https://github.com/dreulavelle/jhin/commit/4b945b5b5d6353dd806adbb74a35f87756a26c54))
* **rank:** preference-order sorting, composable sort chains, weighted patterns ([6171271](https://github.com/dreulavelle/jhin/commit/617127175bac171c9265989b0ddf931ea2c617cc))
* **rank:** score explanations, OPUS/PCM attributes, corpus invariants ([bdbd4f3](https://github.com/dreulavelle/jhin/commit/bdbd4f3f4f61d9ab0dec7b9dcead3cd438c2d564))
* rebrand go-ptt to jhin with a golden-verified parser ([7c3049e](https://github.com/dreulavelle/jhin/commit/7c3049e1f7f8ec97d1733f6316015f2f052c43af))
* **rules:** a rule engine and expression language ([ed8829e](https://github.com/dreulavelle/jhin/commit/ed8829e8cb1a79275904124ca8c90696fdea819d))
* **rules:** add a rule engine and expression language ([fda39be](https://github.com/dreulavelle/jhin/commit/fda39be2a0c905062802e3c03554032dbebf1d42))
* **rules:** close a value set so a typo'd value fails compile ([27a7bdb](https://github.com/dreulavelle/jhin/commit/27a7bdbd6c885e0dd732404c8bd0ecad3ff92a7c))
* **rules:** continue a rule across indented lines ([d3e6b60](https://github.com/dreulavelle/jhin/commit/d3e6b60977eb7340ce230b0cd18fa48b9bfb1f22))
* **rules:** count a list of flags, and locate errors by line ([52b72cc](https://github.com/dreulavelle/jhin/commit/52b72cc9dd34937b439de2ef3ea3fe4eab02e87f))
* **rules:** enforce the profile's syntax version ([95fe8c9](https://github.com/dreulavelle/jhin/commit/95fe8c9c3d3603f3b4a510621dd9906a34b53dde))


### Bug Fixes

* anime episode ranges, bare DE as German, and non-default CLI output ([f4d9495](https://github.com/dreulavelle/jhin/commit/f4d9495a79d9444635997e0fc4fc34e4cc1c758b))
* **ci:** draft releases so binaries attach before publishing ([d4bf34e](https://github.com/dreulavelle/jhin/commit/d4bf34e72aa64d8b5bb85a66ef0cf8923f8d9036))
* **ci:** publish releases with binaries in one step ([f59dd92](https://github.com/dreulavelle/jhin/commit/f59dd9263b1fe92890f91a45abfdf7b79010fa48))
* harden ranker and range handling per codex review ([6d81a25](https://github.com/dreulavelle/jhin/commit/6d81a2571f3422e201d1175db0cb806ea0823dcc))
* make Require a gate instead of a veto bypass ([#19](https://github.com/dreulavelle/jhin/issues/19)) ([3f78ff1](https://github.com/dreulavelle/jhin/commit/3f78ff15b00eb3b28eb93ef692abf0783c5f65f7))
* **parser:** cap range expansion, fold prefilter haystack like RE2 ([88dd068](https://github.com/dreulavelle/jhin/commit/88dd068c36c98fa98ecae6b92abd662d413759f7))
* **parser:** don't read SxxEyy-NNN as an episode range ([e6868ed](https://github.com/dreulavelle/jhin/commit/e6868ed4d7da96c33b77023f6e6a225683099ef7))
* **parser:** stop the site handler from swallowing the whole title ([21ca5d1](https://github.com/dreulavelle/jhin/commit/21ca5d1f914e8710ac085c5f8ebd9c5af3544375))
* **parser:** stop the site handler from swallowing the whole title ([4e6b81d](https://github.com/dreulavelle/jhin/commit/4e6b81d7c4183ec037cb5f4aa5b6accb644ae48e))
* **parser:** widen the contexts where bare DE means German ([9026cfd](https://github.com/dreulavelle/jhin/commit/9026cfd5e6f1c9faf89536f238c41c48547c9d88))
* reduce WxH resolutions to their height when normalizing ([#16](https://github.com/dreulavelle/jhin/issues/16)) ([1db4997](https://github.com/dreulavelle/jhin/commit/1db49970378fa0560bac11d057bb6c5a25227d6b))
* **rules:** an explicit SDR tag is not an HDR fallback ([5b90c81](https://github.com/dreulavelle/jhin/commit/5b90c81eba99d49b9d113af8f80b4a39cfb8a437))
* **rules:** apply review round-1 findings ([d29954c](https://github.com/dreulavelle/jhin/commit/d29954c9747c83b0063ac9a40f95641aac42ee52))
* **rules:** give result-set questions the content kind ([10e3b32](https://github.com/dreulavelle/jhin/commit/10e3b321999c98b41706580f3fdba289a4447418))
* **rules:** keep fmt exact, guard score conversion, count only viable releases ([d847f50](https://github.com/dreulavelle/jhin/commit/d847f50fd05dbb230e3b4444204153cfb9169965))
* **rules:** make the text form a fixed point under fmt ([5766ffc](https://github.com/dreulavelle/jhin/commit/5766ffc57fdda853247316b7fb841f79b6b7cd0d))
* **rules:** read bitDepth out of the parser's "10bit" spelling ([0533bcf](https://github.com/dreulavelle/jhin/commit/0533bcf69378bb528e2d41f79f19211b8c8c8236))
* **rules:** saturate the score total and reserve off inside scope groups ([f3a75c6](https://github.com/dreulavelle/jhin/commit/f3a75c6d4de5b36c1302aa25bce755c56fc6ad17))
* **rules:** serialise an effect's value ([a9c9414](https://github.com/dreulavelle/jhin/commit/a9c9414ec52767ffddebd203e58d6e39a71f4b20))


### Performance Improvements

* gate cleanup regexes and derive gates from lookarounds ([#13](https://github.com/dreulavelle/jhin/issues/13)) ([250eefc](https://github.com/dreulavelle/jhin/commit/250eefc1e0a7e711e48219430bf8528bf39ed9ac))
* literal prefilter — 2.8x parse throughput, zero accuracy change ([6a5a776](https://github.com/dreulavelle/jhin/commit/6a5a7760caea934076970968b59be0a5ca260546))
* **parser:** 2.5x faster parsing via Aho-Corasick prefilter ([3035606](https://github.com/dreulavelle/jhin/commit/303560644c7f93351f172e13cc8224a99a403a30))
* **parser:** allocation-free fast path for cleanup regexes ([1286132](https://github.com/dreulavelle/jhin/commit/12861327bc3fb118221e327ccaae8e2f6dc9d2d4))
* **parser:** multi-factor gates and tighter TELE literals ([d753780](https://github.com/dreulavelle/jhin/commit/d7537809eafa5dfacf6f5428e3d0600e02095f5c))
* **rank:** assemble a release's facts once per batch ([a3e4934](https://github.com/dreulavelle/jhin/commit/a3e493494d0b0443a0caff987ae50015c5280bfa))
* **rules:** branch instead of a map literal in BoolOf ([4ed1954](https://github.com/dreulavelle/jhin/commit/4ed1954928eaafd6eaab8bd6cd825a2b67de3829))

## [0.6.0](https://github.com/dreulavelle/jhin/compare/v0.5.0...v0.6.0) (2026-08-31)


### Features

* **cli:** add rules check, fields and fmt ([3c48ad5](https://github.com/dreulavelle/jhin/commit/3c48ad58d58937e8ed542b3517bb5aef5a12abdb))
* **rules:** a rule engine and expression language ([ed8829e](https://github.com/dreulavelle/jhin/commit/ed8829e8cb1a79275904124ca8c90696fdea819d))
* **rules:** add a rule engine and expression language ([fda39be](https://github.com/dreulavelle/jhin/commit/fda39be2a0c905062802e3c03554032dbebf1d42))
* **rules:** close a value set so a typo'd value fails compile ([27a7bdb](https://github.com/dreulavelle/jhin/commit/27a7bdbd6c885e0dd732404c8bd0ecad3ff92a7c))
* **rules:** continue a rule across indented lines ([d3e6b60](https://github.com/dreulavelle/jhin/commit/d3e6b60977eb7340ce230b0cd18fa48b9bfb1f22))
* **rules:** count a list of flags, and locate errors by line ([52b72cc](https://github.com/dreulavelle/jhin/commit/52b72cc9dd34937b439de2ef3ea3fe4eab02e87f))
* **rules:** enforce the profile's syntax version ([95fe8c9](https://github.com/dreulavelle/jhin/commit/95fe8c9c3d3603f3b4a510621dd9906a34b53dde))


### Bug Fixes

* **rules:** an explicit SDR tag is not an HDR fallback ([5b90c81](https://github.com/dreulavelle/jhin/commit/5b90c81eba99d49b9d113af8f80b4a39cfb8a437))
* **rules:** apply review round-1 findings ([d29954c](https://github.com/dreulavelle/jhin/commit/d29954c9747c83b0063ac9a40f95641aac42ee52))
* **rules:** give result-set questions the content kind ([10e3b32](https://github.com/dreulavelle/jhin/commit/10e3b321999c98b41706580f3fdba289a4447418))
* **rules:** keep fmt exact, guard score conversion, count only viable releases ([d847f50](https://github.com/dreulavelle/jhin/commit/d847f50fd05dbb230e3b4444204153cfb9169965))
* **rules:** make the text form a fixed point under fmt ([5766ffc](https://github.com/dreulavelle/jhin/commit/5766ffc57fdda853247316b7fb841f79b6b7cd0d))
* **rules:** read bitDepth out of the parser's "10bit" spelling ([0533bcf](https://github.com/dreulavelle/jhin/commit/0533bcf69378bb528e2d41f79f19211b8c8c8236))
* **rules:** saturate the score total and reserve off inside scope groups ([f3a75c6](https://github.com/dreulavelle/jhin/commit/f3a75c6d4de5b36c1302aa25bce755c56fc6ad17))
* **rules:** serialise an effect's value ([a9c9414](https://github.com/dreulavelle/jhin/commit/a9c9414ec52767ffddebd203e58d6e39a71f4b20))


### Performance Improvements

* **rank:** assemble a release's facts once per batch ([a3e4934](https://github.com/dreulavelle/jhin/commit/a3e493494d0b0443a0caff987ae50015c5280bfa))
* **rules:** branch instead of a map literal in BoolOf ([4ed1954](https://github.com/dreulavelle/jhin/commit/4ed1954928eaafd6eaab8bd6cd825a2b67de3829))

## [0.5.0](https://github.com/dreulavelle/jhin/compare/v0.4.1...v0.5.0) (2026-08-30)


### ⚠ BREAKING CHANGES

* **cli:** `jhin parse` no longer emits unset fields by default. Scripts reading a fixed shape want `--long`.

### Features

* **cli:** print only the fields parse actually set ([ae18256](https://github.com/dreulavelle/jhin/commit/ae1825648b80e634511c33567fb1377cd01eb685))


### Bug Fixes

* anime episode ranges, bare DE as German, and non-default CLI output ([f4d9495](https://github.com/dreulavelle/jhin/commit/f4d9495a79d9444635997e0fc4fc34e4cc1c758b))
* **parser:** don't read SxxEyy-NNN as an episode range ([e6868ed](https://github.com/dreulavelle/jhin/commit/e6868ed4d7da96c33b77023f6e6a225683099ef7))
* **parser:** widen the contexts where bare DE means German ([9026cfd](https://github.com/dreulavelle/jhin/commit/9026cfd5e6f1c9faf89536f238c41c48547c9d88))

## [0.4.1](https://github.com/dreulavelle/jhin/compare/v0.4.0...v0.4.1) (2026-08-06)


### Bug Fixes

* **parser:** stop the site handler from swallowing the whole title ([4e6b81d](https://github.com/dreulavelle/jhin/commit/4e6b81d7c4183ec037cb5f4aa5b6accb644ae48e))

## [0.4.0](https://github.com/dreulavelle/jhin/compare/v0.3.3...v0.4.0) (2026-07-26)


### ⚠ BREAKING CHANGES

* profiles that relied on Require to exempt releases from other vetoes no longer do so, and profiles whose titles did not match a Require pattern are now rejected rather than fetched.

### Bug Fixes

* make Require a gate instead of a veto bypass ([#19](https://github.com/dreulavelle/jhin/issues/19)) ([3f78ff1](https://github.com/dreulavelle/jhin/commit/3f78ff15b00eb3b28eb93ef692abf0783c5f65f7))

## [0.3.3](https://github.com/dreulavelle/jhin/compare/v0.3.2...v0.3.3) (2026-07-26)


### Bug Fixes

* reduce WxH resolutions to their height when normalizing ([#16](https://github.com/dreulavelle/jhin/issues/16)) ([1db4997](https://github.com/dreulavelle/jhin/commit/1db49970378fa0560bac11d057bb6c5a25227d6b))

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
