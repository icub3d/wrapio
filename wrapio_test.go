// Copyright 2014 Joshua Marsh. All rights reserved. Use of this
// source code is governed by the MIT license that can be found in the
// LICENSE file.

package wrapio

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"testing/iotest"
)

func TestWrap(t *testing.T) {
	// We basically just need to test the error cases.
	expected := fmt.Errorf("test")
	w := wrap{
		f: func(p []byte) error {
			return expected
		},
		r: strings.NewReader("test"),
		w: ioutil.Discard,
	}
	p := make([]byte, 32)
	if n, err := w.Read(p); n != 0 || err != expected {
		t.Errorf("Expected 0 %v with Read() wrapfunc but got: %v %",
			expected, n, err)
	}
	if n, err := w.Write(p); n != 0 || err != expected {
		t.Errorf("Expected 0 %v with Write() wrapfunc but got: %v %",
			expected, n, err)
	}
}

func Example_hashes() {
	// We'll read from this using io.Copy.
	r := strings.NewReader("This is the sample data that we are going to test with.")

	// Generate different hashes on each side.
	m := md5.New()
	s := sha256.New()

	// Create our wrappers and use them.
	hr := NewHashReader(m, r)
	hw := NewHashWriter(s, ioutil.Discard)
	io.Copy(hw, hr)

	// Use the Sum()s which in this case we'll just print it out.
	fmt.Println(hex.EncodeToString(m.Sum(nil)))
	fmt.Println(hex.EncodeToString(s.Sum(nil)))

	// Output:
	// 9bd2f8a51a7745e0e0af586736f93944
	// 52b846d6fedeb0a90acec7ce09f7d590ec4db0e5bd1884bc74c1d81e3c00b471
}

func TestNewHashReader(t *testing.T) {
	tests := []struct {
		data     string
		expected string
		hash     hash.Hash
	}{
		{
			data:     "this is a test.",
			expected: "09cba091df696af91549de27b8e7d0f6",
			hash:     md5.New(),
		},
		{
			data:     "I eat pizza for breakfast and there is nothing you can do to stop me.",
			expected: "9df01f46c48c1be6507a14b73da3ad7007f7815b0ddae3465707931403f32e46",
			hash:     sha256.New(),
		},
	}
	for k, test := range tests {
		sr := strings.NewReader(test.data)
		hr := NewHashReader(test.hash, sr)
		ioutil.ReadAll(hr)
		s := hex.EncodeToString(test.hash.Sum(nil))
		if s != test.expected {
			t.Errorf("Test %v: unexpected Sum(), got vs expected:\n%v\n%v",
				k, s, test.expected)
		}
	}
	// Test the special error cases.
	if NewHashReader(tests[0].hash, nil) != nil {
		t.Errorf("nil io.Reader didn't return nil.")
	}
	if NewHashReader(nil, strings.NewReader("")) != nil {
		t.Errorf("nil hash didn't return nil.")
	}
}

func TestNewHashWriter(t *testing.T) {
	tests := []struct {
		data     string
		expected string
		hash     hash.Hash
	}{
		{
			data:     "this is a test.",
			expected: "09cba091df696af91549de27b8e7d0f6",
			hash:     md5.New(),
		},
		{
			data:     "I eat pizza for breakfast and there is nothing you can do to stop me.",
			expected: "9df01f46c48c1be6507a14b73da3ad7007f7815b0ddae3465707931403f32e46",
			hash:     sha256.New(),
		},
	}
	for k, test := range tests {
		sr := strings.NewReader(test.data)
		hw := NewHashWriter(test.hash, ioutil.Discard)
		io.Copy(hw, sr)
		s := hex.EncodeToString(test.hash.Sum(nil))
		if s != test.expected {
			t.Errorf("Test %v: unexpected Sum(), got vs expected:\n%v\n%v",
				k, s, test.expected)
		}
	}
	// Test the special error cases.
	if NewHashWriter(tests[0].hash, nil) != nil {
		t.Errorf("nil io.Writer didn't return nil.")
	}
	if NewHashWriter(nil, ioutil.Discard) != nil {
		t.Errorf("nil hash did't return nil.")
	}
}

func Example_stats() {
	// We'll read from this using io.Copy.
	sr := strings.NewReader("This is the sample data that we are going to test with.")

	// Create our wrappers and use them.
	s, r := NewStatsReader(iotest.OneByteReader(sr))
	io.Copy(ioutil.Discard, r)

	// Print out the statistics.
	s.Lock()
	defer s.Unlock()
	fmt.Println(s.Total)
	fmt.Println(s.Calls)
	fmt.Println(s.Average)

	// Output:
	// 55
	// 55
	// 1
}

func TestNewStatsReader(t *testing.T) {
	tests := []struct {
		data     string
		expected []int
	}{
		{
			data:     "this is a test.",
			expected: []int{15, 1, 15},
		},
	}
	for k, test := range tests {
		sr := strings.NewReader(test.data)
		s, hr := NewStatsReader(sr)
		ioutil.ReadAll(hr)
		if s.Total != test.expected[0] || s.Calls != test.expected[1] ||
			s.Average != test.expected[2] {
			t.Errorf("Test %v: unexpected stats, got vs expected:\n%v\n%v",
				k, s, test.expected)
		}
	}
}

func TestNewStatsWriter(t *testing.T) {
	tests := []struct {
		data     string
		expected []int
	}{
		{
			data:     "this is a test.",
			expected: []int{15, 1, 15},
		},
	}
	for k, test := range tests {
		sr := strings.NewReader(test.data)
		s, hw := NewStatsWriter(ioutil.Discard)
		io.Copy(hw, sr)
		if s.Total != test.expected[0] || s.Calls != test.expected[1] ||
			s.Average != test.expected[2] {
			t.Errorf("Test %v: unexpected stats, got vs expected:\n%v\n%v",
				k, s, test.expected)
		}
	}
}
