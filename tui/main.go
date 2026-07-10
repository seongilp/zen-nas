package main

import (
	"fmt"
	"os"
)

// version은 빌드 시 -ldflags "-X main.version=..." 로 주입된다.
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("zenfs", version)
		return
	}
	if err := runTUI(); err != nil {
		fmt.Fprintln(os.Stderr, "zenfs:", err)
		os.Exit(1)
	}
}
