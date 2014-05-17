// Package meteredwriter provides tools to combine io.Writer and
// metrics.Histogram interfaces, so that every non-empty Write call latency
// value is sampled to Histogram.
//
// MeteredWriter can be used to bind standard metrics.Histogram to io.Writer.
// SelfCleaningHistogram provides a wrapper over metrics.Histogram with
// self-cleaning capabilities, which can be used for sharing one Histogram over
// multiple io.Writers and cleaning sample pool after period of inactivity.
//
// This package is intended to be used with go-metrics:
// https://github.com/rcrowley/go-metrics
package meteredwriter

import (
	"io"
	"sync"
	"time"
)

// Histogram interface wraps a subset of methods of metrics.Histogram interface
// so it can be used without type conversion.
//
// For full implementation of Histogram interface see go-metrics package:
// http://github.com/rcrowley/go-metrics
type Histogram interface {
	Clear()
	Count() int64
	Max() int64
	Mean() float64
	Min() int64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	StdDev() float64
	Update(int64)
	Variance() float64
}

// MeteredWriter wraps io.Writer and registers each write operation latency in
// attached histogram
type MeteredWriter struct {
	io.Writer
	h Histogram
}

// NewMeteredWriter attaches provided histogram to writer, returning new
// io.Writer. If histogram implements Registrar interface, this would also call
// its Register() method.
func NewMeteredWriter(writer io.Writer, h Histogram) MeteredWriter {
	mw := MeteredWriter{
		Writer: writer,
		h:      h,
	}
	if r, ok := h.(Registrar); ok {
		r.Register()
	}
	return mw
}

// Write implements io.Writer interface; each write operation is timed and
// sampled in attached histogram. Samples are stored in nanoseconds.
func (mw MeteredWriter) Write(p []byte) (n int, err error) {
	var start time.Time
	if mw.h != nil {
		start = time.Now()
	}
	n, err = mw.Writer.Write(p)
	if n > 0 && mw.h != nil {
		mw.h.Update(time.Now().Sub(start).Nanoseconds())
	}
	return n, err
}

// Close implements io.Closer interface. If underlying writer implements
// io.Closer, calling this method would also close it. If attached histogram
// also implements Registrar interface, this would call its Done() method.
func (mw MeteredWriter) Close() error {
	if r, ok := mw.h.(Registrar); ok {
		r.Done()
	}
	if c, ok := mw.Writer.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// SelfCleaningHistogram wraps metrics.Histogram, adding self-cleaning feature
// if no samples were registered for a specified time. SelfCleaningHistogram
// also implements Registrar interface, call Register() method to announce
// following sample updates, call Done() after all samples were added. If no
// outstanding workers registered (for each Register() call Done() call were
// made), self-cleaning timer would start, cleaning histogram's sample pool in
// absence of Register() calls before timer fires.
type SelfCleaningHistogram struct {
	Histogram
	c, q   chan struct{}
	closed bool
	wg     sync.WaitGroup
}

// Registrar interface can be used to track object's concurrent usage.
//
// Its Register method announces goroutine's intent to use this object's
// facilities; Done method should be called when goroutine finished working with
// this object. Shutdown method stops associated background goroutines so that
// resources can be garbage collected.
//
// These methods provide a similar semantics as sync.WaitGroup's Add(1), and
// Done() methods.
type Registrar interface {
	Register()
	Done()
	Shutdown()
}

// NewSelfCleaningHistogram returns SelfCleaningHistogram wrapping specified
// histogram; its self-cleaning period set to delay.
func NewSelfCleaningHistogram(histogram Histogram, delay time.Duration) *SelfCleaningHistogram {
	h := &SelfCleaningHistogram{
		Histogram: histogram,
		c:         make(chan struct{}),
		q:         make(chan struct{}),
	}
	// make sure goroutine is started before returning
	guard := make(chan struct{})
	go h.decay(delay, guard)
	<-guard
	return h
}

// decay tracks usage of SelfCleaningHistogram, starting and stopping cleaning
// timer as needed
func (h *SelfCleaningHistogram) decay(delay time.Duration, guard chan<- struct{}) {
	var t *time.Timer
	close(guard)
	for {
		select {
		case <-h.c:
		case <-h.q:
			if t != nil {
				t.Stop()
			}
			return
		}
		if t != nil {
			t.Stop()
		}
		h.wg.Wait()
		t = time.AfterFunc(delay, h.Clear)
	}
}

// Register implements Registrar interface, using sync.WaitGroup.Add(1) for each
// call, blocking self-cleaning timer until all object's users releases it with
// Done() call.
func (h *SelfCleaningHistogram) Register() {
	h.wg.Add(1)
	select {
	case h.c <- struct{}{}:
	default:
	}
}

// Done implements Registrar interface, using sync.WaitGroup.Done() for each
// call.
func (h *SelfCleaningHistogram) Done() {
	h.wg.Done()
}

// Shutdown implements Registrar interface, it stops background goroutine. This
// method should be called as the very last method on object and needed only if
// object has to be removed and garbage collected.
func (h *SelfCleaningHistogram) Shutdown() {
	if !h.closed {
		h.closed = true
		close(h.q)
	}
}
