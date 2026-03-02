package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
)

func runSOW(args []string) error {
	fs := flag.NewFlagSet("sow", flag.ContinueOnError)
	server := fs.String("server", "", "AMPS URI")
	topic := fs.String("topic", "", "SOW topic to query")
	filter := fs.String("filter", "", "content filter expression")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	pretty := fs.Bool("pretty", false, "pretty-print JSON output")
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
	defer func() {
		flushOutput()
		_ = client.Close()
	}()

	cmd := amps.NewCommand("sow").SetTopic(*topic).AddAckType(amps.AckTypeCompleted)
	if *filter != "" {
		cmd.SetFilter(*filter)
	}

	prettyVal := *pretty
	done := make(chan struct{}, 1)
	var once sync.Once

	_, err = client.ExecuteAsync(cmd, func(msg *amps.Message) error {
		cmdType, _ := msg.Command()
		switch cmdType {
		case amps.CommandSOW:
			writeMessage(msg, prettyVal)
		case amps.CommandGroupEnd:
			once.Do(func() { close(done) })
		case amps.CommandAck:
			// completed ack signals SOW is done (fallback).
			if ackType, ok := msg.AckType(); ok && ackType == amps.AckTypeCompleted {
				once.Do(func() { close(done) })
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sow: %w", err)
	}

	select {
	case <-done:
	case <-time.After(*timeout):
		return fmt.Errorf("sow query timed out")
	}

	return nil
}
