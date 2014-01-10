// Copyright 2014 Joshua Marsh. All rights reserved. Use of this
// source code is governed by the MIT license that can be found in the
// LICENSE file.

// Package wrapio implements wrappers for the io.Reader and io.Writer
// interfaces. These wrappers act as middlemen that allow you to do
// multiple things with a single stream of data. They are useful when
// requesting the data multiple times may be difficult or
// expensive. They also eliminate the need to track and maintain all
// of these items yourself and make functions like io.Copy and
// ioutil.ReadAll extremely useful.
package wrapio

import (
	"hash"
	"io"
	"sync"
)

// wrapfunc is the signature used by the wrap struct.
type wrapfunc func([]byte) error

// Wrap implements the io.Reader and io.Writer interface. It contains
// common simple algorithms that many of the New* functions use.
type wrap struct {
	// F is the function that is called during the Read/Write
	// process. It is called after a Read() and before a Write(). An
	// error suggests something fatal and the process should stop.
	f wrapfunc
	r io.Reader
	w io.Writer
}

// Read implements the io.Reader interface.
func (w *wrap) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	if n > 0 {
		err := w.f(p[0:n])
		if err != nil {
			return 0, err
		}
	}
	return n, err
}

// Write implements the io.Writer interface.
func (w *wrap) Write(p []byte) (int, error) {
	err := w.f(p)
	if err != nil {
		return 0, err
	}
	return w.w.Write(p)
}

// makeHashWrap Makes a closure around the given hash.Hash.
func makeHashWrap(h hash.Hash) wrapfunc {
	return func(p []byte) error {
		// Per the documentation, this never returns an error, so we can
		// safely ignore the results.
		h.Write(p)
		return nil
	}
}

// NewHashReader returns an io.Reader that wraps the given io.Reader
// with the given hash.Hash. Any Read() operations will also be
// written to the hash allowing you to simultaneously read something
// and get the hash of that thing. If either of the parameters are
// nil, nil is returned.
func NewHashReader(h hash.Hash, r io.Reader) io.Reader {
	if h == nil || r == nil {
		return nil
	}
	return &wrap{f: makeHashWrap(h), r: r}
}

// NewHashWriter returns an io.Writer that wraps the given io.Writer
// with the given hash.Hash. Any Write() operations will also be
// written to the hash allowing you to simultaneously write something
// and get the hash of that thing. If either of the parameters are
// nil, nil is returned.
func NewHashWriter(h hash.Hash, w io.Writer) io.Writer {
	if h == nil || w == nil {
		return nil
	}
	return &wrap{f: makeHashWrap(h), w: w}
}

// Stats maintains the statistics about the I/O. It is updated with
// each read/write operation. If you are accessing the values, you
// should Lock() before accessing them and Unlock() after you are done
// to prevent possible race conditions.
type Stats struct {
	sync.Mutex
	Total   int // The total number of bytes that have passed through.
	Average int // The average number of bytes read or written per call.
	Calls   int // The number of calls made to Read or Write.
}

// Wrap implements the wrap interface. It analyzes the given bytes and
// updates the statistics accordingly.
func makeStatsWrap(s *Stats) wrapfunc {
	return func(p []byte) error {
		s.Lock()
		defer s.Unlock()
		s.Total += len(p)
		s.Calls++
		s.Average = s.Total / s.Calls
		return nil
	}
}

// NewStatsReader returns an io.Reader that wraps the given io.Reader
// with the returned statistical analyzer. Any Read() operations will
// be analyzed and the statistics updated. If either of the parameters
// are nil, nil is returned.
func NewStatsReader(r io.Reader) (*Stats, io.Reader) {
	s := &Stats{}
	return s, &wrap{f: makeStatsWrap(s), r: r}
}

// NewStatsWriter returns an io.Writer that wraps the given io.Writer
// with the returned statistical analyzer. Any Write() operations will
// be analyzed and the statistics updated. If either of the parameters
// are nil, nil is returned.
func NewStatsWriter(w io.Writer) (*Stats, io.Writer) {
	s := &Stats{}
	return s, &wrap{f: makeStatsWrap(s), w: w}
}
