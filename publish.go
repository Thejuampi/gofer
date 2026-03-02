package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func runPublish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	server := fs.String("server", "", "AMPS URI")
	topic := fs.String("topic", "", "destination topic")
	data := fs.String("data", "", "message payload (if empty, reads from stdin)")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *topic == "" {
		return fmt.Errorf("topic is required (-topic flag)")
	}

	client, err := connect(*server, *timeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	// Single-message mode via -data flag.
	if *data != "" {
		return client.Publish(*topic, *data)
	}

	// Streaming mode: read lines from stdin.
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := client.Publish(*topic, line); err != nil {
			return fmt.Errorf("publish message #%d: %w", count+1, err)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	fmt.Fprintf(writer, "published %d message(s)\n", count)
	flushOutput()
	return nil
}
