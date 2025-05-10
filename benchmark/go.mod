module addon

go 1.24

require github.com/Dasio/go-stremio v0.0.0

require (
	github.com/VictoriaMetrics/metrics v1.36.0 // indirect
	github.com/valyala/fastrand v1.1.0 // indirect
	github.com/valyala/histogram v1.2.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
)

replace github.com/Dasio/go-stremio => ../
