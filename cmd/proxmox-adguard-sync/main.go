package main

import (
	"fmt"
	"os"
)

const applicationName = "proxmox-adguard-sync"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", applicationName, err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Printf("%s Go rewrite initialized successfully\n", applicationName)

	return nil
}
