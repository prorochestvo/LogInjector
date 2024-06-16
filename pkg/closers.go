package loginjector

import (
	"io"
	"log"
)

func CloseOrPanic(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		panic(err)
	}
}

func CloseOrLog(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

func CloseOrIgnore(closer io.Closer) {
	_ = closer.Close()
}
