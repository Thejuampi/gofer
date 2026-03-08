package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
)

func runSOWAndSubscribe(args []string) (err error) {
	fs := flag.NewFlagSet("sow_and_subscribe", flag.ContinueOnError)
	var transport transportOptions
	addTransportFlags(fs, &transport, true)
	topic := fs.String("topic", "", "SOW topic")
	filter := fs.String("filter", "", "content filter expression")
	copyServer := fs.String("copy", "", "secondary server for mirrored output")
	format := fs.String("format", "", "output format template")
	delta := fs.Bool("delta", false, "request delta subscription")
	batchSize := fs.String("batchsize", "", "query batch size")
	orderBy := fs.String("orderby", "", "query ordering")
	topN := fs.String("topn", "", "max records to return")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
	maxN := fs.Int("n", 0, "max messages to receive (0 = unlimited)")
	pretty := fs.Bool("pretty", false, "pretty-print JSON output")
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
	defer func() {
		if flushErr := flushOutput(); err == nil && flushErr != nil {
			err = flushErr
		}
		if closeErr := client.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	copyClient, err := newCopyPublisher(*copyServer, transport, *timeout)
	if err != nil {
		return err
	}
	defer copyClient.Close()

	batchSizeValue, err := parseUintOrDefault(*batchSize, 0)
	if err != nil {
		return fmt.Errorf("parse batchsize: %w", err)
	}
	topNValue, err := parseUintOrDefault(*topN, 0)
	if err != nil {
		return fmt.Errorf("parse topn: %w", err)
	}

	var commandName = "sow_and_subscribe"
	if *delta {
		commandName = "sow_and_delta_subscribe"
	}
	cmd := amps.NewCommand(commandName).SetTopic(*topic)
	if *filter != "" {
		cmd.SetFilter(*filter)
	}
	if batchSizeValue > 0 {
		cmd.SetBatchSize(batchSizeValue)
	}
	if strings.TrimSpace(*orderBy) != "" {
		cmd.SetOrderBy(*orderBy)
	}
	if topNValue > 0 {
		cmd.SetTopN(topNValue)
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
			if err := writeMessage(msg, *format, prettyVal); err != nil {
				return err
			}
			if err := copyClient.Publish(*topic, msg.Data(), *delta); err != nil {
				return err
			}
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
