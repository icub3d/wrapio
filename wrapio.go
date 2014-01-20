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
	"fmt"
	"hash"
	"io"
	"sync"
)

// Wrap implements the io.Closer, io.Reader, and io.Writer
// interface. It contains common simple algorithms that many of the
// New* functions use.
type wrap struct {
	handler func([]byte)
	r       io.Reader
	w       io.Writer
}

// Read implements the io.Reader interface.
func (w *wrap) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	if n > 0 {
		w.handler(p[:n])
	}
	return n, err
}

// Write implements the io.Writer interface.
func (w *wrap) Write(p []byte) (int, error) {
	w.handler(p)
	return w.w.Write(p)
}

// NewFuncReader returns an io.Reader that wraps the given io.Reader
// with the given handler. Any Read() operations that read at least
// one byte will run through the handler before being returned. If
// either of the parameters are nil, nil is returned.
func NewFuncReader(handler func([]byte), r io.Reader) io.Reader {
	if handler == nil || r == nil {
		return nil
	}
	return &wrap{handler: handler, r: r}
}

// NewFuncWriter returns an io.Writer that wraps the given io.Writer
// with the given handler. Any Write() operations will run through the
// handler before being written. If either of the parameters are nil,
// nil is returned.
//
// Since the handler is called with all the data before the write, if
// an error occurs and not all of it is written, sending that data
// again will cause it to be sent to the handler again as well. This
// is a special case because most errors on write are fatal, but in
// cases where writing will continue, this must be taken into account.
func NewFuncWriter(handler func([]byte), w io.Writer) io.Writer {
	if handler == nil || w == nil {
		return nil
	}
	return &wrap{handler: handler, w: w}
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
	return NewFuncReader(func(p []byte) {
		h.Write(p)
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
	return NewFuncWriter(func(p []byte) {
		h.Write(p)
	}, w)
}

// Stats maintains the statistics about the I/O. It is updated with
// each read/write operation. If you are accessing the values, you
// should Lock() before accessing them and Unlock() after you are done
// to prevent possible race conditions.
type Stats struct {
	sync.Mutex
	Total   int     // The total number of bytes that have passed through.
	Average float64 // The average number of bytes read or written per call.
	Calls   int     // The number of calls made to Read or Write.
}

// String implements the fmt.Stringer interface.
func (s Stats) String() string {
	return fmt.Sprintf("[Total: %d, Average: %f, Calls: %d]",
		s.Total, s.Average, s.Calls)
}

func (s *Stats) update(p []byte) {
	s.Lock()
	defer s.Unlock()
	s.Total += len(p)
	s.Calls++
	s.Average = float64(s.Total / s.Calls)
}

// NewStatsReader returns an io.Reader that wraps the given io.Reader
// with the returned statistical analyzer. Any Read() operations will
// be analyzed and the statistics updated. If either of the parameters
// are nil, nil is returned.
func NewStatsReader(r io.Reader) (*Stats, io.Reader) {
	s := &Stats{}
	return s, NewFuncReader(s.update, r)
}

// NewStatsWriter returns an io.Writer that wraps the given io.Writer
// with the returned statistical analyzer. Any Write() operations will
// be analyzed and the statistics updated. If either of the parameters
// are nil, nil is returned.
func NewStatsWriter(w io.Writer) (*Stats, io.Writer) {
	s := &Stats{}
	return s, NewFuncWriter(s.update, w)
}

type block struct {
	r    io.Reader
	w    io.Writer
	size int
	buf  []byte
	err  error // The non-nil error from the last Read().

}

// Read implements the io.Reader interface.
func (b *block) Read(p []byte) (int, error) {
	// If we've finished reading, we can quit.
	if b.err != nil && len(b.buf) == 0 {
		return 0, b.err
	}
	// We'll only fill p with full blocks.
	n := (len(p) / b.size) * b.size
	if n == 0 {
		return 0, nil
	}
	if b.err == nil {
		// Fill p temporarily and append it to our buffer.
		l, err := b.r.Read(p)
		b.err = err
		b.buf = append(b.buf, p[:l]...)
	}
	// If the size of p if bigger than what we have, only pull the
	// number of blocks that is in the buffer.
	if n > len(b.buf) {
		n = (len(b.buf) / b.size) * b.size
	}
	// If we've reached the end and we don't have a full block, make it
	// our last send.
	if b.err != nil && n == 0 {
		n = len(b.buf)
	}
	// Copy what we have to p.
	copy(p, b.buf[:n])
	copy(b.buf, b.buf[n:])
	b.buf = b.buf[:len(b.buf)-n]
	return n, nil
}

// Write implements the io.Writer interface.
func (b *block) Write(p []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	// We should first append p to our buffer.
	b.buf = append(b.buf, p...)
	// Write out any whole blocks.
	l := (len(b.buf) / b.size) * b.size
	if l > 0 {
		n, err := b.w.Write(b.buf[:l])
		// Move the unwritten portion to the beginning of the buffer and
		// reslice the buffer.
		copy(b.buf, b.buf[l:])
		b.buf = b.buf[:len(b.buf)-l]
		// In the error case, we want to report the actual written
		// information.
		if err != nil {
			b.err = err
			return n, err
		}
	}
	return len(p), nil
}

// Close implements the io.Closer interface.
func (b *block) Close() error {
	if b.err != nil {
		return b.err
	}
	// Write out any remaining data (which wouldn't have fit into a
	// block).
	if len(b.buf) > 0 {
		_, err := b.w.Write(b.buf)
		return err
	}
	return nil
}

// NewBlockReader returns a reader that sends data to the given reader
// in blocks that are a multiple of size. The one exception of this is
// the last Read() in which there may be an incomplete block. If p in
// Read(p) is not the length of a block, no data will be written to it
// (i.e it will return 0, nil). This may cause an infinite loop if you
// never give a slice larger than size.
func NewBlockReader(size int, r io.Reader) io.Reader {
	if r == nil || size < 1 {
		return nil
	}
	return &block{r: r, size: size}
}

// NewBlockWriter returns a writer that sends data to the given writer
// in blocks that are a multiple of size. Writes may be held if there
// is not enough data to Write() a complete block. To adhere to the
// io.Writer documentation though, the returned number of written
// bytes will always be the length of the given slice unless an error
// occurred in writing.
//
// Because it is impossible to tell when writing is completed, the
// returned writer is also a closer. The close operation should be
// called to flush out the remaining unwritten data that did not fit
// into a block size.
func NewBlockWriter(size int, w io.Writer) io.WriteCloser {
	if w == nil || size < 1 {
		return nil
	}
	return &block{w: w, size: size}
}
