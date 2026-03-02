package main

import (
	"flag"
	"fmt"
	"time"
)

func runPing(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	server := fs.String("server", "", "AMPS URI (e.g. tcp://localhost:9007/amps/json)")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	start := time.Now()
	client, err := connect(*server, *timeout)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)
	defer func() { _ = client.Close() }()

	version := client.ServerVersion()
	fmt.Fprintf(writer, "OK %s (%v)\n", version, elapsed.Round(time.Microsecond))
	flushOutput()
	return nil
}
