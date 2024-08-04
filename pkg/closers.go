package loginjector

import (
	"io"
	"log"
)

// CloseOrPanic closes the closer and panics if there is an error.
func CloseOrPanic(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		panic(err)
	}
}

// CloseOrLog closes the closer and logs the error if there is one.
func CloseOrLog(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

// CloseOrPrintLn closes the closer and prints the error if there is one.
func CloseOrPrintLn(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		println(err)
	}
}

// CloseOrIgnore closes the closer and ignores the error if there is one.
func CloseOrIgnore(closer io.Closer) {
	_ = closer.Close()
}
