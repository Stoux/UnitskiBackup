package main

import (
	"fmt"
	"os"
	"unitski-backup/unitski/commands"
)

func main() {
	cmd := os.Args[0]
	args := os.Args[1:]
	argsLength := len(args)

	// TODO: Add arguments for testing the config file
	// TODO: Add interactive CLI to add & test a backup configuration

	if argsLength != 2 {
		usage(cmd)
	} else if args[0] == "backup" {
		commands.Sync(args[1])
	} else if args[0] == "test" {
		panic("Not implemented yet")
	} else {
		usage(cmd)
	}
}

func usage(cmd string) {
	fmt.Println("Usage: " + cmd + " (backup|test) config-file")
	os.Exit(1)
}
