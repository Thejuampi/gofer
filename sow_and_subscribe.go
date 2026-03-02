package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
)

func runSOWAndSubscribe(args []string) error {
	fs := flag.NewFlagSet("sow_and_subscribe", flag.ContinueOnError)
	server := fs.String("server", "", "AMPS URI")
	topic := fs.String("topic", "", "SOW topic")
	filter := fs.String("filter", "", "content filter expression")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	maxN := fs.Int("n", 0, "max messages to receive (0 = unlimited)")
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

	cmd := amps.NewCommand("sow_and_subscribe").SetTopic(*topic)
	if *filter != "" {
		cmd.SetFilter(*filter)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	var count int32
	limit := int32(*maxN)
	prettyVal := *pretty
	done := make(chan struct{}, 1)

	_, err = client.ExecuteAsync(cmd, func(msg *amps.Message) error {
		cmdType, _ := msg.Command()
		switch cmdType {
		case amps.CommandSOW, amps.CommandPublish:
			writeMessage(msg, prettyVal)
			if limit > 0 && atomic.AddInt32(&count, 1) >= limit {
				select {
				case done <- struct{}{}:
				default:
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sow_and_subscribe: %w", err)
	}

	if limit > 0 {
		select {
		case <-done:
		case <-sigCh:
		case <-time.After(*timeout):
			return fmt.Errorf("sow_and_subscribe timed out")
		}
	} else {
		<-sigCh
	}

	return nil
}
