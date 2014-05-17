# meteredwriter

Package meteredwriter provides tools to combine io.Writer and
[metrics.Histogram][1] interfaces, so that every non-empty Write call latency
value is sampled to Histogram.

See [documentation on
godoc.org](http://godoc.org/github.com/artyom/meteredwriter)

This package is intended to be used with
[go-metrics](https://github.com/rcrowley/go-metrics)

[1]: http://godoc.org/github.com/rcrowley/go-metrics#Histogram
