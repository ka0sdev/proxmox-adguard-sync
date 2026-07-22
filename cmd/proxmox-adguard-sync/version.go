package main

import (
	"fmt"
	"io"
)

var (
	version = "development"
	commit  = "unknown"
	buildAt = "unknown"
)

func printVersion(writer io.Writer) {
	_, _ = fmt.Fprintf(
		writer,
		"%s %s\ncommit: %s\nbuilt: %s\n",
		applicationName,
		version,
		commit,
		buildAt,
	)
}
