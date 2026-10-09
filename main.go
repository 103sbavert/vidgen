package main

import (
	"fmt"
	"os"
	"path/filepath"

	"sbavert/vidgen/config"
)

func main() {
	if len(os.Args) > 2 {
		fmt.Println("Unknown arguments specified")
		os.Exit(1)
	}

	confPath := os.Args[1]
	absConfPath, err := filepath.Abs(confPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config := config.LoadJSONFile(&absConfPath)

	fmt.Println(config)
}
