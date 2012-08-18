package myerr

import (
	"fmt"
	"os"
)

var exitCode int = 0

// display an error using fmt.Printf style args to stderr
func MyError(formatString string, elements ...interface{}) {
	fmt.Fprintf(os.Stderr, formatString, elements...)
	fmt.Fprintln(os.Stderr)
}

func MyFatal(code int, formatString string, elements ...interface{}) {
	exitCode = code
	MyError(formatString, elements...)
}

// display an error using fmt.Printf style args to stderr; exit with specified status code
func MyImmediateFatal(code int, formatString string, elements ...interface{}) {
	MyError(formatString, elements...)
	os.Exit(code)
}

func MyPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func MyDefer() {
	s := recover()
	if s != nil {
		MyError("panic: %s", s)
	}
	os.Exit(exitCode)
}
