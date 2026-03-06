package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func runSOWDelete(args []string) error {
	fs := flag.NewFlagSet("sow_delete", flag.ContinueOnError)
	var transport transportOptions
	addTransportFlags(fs, &transport, true)
	topic := fs.String("topic", "", "SOW topic")
	filter := fs.String("filter", "", "filter expression for records to delete")
	filePath := fs.String("file", "", "file containing records to delete")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *topic == "" {
		return fmt.Errorf("topic is required (-topic flag)")
	}

	client, _, err := connect(transport, *timeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	var started = time.Now()
	var deleted uint64
	if *filePath != "" {
		payload, readErr := os.ReadFile(*filePath)
		if readErr != nil {
			return fmt.Errorf("read file: %w", readErr)
		}
		for _, record := range splitRecords(payload, '\n') {
			stats, deleteErr := client.SowDeleteByData(*topic, record)
			if deleteErr != nil {
				return fmt.Errorf("sow_delete: %w", deleteErr)
			}
			if matches, ok := stats.Matches(); ok {
				deleted += uint64(matches)
			}
		}
	} else {
		var deleteFilter = *filter
		if strings.TrimSpace(deleteFilter) == "" {
			deleteFilter = "1=1"
		}
		stats, deleteErr := client.SowDelete(*topic, deleteFilter)
		if deleteErr != nil {
			return fmt.Errorf("sow_delete: %w", deleteErr)
		}
		if matches, ok := stats.Matches(); ok {
			deleted = uint64(matches)
		}
	}

	fmt.Fprintf(writer, "deleted %d records in %v\n", deleted, time.Since(started).Round(time.Millisecond))
	flushOutput()
	return nil
}
