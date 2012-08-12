/*
This package implements the efficient Boyer-Moore string search algorithm. This central
algorithm is a translation of the Java source code that appeared on the Wikipedia page
describing the algorithm (captured 2012-08-07).
See: http://en.wikipedia.org/wiki/Boyer-Moore_string_search_algorithm .

Copyright © 2012 by J. E. Ivancich.
This work is licensed under a Creative Commons Attribution-ShareAlike 3.0 Unported License.
See: http://creativecommons.org/licenses/by-sa/3.0/
*/
package substr

import (
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	byteCount   = 1 + math.MaxUint8
	buffSize    = 4 * 1024 // 4KB
	outChanSize = 64
	errorOffset = math.MaxUint32
)

// The error returned if an empty needle is provided to one of the search functions.
var ErrEmptyNeedle = errors.New("boyer_moore: the needle may not be empty")


// A processed version of the needle in which various tables have been
// created that make the searching efficient (via Boyer-Moore algorithm).
// If one is searching multiple blocks of data, it's better to calculate
// the Needle once and use functions that take a pointer to it as a
// parameter to avoid repeating the pre-processing.
type Needle struct {
	bytes       []byte
	length      uint32
	charTable   [byteCount]uint32
	offsetTable []uint32
}

// Return a pre-processed Needle given an array of bytes.
func NewNeedleBytes(needle []byte) *Needle {
	return &Needle{
		bytes:       needle,
		length:      uint32(len(needle)),
		charTable:   makeCharTable(needle),
		offsetTable: makeOffsetTable(needle)}
}

// Return a pre-processed Needle given a string.
func NewNeedleStr(needle string) *Needle {
	return NewNeedleBytes([]byte(needle))
}

// A result from a search. It either contains an error, if Error is not nil.
// If Error is nil, then Offset contains the offset of a match within the
// data searched.
type Result struct {
	Offset uint32
	Error  error
}

// Returns a string version of a Result, which can be used in testing.
func (r *Result) string() string {
	return fmt.Sprintf("Result{%v, %v}", r.Offset, r.Error)
}

// Searches for needle within haystack. Returns any=true if any match is
// found; firstOffset is location of first match; and e is any error that
// occurred.
func IndexWithinReaderStr(haystack io.Reader, needle string) (any bool, firstOffset uint32, e error) {
	return IndexWithinReaderNeedle(haystack, NewNeedleStr(needle))
}

// Searches for needle within haystack. Returns any=true if any match is
// found; firstOffset is location of first match; and e is any error that
// occurred.
func IndexesWithinReaderStr(haystack io.Reader, needle string) <-chan Result {
	return IndexesWithinReaderNeedle(haystack, NewNeedleStr(needle))
}

// Searches for needle within haystack. Returns any=true if any match is
// found; firstOffset is location of first match; and e is any error that
// occurred.
func IndexWithinReaderBytes(haystack io.Reader, needle []byte) (any bool, firstOffset uint32, e error) {
	return IndexWithinReaderNeedle(haystack, NewNeedleBytes(needle))
}

// Searches for needle within haystack. Returns any=true if any match is
// found; firstOffset is location of first match; and e is any error that
// occurred.
func IndexesWithinReaderBytes(haystack io.Reader, needle []byte) <-chan Result {
	return IndexesWithinReaderNeedle(haystack, NewNeedleBytes(needle))
}

// Searches for needle within haystack. Returns any=true if any match is
// found; firstOffset is location of first match; and e is any error that
// occurred.
func IndexWithinReaderNeedle(haystack io.Reader, needle *Needle) (any bool, firstOffset uint32, e error) {
	return returnOne(indexesWithinReaderHelp(haystack, needle, true))
}

// Searches for needle within haystack. Returns any=true if any match is
// found; firstOffset is location of first match; and e is any error that
// occurred.
func IndexesWithinReaderNeedle(haystack io.Reader, needle *Needle) <-chan Result {
	return indexesWithinReaderHelp(haystack, needle, false)
}

// Searches for needle within haystack. stopAtFirst determines whether
// it keeps searching once a match is found. The results are sent on
// the channel returned.
func indexesWithinReaderHelp(haystack io.Reader, needle *Needle, stopAtFirst bool) <-chan Result {
	out := make(chan Result, outChanSize)

	go func() {
		if needle.length == 0 {
			out <- Result{errorOffset, ErrEmptyNeedle}
			close(out)
		}

		offset := uint32(0)
		var buffer [buffSize]byte
		used := uint32(0)
		done := false

	outer:
		for {
			count, err := haystack.Read(buffer[used:])
			if count > 0 {
				used += uint32(count)
				if used < buffSize {
					continue
				}
			} else if err != io.EOF {
				out <- Result{errorOffset, err}
				break outer
			} else {
				done = true
			}

			haystackSkip := uint32(0)
			for {
				index := indexOfHelper(buffer[0:used], needle, used, haystackSkip)
				if index == errorOffset {
					break
				}
				out <- Result{offset + index, nil}
				if stopAtFirst {
					break outer
				}
				haystackSkip = index + 1
			}

			if done {
				break
			}

			copy(buffer[0:], buffer[used-needle.length+1:used])
			offset += used - needle.length - 1
			used = needle.length - 1
		}

		close(out)
	}()

	return out
}

