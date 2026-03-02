package main

import (
	"flag"
	"fmt"
)

func runSOWDelete(args []string) error {
	fs := flag.NewFlagSet("sow_delete", flag.ContinueOnError)
	server := fs.String("server", "", "AMPS URI")
	topic := fs.String("topic", "", "SOW topic")
	filter := fs.String("filter", "", "filter expression for records to delete")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *topic == "" {
		return fmt.Errorf("topic is required (-topic flag)")
	}
	if *filter == "" {
		return fmt.Errorf("filter is required (-filter flag)")
	}

	client, err := connect(*server, *timeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	stats, err := client.SowDelete(*topic, *filter)
	if err != nil {
		return fmt.Errorf("sow_delete: %w", err)
	}

	if stats != nil {
		if matches, ok := stats.Matches(); ok {
			fmt.Fprintf(writer, "deleted %d record(s)\n", matches)
			flushOutput()
			return nil
		}
	}

	fmt.Fprintln(writer, "deleted (count unavailable)")
	flushOutput()
	return nil
}
