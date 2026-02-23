package main

import (
	"errors"
	"fmt"
	"os"
)

var version = "dev"

func init() {
	rootCmd.Version = version
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		var statusErr *httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
