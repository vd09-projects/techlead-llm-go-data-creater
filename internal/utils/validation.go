package utils

import (
	"fmt"
	"os"
)

func MustNotErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
