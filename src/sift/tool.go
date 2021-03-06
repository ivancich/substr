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
	ba "bytearray"
	"flag"
	"fmt"
	"io"
	"myerr"
	"os"
	"substr"
)

const (
	status_found       = 0
	status_none_found  = 1
	status_fatal_error = 2
)

var statFunction func (string) (os.FileInfo, error)

var needleString *string = flag.String("t", "", "text to look for within input(s)")
var findAll *bool = flag.Bool("a", false, "display all matching offsets")
var recursive *bool = flag.Bool("r", false, "recursively descend directories")
var displayCount *bool = flag.Bool("c", false, "display count of matches")
var quiet *bool = flag.Bool("q", false, "quiet; exit immediatly with status 0 if any matches found")
var processStdin *bool = flag.Bool("stdin", false, "process stdin as one of the inputs")
var swapOutput *bool = flag.Bool("swap", false, "output in format for swap tool")
var followSymbolicLinks *bool = flag.Bool("L", false, "follow symbolic links")

var needleBytes ba.ByteArray
var needle *substr.Needle

func processReader(path string, in io.Reader, in_size int64) {
	if *displayCount {
		count := findCount(path, substr.IndexesWithinReaderNeedle(in, needle))
		fmt.Printf("%s: %d\n", path, count)
	} else if *swapOutput {
		found := false
		gotError := false
		for result := range substr.IndexesWithinReaderNeedle(in, needle) {
			if gotError {
				if result.Error != nil {
					myerr.MyError("    error: %s", result.Error)
				}
				continue
			} else if !found {
				if result.Error != nil {
					myerr.MyError("    error: %s", result.Error)
					gotError = true
				} else {
					fmt.Printf("\"%s\" %d", path, result.Offset)
					found = true
				}
			} else {
				if result.Error != nil {
					fmt.Println()
					myerr.MyError("    error: %s", result.Error)
					gotError = true
				} else {
					fmt.Printf(" %d", result.Offset)
				}
			}
		}
		if found && !gotError {
			fmt.Println()
		}
	} else if *findAll {
		count := 0
		for result := range substr.IndexesWithinReaderNeedle(in, needle) {
			if count == 0 {
				fmt.Printf("%s:\n", path)
			}
			count++
			if result.Error != nil {
				myerr.MyError("    error: %s", result.Error)
			} else {
				fmt.Printf("    match %3d at offset %*d\n", count, calcWidth(in_size), result.Offset)
			}
		}
	} else {
		found, offset, err := substr.IndexWithinReaderNeedle(in, needle)
		if err != nil {
			myerr.MyError("%s: error -- %s", path, err)
		} else if found {
			if *quiet {
				os.Exit(status_found)
			} else {
				fmt.Printf("%s: first offset %d\n", path, offset)
			}
		}
	}
}

// process entry of given name in current directory; recursively descend if
// entry names a directory and the recursive flag is set
func processInputs(entry, accumulatedPath string) {
	var err error
	var info os.FileInfo

	if info, err = statFunction(entry); err != nil {
		myerr.MyError("error: %s", err)
		return
	}
	
	// skip over non-regular files and non-directories
	if 0 != info.Mode() & (os.ModeSymlink | os.ModeNamedPipe | os.ModeSocket | os.ModeDevice) {
		return
	}

	if info.IsDir() {
		if !*recursive {
			myerr.MyError("%s is a directory without recursive flag", accumulatedPath)
			return
		}

		var thisDir string
		if thisDir, err = os.Getwd(); err != nil {
			myerr.MyPanic(err)
		}
		myerr.MyPanic(os.Chdir(entry))
		defer func() {
			myerr.MyPanic(os.Chdir(thisDir))
		}()

		var f *os.File
		if f, err = os.Open("."); err != nil {
			myerr.MyError("error: could not open directory %s; %s", accumulatedPath, err)
			return
		}
		var entries_info []os.FileInfo
		if entries_info, err = f.Readdir(-1); err != nil {
			myerr.MyPanic(f.Close())
			myerr.MyError("error: could not read directory %s; %s", accumulatedPath, err)
			return
		}
		myerr.MyPanic(f.Close())

		for _, entry := range entries_info {
			newAccumulatedPath := fmt.Sprintf("%s%c%s", accumulatedPath, os.PathSeparator, entry.Name())
			processInputs(entry.Name(), newAccumulatedPath)
		}
	} else {
		var f *os.File
		var e error
		if f, e = os.Open(entry); e != nil {
			myerr.MyError("warning: could not open %s; skipping", accumulatedPath)
			return
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
			myerr.MyError("%s: error -- %s", path, r.Error)
		} else {
			count++
		}
	}
	return count
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
			myerr.MyFatal(status_fatal_error, "error: specified both -t and -b parameters")
		}
	} else if len(needleBytes) == 0 {
		myerr.MyFatal(status_fatal_error, "error: specified neither -t nor -b parameter")
	} else {
		needle = substr.NewNeedleBytes(needleBytes)
	}
	
	if *followSymbolicLinks {
		statFunction = os.Stat
	} else {
		statFunction = os.Lstat
	}

	if *quiet {
		*findAll = false
	}

	if *swapOutput {
		*findAll = true
	}

	inputs := flag.Args()

	if len(inputs) == 0 && !*processStdin {
		myerr.MyFatal(status_fatal_error, "error: did not specify any input files or directories or provide the standard input flag")
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
