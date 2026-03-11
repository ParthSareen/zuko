package main

import (
	"os"
	"path/filepath"

	"github.com/ParthSareen/zuko/cmd"
	"github.com/ParthSareen/zuko/proxy"
)

func main() {
	invoked := filepath.Base(os.Args[0])
	if invoked != "zuko" {
		proxy.Run(invoked, os.Args[1:])
		return
	}
	cmd.Execute()
}
