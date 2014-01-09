// Copyright 2014 Joshua Marsh. All rights reserved. Use of this
// source code is governed by the MIT license that can be found in the
// LICENSE file.

// Package wrapio implements wrappers for the io.Reader and io.Writer
// interfaces. These wrappers act as middlemen that allow you to do
// multiple things with a single stream of data. They are useful when
// requesting the data multiple times may be difficult or
// expensive. They also elinimate the need to track and maintain all
// of these items yourself and make functions like io.Copy and
// ioutil.ReadAll extremely useful.
package wrapio

import (
	"hash"
	"io"
)

// Hasher is the underlying implementation of the
// Hash(Reader/Writer). It implements the interfaces which just push
// the same data through, but sends the data to the hash before doing
// it.
type hasher struct {
	h hash.Hash
	r io.Reader
	w io.Writer
}

// Read implements the io.Reader interface.
func (hr hasher) Read(p []byte) (int, error) {
	n, err := hr.r.Read(p)
	if n > 0 {
		// Per the documentation, this never returns an error, so we can
		// safely ignore the results.
		hr.h.Write(p[0:n])
	}
	return n, err
}

// Write implements the io.Writer interface.
func (hr hasher) Write(p []byte) (n int, err error) {
	// Per the documentation, this never returns an error, so we can
	// safely ignore the results.
	hr.h.Write(p)
	return hr.w.Write(p)
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
	return &hasher{h: h, r: r}
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
	return &hasher{h: h, w: w}
}
