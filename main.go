package main

import (
	"log"

	"github.com/qshuai/fswatcher/cmd"
)

func main() {
	command, err := cmd.New()
	if err != nil {
		log.Fatalf("Program initialize failed: %s", err)
	}

	command.Execute()
}
