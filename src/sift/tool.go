/*
This file implements a command-line tool, along the lines of grep, that will
search a file (or stdin) for a substring. It uses the substr package, which
implements the Boyer-Moore string search algorithm.

Copyright © 2012 by J. E. Ivancich.
This work is licensed under a Creative Commons Attribution-ShareAlike 3.0 Unported License.
See: http://creativecommons.org/licenses/by-sa/3.0/
*/
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"substr"
)

const (
	status_found       = 0
	status_none_found  = 1
	status_fatal_error = 2
)

type byteArrayArg []byte

var needleString *string = flag.String("t", "", "text to look for within input(s)")
var findAll *bool = flag.Bool("a", false, "display all matching offsets")
var recursive *bool = flag.Bool("r", false, "recursively descend directories")
var displayCount *bool = flag.Bool("c", false, "display count of matches")
var quiet *bool = flag.Bool("q", false, "quiet; exit immediatly with status 0 if any matches found")
var processStdin *bool = flag.Bool("stdin", false, "process stdin as one of the inputs")

var needleBytes byteArrayArg
var needle *substr.Needle

func charToValue(b byte) (byte, error) {
	if b >= '0' && b <= '9' {
		return b - '0', nil
	} else if b >= 'a' && b <= 'f' {
		return b - 'a' + 10, nil
	} else if b >= 'A' && b <= 'F' {
		return b - 'A' + 10, nil
	}
	return 0, fmt.Errorf("%q is not a valid hex character", b)
}

func (n *byteArrayArg) Set(value string) error {
	l := len(value)
	if l%2 != 0 {
		return errors.New("must specify an even number of (hex) characters to specify a byte sequence")
	}
	*n = make([]byte, 0, l/2)
	for i := 0; i < l; i += 2 {
		var err error
		var v1, v2 byte
		b1 := value[i]
		b2 := value[i+1]
		if v1, err = charToValue(b1); err != nil {
			return err
		}
		if v2, err = charToValue(b2); err != nil {
			return err
		}

		*n = append(*n, v1*16+v2)
	}

	return nil
}

func (n *byteArrayArg) String() string {
	var buf bytes.Buffer
	for _, b := range *n {
		buf.WriteString(fmt.Sprintf("%02X", b))
	}
	return buf.String()
}

func processReader(accumulatedPath string, in io.Reader, in_size int64) {
	if *displayCount {
		count := findCount(accumulatedPath, substr.IndexesWithinReaderNeedle(in, needle))
		fmt.Printf("%s: %d\n", accumulatedPath, count)
	} else if *findAll {
		count := 0
		for result := range substr.IndexesWithinReaderNeedle(in, needle) {
			if count == 0 {
				fmt.Printf("%s:\n", accumulatedPath)
			}
			count++
			if result.Error != nil {
				myError("    error: %s", result.Error)
			} else {
				fmt.Printf("    match %3d at offset %*d\n", count, calcWidth(in_size), result.Offset)
			}
		}
	} else {
		found, offset, err := substr.IndexWithinReaderNeedle(in, needle)
		if err != nil {
			myError("%s: error -- %s", accumulatedPath, err)
		} else if found {
			if *quiet {
				os.Exit(status_found)
			} else {
				fmt.Printf("%s: first offset %d\n", accumulatedPath, offset)
			}
		}
	}
}

// process entry of given name in current directory; recursively descend if
// entry names a directory and the recursive flag is set
func processInputs(entry, accumulatedPath string) {
	var err error
	var info os.FileInfo

	if info, err = os.Stat(entry); err != nil {
		myError("error: %s", err)
		return
	}

	if info.IsDir() {
		if !*recursive {
			myError("%s is a directory without recursive flag", accumulatedPath)
			return
		}

		var thisDir string
		if thisDir, err = os.Getwd(); err != nil {
			panic(err)
		}
		if err = os.Chdir(entry); err != nil {
			panic(err)
		}
		defer func() {
			if err := os.Chdir(thisDir); err != nil {
				panic(err)
			}
		}()

		var f *os.File
		if f, err = os.Open("."); err != nil {
			myError("error: could not open directory %s; %s", accumulatedPath, err)
			return
		}
		var entries_info []os.FileInfo
		if entries_info, err = f.Readdir(-1); err != nil {
			myError("error: could not read directory %s; %s", accumulatedPath, err)
			return
		}
		if err = f.Close(); err != nil {
			panic(err)
		}

		for _, entry := range entries_info {
			newAccumulatedPath := fmt.Sprintf("%s%c%s", accumulatedPath, os.PathSeparator, entry.Name())
			processInputs(entry.Name(), newAccumulatedPath)
		}
	} else {
		var f *os.File
		var e error
		if f, e = os.Open(entry); e != nil {
		}

		defer func() {
			f.Close()
		}()

		processReader(accumulatedPath, f, info.Size())
	}
}

// count the results coming in through a channel and report the final amount
func findCount(path string, results <-chan substr.Result) int {
	count := 0
	for r := range results {
		if r.Error != nil {
			myError("%s: error -- %s", path, r.Error)
		} else {
			count++
		}
	}
	return count
}

// display an error using fmt.Printf style args to stderr
func myError(formatString string, elements ...interface{}) {
	fmt.Fprintf(os.Stderr, formatString, elements...)
	fmt.Fprintln(os.Stderr)
}

// display an error using fmt.Printf style args to stderr; exit with specified status code
func myFatal(code int, formatString string, elements ...interface{}) {
	myError(formatString, elements...)
	os.Exit(code)
}

// calculate how many digits are needed for numbers up to max
func calcWidth(max int64) int {
	width := 1
	max /= 10
	for max != 0 {
		width++
		max /= 10
	}
	return width
}

func main() {
	flag.Var(&needleBytes, "b", "bytes to look for within input(s); e.g., \"-b 00ff00AA\"")
	flag.Parse() // scan the arguments list

	if len(*needleString) != 0 {
		if len(needleBytes) == 0 {
			needle = substr.NewNeedleStr(*needleString)
		} else {
			myFatal(status_fatal_error, "error: specified both -t and -b parameters")
		}
	} else if len(needleBytes) == 0 {
		myFatal(status_fatal_error, "error: specified neither -t nor -b parameter")
	} else {
		needle = substr.NewNeedleBytes(needleBytes)
	}

	if *quiet {
		*findAll = false
	}

	inputs := flag.Args()

	if len(inputs) == 0 && !*processStdin {
		myFatal(status_fatal_error, "error: did not specify any input files or directories or provide the standard input flag")
	}

	if *processStdin {
		processReader("STDIN", os.Stdin, 0)
	}

	for _, fname := range inputs {
		processInputs(fname, fname)
	}

	if *quiet {
		os.Exit(status_none_found)
	}
}
