package loginjector

import (
	"io"
	"log"
)

// CloseOrPanic closes the closer and panics if there is an error.
func CloseOrPanic(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		panic(err.Error() + "\n" + StackTrace())
	}
}

// CloseOrLog closes the closer and logs the error if there is one.
func CloseOrLog(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

// CloseOrIgnore closes the closer and ignores the error if there is one.
func CloseOrIgnore(closer io.Closer) {
	_ = closer.Close()
}
