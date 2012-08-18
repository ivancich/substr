package main

import (
	ba "bytearray"
	// "bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"myerr"
	"os"
	"sort"
	"strconv"
)

const status_fatal_error = 1

type uint64Slice []uint64

func NewUint64Slice() uint64Slice {
	return make([]uint64, 0)
}

func (d uint64Slice) Len() int {
	return len(d)
}

func (d uint64Slice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d uint64Slice) Less(i, j int) bool {
	return d[i] < d[j]
}

var fromString *string = flag.String("from", "", "text to replace; used as insurance")
var toString *string = flag.String("to", "", "replacement text")
var quiet *bool = flag.Bool("q", false, "quiet")
var processStdin *bool = flag.Bool("stdin", false, "process stdin as one of the inputs")

var fromBytes, toBytes, buffer ba.ByteArray

func main() {
	defer myerr.MyDefer()

	var err error

	flag.Var(&fromBytes, "fromb", "bytes to replace; used to make sure you don't overwrite wrong data; e.g., \"-b 00ff00AA\"")
	flag.Var(&toBytes, "tob", "replacement bytes; e.g., \"-b 0FE32d17\"")
	flag.Parse() // scan the arguments list

	if len(*fromString) != 0 {
		if len(fromBytes) == 0 {
			fromBytes = []byte(*fromString)
		} else {
			myerr.MyFatal(status_fatal_error, "error: specified both -from and -fromb parameters")
			return
		}
	}

	if len(*toString) != 0 {
		if len(toBytes) == 0 {
			toBytes = []byte(*toString)
		} else {
			myerr.MyFatal(status_fatal_error, "error: specified both -to and -tob parameters")
			return
		}
	} else if len(toBytes) == 0 {
		myerr.MyFatal(status_fatal_error, "error: must specify either -to or -tob parameter")
		return
	}

	if len(fromBytes) != 0 && len(fromBytes) != len(toBytes) {
		myerr.MyFatal(status_fatal_error, "error: if you specify -from or -fromb it must be the same size as -to or -tob; %d is not equal to %d", len(fromBytes), len(toBytes))
		return
	}

	var inFileName string
	positions := NewUint64Slice()
	gotError := false

	for i, arg := range flag.Args() {
		if i == 0 {
			inFileName = arg
		} else {
			var v uint64
			v, err = strconv.ParseUint(arg, 10, 64)
			if err != nil {
				myerr.MyError("error: trying to parse \"%s\" as an offset; got %s", arg, err)
				gotError = true
			} else {
				positions = append(positions, v)
			}
		}
	}

	sort.Sort(positions)

	if gotError {
		myerr.MyFatal(status_fatal_error, "must exit due to errors")
		return
	}

	inFile, oe := os.Open(inFileName)
	if oe != nil {
		myerr.MyFatal(status_fatal_error, "could not open file \"%s\"; %s", inFileName, oe)
	}
	defer func() {
		inFile.Close()
	}()

	outFileName, outFile, oe2 := makeTempFile(inFileName, "tmp")
	if oe2 != nil {
		myerr.MyFatal(status_fatal_error, "%s", oe2)
	}
	complete := false
	defer func() {
		outFile.Close()
		if complete {
			var mode os.FileMode
			if fi, e := inFile.Stat(); e == nil {
				mode = fi.Mode()
			} else {
				myerr.MyPanic(e)
			}
			
			var backupName string
			var backupFile *os.File
			backupName, backupFile, err = makeTempFile(inFileName, "backup")
			myerr.MyPanic(err)
			err = backupFile.Close()
			myerr.MyPanic(err)
			err = os.Rename(inFileName, backupName)
			myerr.MyPanic(err)
			err = os.Rename(outFileName, inFileName)
			myerr.MyPanic(err)
			err = os.Chmod(inFileName, mode)
			myerr.MyPanic(err)
		} else {
			err = os.Remove(outFileName)
		}
	}()

	if _, err = io.Copy(outFile, inFile); err != nil {
		myerr.MyFatal(status_fatal_error, "error: %s", err)
		return
	}

	buffer := make([]byte, len(fromBytes), len(fromBytes))
	for _, offset := range positions {
		skip := false
		if len(fromBytes) != 0 {
			_, err = outFile.ReadAt(buffer, int64(offset))
			myerr.MyPanic(err)
			if !sameBytes(fromBytes, buffer) {
				fmt.Printf("warning: not same at offset %d; skipping\n", offset) 
				skip = true
			}
		}

		if !skip {
			_, err = outFile.WriteAt(toBytes, int64(offset))
			myerr.MyPanic(err)
		}
	}

	complete = true
}

func makeTempFile(template, suffix string) (fname string, file *os.File, err error) {
	template2 := template + "." + suffix
	for i := 0; i <= 100; i++ {
		fname = template2 + strconv.Itoa(i)
		file, err = os.OpenFile(fname, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err == nil {
			break
		}
	}

	if err != nil {
		err = errors.New(fmt.Sprintf("could not create temp file based on \"%s\"", template))
	}

	return
}

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	
	for i, av := range a {
		if av != b[i] {
			return false
		}
	}
	
	return true
}
