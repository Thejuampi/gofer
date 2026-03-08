package main

import (
	"flag"
	"fmt"
)

func runPing(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	var transport transportOptions
	addTransportFlags(fs, &transport, false)
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, uri, err := connect(transport, *timeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if _, err := fmt.Fprintf(writer, "Successfully connected to %s\n", uri); err != nil {
		return err
	}
	return flushOutput()
}
