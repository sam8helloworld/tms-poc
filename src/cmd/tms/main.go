package main

import (
	"os"

	"github.com/sam8helloworld/tms-poc/cmd/tms/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
