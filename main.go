// Package main implements gofer — a cross-platform CLI for interacting with
// AMPS instances. It compiles to a single native binary with zero external
// dependencies beyond the amps-client-go package.
package main

import (
	"fmt"
	"os"
)

// version is stamped at release time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

const usage = `gofer — AMPS command-line client (Go edition)

Usage: gofer <command> [flags]

Commands:
  ping              Test connectivity to an AMPS instance
  publish           Publish a message to a topic
  subscribe         Subscribe to a topic and stream messages
  sow               Query the State-of-the-World for a topic
  sow_and_subscribe SOW snapshot followed by live subscription
  sow_delete        Delete records from a SOW topic

Run 'gofer <command> -help' for details on a specific command.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	command := os.Args[1]
	// Strip the subcommand so each handler sees only its own flags.
	args := os.Args[2:]

	var err error
	switch command {
	case "ping":
		err = runPing(args)
	case "publish":
		err = runPublish(args)
	case "subscribe":
		err = runSubscribe(args)
	case "sow":
		err = runSOW(args)
	case "sow_and_subscribe":
		err = runSOWAndSubscribe(args)
	case "sow_delete":
		err = runSOWDelete(args)
	case "help", "-help", "--help", "-h":
		fmt.Fprint(os.Stdout, usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "gofer: unknown command %q\n\n%s", command, usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "gofer %s: %v\n", command, err)
		os.Exit(1)
	}
}
