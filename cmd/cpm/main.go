package main

import (
	"github.com/CognitiveOS-Project/cpm/cmd"
	"github.com/CognitiveOS-Project/cpm/internal/log"
)

func main() {
	log.Init("")
	cmd.Execute()
}
