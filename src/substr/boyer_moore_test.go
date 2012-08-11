/*
This file includes a number of tests for the substr package, a package to
implement the Boyer-Moore string search algorithm.

Copyright © 2012 by J. E. Ivancich.
This work is licensed under a Creative Commons Attribution-ShareAlike 3.0 Unported License.
See: http://creativecommons.org/licenses/by-sa/3.0/
*/

package substr

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// convert a channel of results into an array of offsets plus any error
func convert(in <-chan Result) ([]uint32, error) {
	results := make([]uint32, 0)
	var e error

	for r := range in {
		if r.Error != nil {
			e = r.Error
		} else {
			results = append(results, r.Offset)
		}
	}

	return results, e
}

// expect 0 results to come in through a channel
func expect0(t *testing.T, in <-chan Result, notation interface{}) {
	r, ok := <-in
	if ok {
		t.Error(fmt.Sprintf("expected 0 match, got at least 1; %s (note: %v)", r, notation))
	}
}

// expect 1 result to come in through a channel
func expect1(t *testing.T, in <-chan Result, value uint32, notation interface{}) {
	var r Result
	var ok bool

	r, ok = <-in
	if !ok {
		t.Error(fmt.Sprintf("got 0 matches, expected 1 (note: %v)", notation))
		return
	}
	if r.Error != nil {
		t.Error(fmt.Sprintf("expected no error got %v (note: %v)", r.Error, notation))
	}
	if r.Offset != value {
		t.Error(fmt.Sprintf("expected %d got %d (note: %v)", value, r.Offset, notation))
	}

	r, ok = <-in
	if ok {
		t.Error(fmt.Sprintf("expected 1 match, got more than 1; %s (note: %v)", r, notation))
		return
	}
}

func got1(t *testing.T, found bool, offset uint32, err error, expectedOffset uint32, notation interface{}) {
	if !found {
		t.Error(fmt.Sprintf("got 0 matches, expected 1 (note: %v)", notation))
	} else if offset != expectedOffset {
		t.Error(fmt.Sprintf("expected offset %d got %d (note: %v)", expectedOffset, offset, notation))
	}
	if err != nil {
		t.Error(fmt.Sprintf("got unexpected error %s (note: %v)", err, notation))
	}
}

func got0(t *testing.T, found bool, offset uint32, err error, notation interface{}) {
	if found {
		t.Error(fmt.Sprintf("got a match (%d), expected none (note: %v)", offset, notation))
	}
	if err != nil {
		t.Error(fmt.Sprintf("got unexpected error %s (note: %v)", err, notation))
	}
}

func gotError(t *testing.T, found bool, offset uint32, err error, expectedError error, notation interface{}) {
	if found {
		t.Error(fmt.Sprintf("got a match (%d), expected none (note: %v)", offset, notation))
	}
	if err == nil {
		t.Error(fmt.Sprintf("expected error but did not get one (note: %v)", notation))
	} else if err != expectedError {
		t.Error(fmt.Sprintf("expected error %s but got %s (note: %v)", expectedError, err, notation))
	}
}

func expectList(t *testing.T, in <-chan Result, values []uint32, notation interface{}) {
	var r Result
	var ok bool

	for i := 0; i < len(values); i++ {
		r, ok = <-in
		if !ok {
			t.Error(fmt.Sprintf("got %d matches, expected %d (note: %v)", i, len(values), notation))
			break
		}
		if r.Error != nil {
			t.Error(fmt.Sprintf("expected no error got %v (note: %v)", r.Error, notation))
		}
		if r.Offset != values[i] {
			t.Error(fmt.Sprintf("expected %d got %d (note: %v)", values[i], r.Offset, notation))
		}
	}

	r, ok = <-in
	if ok {
		t.Error(fmt.Sprintf("expected no more values, got Result{%v, %v}", r.Offset, r.Error))
	}
}

func expectError(t *testing.T, in <-chan Result, err error, notation interface{}) {
	var r Result
	var ok bool

	r, ok = <-in
	if !ok {
		t.Error("got 0 inputs, expected 1")
		return
	}
	if r.Error == nil {
		t.Error(fmt.Sprintf("expected error %v, got none", err))
	} else if r.Error != err {
		t.Error(fmt.Sprintf("expected error %v, got %v", err, r.Error))
	}

	r, ok = <-in
	if ok {
		t.Error(fmt.Sprintf("expected 1 error and nothing after, got %s (note: %v)", r, notation))
		return
	}
}

