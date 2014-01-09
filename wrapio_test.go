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
)

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
		t.Errorf("nil io.Reader didn't return nil.")
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
	if NewHashReader(tests[0].hash, nil) != nil {
		t.Errorf("nil io.Reader didn't return nil.")
	}
	if NewHashReader(nil, strings.NewReader("")) != nil {
		t.Errorf("nil io.Reader didn't return nil.")
	}
}
