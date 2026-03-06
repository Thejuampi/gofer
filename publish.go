package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

func runPublish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	var transport transportOptions
	addTransportFlags(fs, &transport, true)
	topic := fs.String("topic", "", "destination topic")
	data := fs.String("data", "", "message payload (if empty, reads from stdin)")
	filePath := fs.String("file", "", "file to publish records from")
	delimiter := fs.String("delimiter", "10", "decimal message delimiter")
	delta := fs.Bool("delta", false, "use delta publish")
	rate := fs.Float64("rate", 0, "messages per second")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *topic == "" {
		return fmt.Errorf("topic is required (-topic flag)")
	}

	delimiterValue, err := strconv.Atoi(*delimiter)
	if err != nil || delimiterValue < 0 || delimiterValue > 255 {
		return fmt.Errorf("delimiter must be a decimal byte value")
	}

	client, _, err := connect(transport, *timeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	var payload []byte
	if *data != "" {
		payload = []byte(*data)
	} else if *filePath != "" {
		payload, err = os.ReadFile(*filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	} else {
		payload, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	}

	var started = time.Now()
	var pacer = newPublishPacer(*rate, time.Now, time.Sleep)
	var records = splitRecords(payload, byte(delimiterValue))
	for index, record := range records {
		pacer.Wait()
		if *delta {
			err = client.DeltaPublishBytes(*topic, record)
		} else {
			err = client.Publish(*topic, string(record))
		}
		if err != nil {
			return fmt.Errorf("publish message #%d: %w", index+1, err)
		}
	}
	if len(records) > 0 {
		if err := client.Flush(); err != nil {
			return fmt.Errorf("flush publish pipeline: %w", err)
		}
	}

	writeSummary("total messages published:", len(records), started)
	flushOutput()
	return nil
}