func expectCount(t *testing.T, in <-chan Result, count uint32, notation interface{}) {
	var r Result
	var ok bool

	for i := uint32(0); i < count; i++ {
		r, ok = <-in
		if !ok {
			t.Error(fmt.Sprintf("got %d matches, expected %d (note: %v)", i, count, notation))
			return
		}
		if r.Error != nil {
			t.Error(fmt.Sprintf("expected no error got %v (note: %v)", r.Error, notation))
			return
		}
	}

	r, ok = <-in
	if ok {
		t.Error(fmt.Sprintf("expected no more values, got %s (note: %v)", r, notation))
	}
}

func TestFirstFound(t *testing.T) {
	found, offset, err := IndexOfStr("here is a simple example", "example")
	got1(t, found, offset, err, 17, "TestFirstFound")
}

func TestFirstEmpty(t *testing.T) {
	found, offset, err := IndexOfStr("here is a simple example", "")
	gotError(t, found, offset, err, ErrEmptyNeedle, "TestFirstEmpty")
}

func TestFirstNotFound(t *testing.T) {
	found, offset, err := IndexOfStr("here is a simple example", "axample")
	got0(t, found, offset, err, "TestFirstNotFound")
}

func TestFirstOfMany(t *testing.T) {
	found, offset, err := IndexOfStr("to be or not to be, that is the becoming question", "be")
	got1(t, found, offset, err, 3, "TestFirstOfMany")
}

func TestAllFound(t *testing.T) {
	c := IndexesOfStr("here is a simple example", "example")
	expect1(t, c, 17, "TestAllFound")
}

func TestAllEmpty(t *testing.T) {
	c := IndexesOfStr("here is a simple example", "")
	expectError(t, c, ErrEmptyNeedle, "TestAllEmpty")
}

func TestAllNotFound(t *testing.T) {
	c := IndexesOfStr("here is a simple example", "axample")
	expect0(t, c, "TestAllNotFound")
}

func TestAllOfMany(t *testing.T) {
	c := IndexesOfStr("to be or not to be, that is the becoming question", "be")
	expectList(t, c, []uint32{3, 16, 32}, "TestAllOfMany")
}

func TestAllOfOverlapping(t *testing.T) {
	c := IndexesOfStr("many bananas", "ana")
	expectList(t, c, []uint32{6, 8}, "TestAllOfOverlapping")
}

func TestAllOfOverlapping2(t *testing.T) {
	c := IndexesOfStr("abcaaadeaaaaf", "aa")
	expectList(t, c, []uint32{3, 4, 8, 9, 10}, "TestAllOfOverlapping2")
}

func TestSmallReader(t *testing.T) {
	r := strings.NewReader("to be or not to be, that is the becoming question")
	c := IndexesWithinReaderStr(r, "be")
	expectList(t, c, []uint32{3, 16, 32}, "TestSmallReader")
}

func TestHugeReaderAll(t *testing.T) {
	functions := []func(int) (*bytes.Buffer, string, uint32){prepBuffer1, prepBuffer2, prepBuffer3}
	for funcIndex, function := range functions {
		buffer, needle, expect := function(9 * 1024)
		r := bytes.NewReader(buffer.Bytes())
		c := IndexesWithinReaderStr(r, needle)
		expectCount(t, c, expect, funcIndex)
	}
}

func TestHugeReaderFirst(t *testing.T) {
	functions := []func(int) (*bytes.Buffer, string, uint32){prepBuffer1, prepBuffer2, prepBuffer3}
	for _, function := range functions {
		buffer, needle, expect := function(9 * 1024)
		r := bytes.NewReader(buffer.Bytes())
		found, offset, err := IndexWithinReaderStr(r, needle)
		if expect == 0 && found {
			t.Error(fmt.Sprintf("expected no results, but got one -- %d", offset))
		} else if expect != 0 && !found {
			t.Error("expected a result, but got none")
		}
		if err != nil {
			t.Error(fmt.Sprintf("expected a result, but got an error %s", err))
		}
	}
}

func prepBuffer1(size int) (*bytes.Buffer, string, uint32) {
	buffer := new(bytes.Buffer)
	portion := "come to become a believer in x comedy to be"
	count := size / len(portion)
	for c := count; c > 0; c-- {
		buffer.WriteString(portion)
	}
	return buffer, "become", uint32(2*count - 1)
}

func prepBuffer2(size int) (*bytes.Buffer, string, uint32) {
	buffer := new(bytes.Buffer)
	portion := "a"
	count := size / len(portion)
	for c := count; c > 0; c-- {
		buffer.WriteString(portion)
	}
	return buffer, "aaa", uint32(count - 2)
}

func prepBuffer3(size int) (*bytes.Buffer, string, uint32) {
	buffer := new(bytes.Buffer)
	portion := "to be or not to be that is the question"
	count := size / len(portion)
	for c := count; c > 0; c-- {
		buffer.WriteString(portion)
	}
	return buffer, "unto", uint32(0)
}
