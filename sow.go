package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
)

func runSOW(args []string) error {
	fs := flag.NewFlagSet("sow", flag.ContinueOnError)
	var transport transportOptions
	addTransportFlags(fs, &transport, true)
	topic := fs.String("topic", "", "SOW topic to query")
	filter := fs.String("filter", "", "content filter expression")
	copyServer := fs.String("copy", "", "secondary server for mirrored output")
	format := fs.String("format", "", "output format template")
	batchSize := fs.String("batchsize", "", "query batch size")
	orderBy := fs.String("orderby", "", "query ordering")
	topN := fs.String("topn", "", "max records to return")
	timeout := fs.Duration("timeout", defaultTimeout, "connection timeout")
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
		flushOutput()
		_ = client.Close()
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

	cmd := amps.NewCommand("sow").SetTopic(*topic).AddAckType(amps.AckTypeCompleted)
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

	prettyVal := *pretty
	done := make(chan struct{}, 1)
	var once sync.Once
	var started = time.Now()
	var count int32

	_, err = client.ExecuteAsync(cmd, func(msg *amps.Message) error {
		cmdType, _ := msg.Command()
		switch cmdType {
		case amps.CommandSOW:
			writeMessage(msg, *format, prettyVal)
			if err := copyClient.Publish(*topic, msg.Data(), false); err != nil {
				return err
			}
			atomic.AddInt32(&count, 1)
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

	writeSummary("Total messages received:", int(count), started)
	return nil
}
