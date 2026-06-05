package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sbavert/vidgen/configuration"
)

func main() {
	argument_list := os.Args

	if len(argument_list) > 2 {
		fmt.Println("Unknown arguments specified")
		os.Exit(1)
	}

	config_json_file := os.Args[1]
	abs_config_path, err := filepath.Abs(config_json_file)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var config = configuration.LoadJsonFile(&abs_config_path)

	fmt.Println(config)

}
