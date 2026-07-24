module github.com/dreulavelle/jhin/benchmarks

go 1.25

replace github.com/dreulavelle/jhin => ../

require (
	github.com/ProfChaos/torrent-name-parser v0.5.1
	github.com/dreulavelle/jhin v0.0.0-00010101000000-000000000000
	github.com/middelink/go-parse-torrent-name v0.0.0-20190301154245-3ff4efacd4c4
	github.com/razsteinmetz/go-ptn v1.0.0
)

require golang.org/x/text v0.3.7 // indirect
