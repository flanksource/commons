package main

import (
	"os"

	"github.com/flanksource/commons/deps"
	"github.com/flanksource/commons/logger"
)

func main() {

	logger.StandardLogger().SetLogLevel(2)
	for _, arg := range os.Args[1:] {
		if err := deps.InstallDependency(arg, "", "bin"); err != nil {
			logger.Fatalf("Failed to download %s: %v", arg, err)
		}
	}
}