/*
Returns the index of the first match of needle within haystack.
If no matches are found, returns -1. Parameter needle must not be empty.
*/
func IndexOfStr(haystack, needle string) (any bool, firstOffset uint32, e error) {
	return IndexOf([]byte(haystack), []byte(needle))
}

/*
Returns the indexes of all matches of needle within haystack.
If no matches are found returns a slice of size 0.
Parameter needle must not be empty.
*/
func IndexesOfStr(haystack, needle string) <-chan Result {
	return IndexesOf([]byte(haystack), []byte(needle))
}

// Returns the index of the first match of needle within haystack.
// If no matches are found, returns -1. Parameter needle must not be empty.
func IndexOf(haystack, needleBytes []byte) (any bool, firstOffset uint32, e error) {
	out := make(chan Result, outChanSize)

	go func() {
		needle := NewNeedleBytes(needleBytes)
		if needle.length == 0 {
			out <- Result{errorOffset, ErrEmptyNeedle}
		}

		haystackLen := uint32(len(haystack))

		index := indexOfHelper(haystack, needle, haystackLen, 0)
		if index != errorOffset {
			out <- Result{index, nil}
		}
		close(out)
	}()

	return returnOne(out)
}

// Returns the indexes of all matches of needle within haystack.
// If no matches are found returns a slice of size 0.
// Parameter needle must not be empty.
func IndexesOf(haystack, needleBytes []byte) <-chan Result {
	out := make(chan Result, 64)

	go func() {
		needle := NewNeedleBytes(needleBytes)
		if needle.length == 0 {
			out <- Result{errorOffset, ErrEmptyNeedle}
		}

		haystackLen := uint32(len(haystack))
		var haystackStartingIndex uint32 = 0

		for {
			index := indexOfHelper(haystack, needle, haystackLen, haystackStartingIndex)
			if index == errorOffset {
				break
			}
			out <- Result{index, nil}
			haystackStartingIndex = index + 1
		}

		close(out)
	}()

	return out
}

// Returns the next found index of needle within haystack after skipping
// haystackSkip positions. Returns errorOffset if no matches are found.
func indexOfHelper(haystack []byte, needle *Needle, haystackLen, haystackSkip uint32) uint32 {
	for i := needle.length - 1 + haystackSkip; i < haystackLen; {
		var j uint32
		for j = needle.length - 1; needle.bytes[j] == haystack[i]; i, j = i-1, j-1 {
			if j == 0 {
				return i
			}
		}

		i += maxUint32(needle.offsetTable[needle.length-1-j], needle.charTable[haystack[i]])
	}

	return errorOffset
}

// Makes the jump table based on the mismatched character information.
func makeCharTable(needle []byte) (table [byteCount]uint32) {
	needleLen := len(needle)

	for i := 0; i < byteCount; i++ {
		table[i] = uint32(needleLen)
	}

	for i := 0; i < needleLen-1; i++ {
		table[needle[i]] = uint32(needleLen - 1 - i)
	}

	return
}

// Makes the jump table based on the scan offset which mismatch occurs.
func makeOffsetTable(needle []byte) (table []uint32) {
	needleLen := len(needle)
	table = make([]uint32, needleLen)
	lastPrefixPosition := needleLen
	for i := int(needleLen - 1); i >= 0; i-- {
		if isPrefix(needle, i+1) {
			lastPrefixPosition = i + 1
		}
		table[needleLen-1-i] = uint32(lastPrefixPosition - i + needleLen - 1)
	}
	for i := 0; i < needleLen-1; i++ {
		slen := suffixLength(needle, i)
		table[slen] = uint32(needleLen - 1 - i + slen)
	}
	return
}

// Is needle[p:end] a prefix of needle?
func isPrefix(needle []byte, p int) bool {
	needleLen := len(needle)
	for i, j := p, 0; i < needleLen; i, j = i+1, j+1 {
		if needle[i] != needle[j] {
			return false
		}
	}
	return true
}

// Returns the maximum length of the substring ends at p and is a suffix.
func suffixLength(needle []byte, p int) int {
	length := 0
	for i, j := p, len(needle)-1; i >= 0 && needle[i] == needle[j]; i, j = i-1, j-1 {
		length += 1
	}
	return length
}


// Given a channel of Result/s returns the first Result and insures that no
// more are returned (if there is another result, it panics).
func returnOne(c <-chan Result) (any bool, firstOffset uint32, e error) {
	if result, ok := <-c; ok {
		if result.Error == nil {
			if _, ok = <-c; ok {
				panic("got more than one result")
			}
			return true, result.Offset, nil
		} else {
			if _, ok = <-c; ok {
				panic("got more than one result")
			}
			return false, result.Offset, result.Error
		}
	}
	return false, 0, nil
}

// Returns the larger of its two (unsigned) parameters.
func maxUint32(i, j uint32) uint32 {
	if i > j {
		return i
	}
	return j
}
