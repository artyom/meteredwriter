# DEPRECATION WARNING

This package considered as deprecated. Please look at [mio][] package, it is a
fork with a bit cleaner interface and Reader support.

* * *

# meteredwriter

Package meteredwriter provides tools to combine io.Writer and
[metrics.Histogram][1] interfaces, so that every non-empty Write call latency
value is sampled to Histogram.

See [documentation on godoc.org][doc].

This package is intended to be used with [go-metrics][2] or [metrics][3]
packages.

[1]: http://godoc.org/github.com/rcrowley/go-metrics#Histogram
[2]: https://github.com/rcrowley/go-metrics
[3]: https://github.com/facebookgo/metrics
[doc]: http://godoc.org/github.com/artyom/meteredwriter
[mio]: https://github.com/artyom/mio
