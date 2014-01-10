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

// NewFuncReader returns an io.Reader that wraps the given io.Reader
// with the given function. Any Read() operations will run through the
// translation function before being returned. If the function
// encounters something fatal, an error can be returned and that will
// be returned for the Read(). If either of the parameters are nil,
// nil is returned.
func NewFuncReader(f func([]byte) error, r io.Reader) io.Reader {
	if f == nil || r == nil {
		return nil
	}
	return &wrap{f: func(p []byte) error {
		return f(p)
	}, r: r}
}

// NewFuncWriter returns an io.Writer that wraps the given io.Writer
// with the given function. Any Write() operations will run through
// the translation function before being written. If the function
// encounters something fatal, an error can be returned and that will
// be returned for the Write() without data being written to the
// original io.Writer. If either of the parameters are nil, nil is
// returned.
func NewFuncWriter(f func([]byte) error, w io.Writer) io.Writer {
	if f == nil || w == nil {
		return nil
	}
	return &wrap{f: func(p []byte) error {
		return f(p)
	}, w: w}
}

// NewHashReader returns an io.Reader that wraps the given io.Reader
// with the given hash.Hash. Any Read() operations will also be
// written to the hash allowing you to simultaneously read something
// and get the hash of that thing. If either of the parameters are
// nil, nil is returned.
func NewHashReader(h hash.Hash, r io.Reader) io.Reader {
	if h == nil {
		return nil
	}
	return NewFuncReader(func(p []byte) error {
		h.Write(p)
		return nil
	}, r)
}

// NewHashWriter returns an io.Writer that wraps the given io.Writer
// with the given hash.Hash. Any Write() operations will also be
// written to the hash allowing you to simultaneously write something
// and get the hash of that thing. If either of the parameters are
// nil, nil is returned.
func NewHashWriter(h hash.Hash, w io.Writer) io.Writer {
	if h == nil {
		return nil
	}
	return NewFuncWriter(func(p []byte) error {
		h.Write(p)
		return nil
	}, w)
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
	return s, NewFuncReader(makeStatsWrap(s), r)
}

// NewStatsWriter returns an io.Writer that wraps the given io.Writer
// with the returned statistical analyzer. Any Write() operations will
// be analyzed and the statistics updated. If either of the parameters
// are nil, nil is returned.
func NewStatsWriter(w io.Writer) (*Stats, io.Writer) {
	s := &Stats{}
	return s, NewFuncWriter(makeStatsWrap(s), w)
}
