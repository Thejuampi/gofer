package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"

	"github.com/Thejuampi/amps-client-go/amps"
)

func runSubscribe(args []string) error {
	fs := flag.NewFlagSet("subscribe", flag.ContinueOnError)
	var transport transportOptions
	addTransportFlags(fs, &transport, true)
	topic := fs.String("topic", "", "topic to subscribe to")
	filter := fs.String("filter", "", "content filter expression")
	copyServer := fs.String("copy", "", "secondary server for mirrored output")
	format := fs.String("format", "", "spark-style output format")
	delta := fs.Bool("delta", false, "use delta subscription")
	ack := fs.Bool("ack", false, "enable auto-ack for queue messages")
	backlog := fs.Bool("backlog", false, "request backlog when reading from queues")
	maxBacklog := fs.String("max_backlog", "", "queue backlog depth")
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
		flushOutput()
		_ = client.Close()
	}()

	copyClient, err := newCopyPublisher(*copyServer, transport, *timeout)
	if err != nil {
		return err
	}
	defer copyClient.Close()

	var commandName = "subscribe"
	if *delta {
		commandName = "delta_subscribe"
	}

	cmd := amps.NewCommand(commandName).SetTopic(*topic)
	if *filter != "" {
		cmd.SetFilter(*filter)
	}
	var options []string
	if *backlog && strings.TrimSpace(*maxBacklog) == "" {
		options = append(options, "max_backlog=2")
	}
	if strings.TrimSpace(*maxBacklog) != "" {
		options = append(options, "max_backlog="+strings.TrimSpace(*maxBacklog))
	}
	if len(options) > 0 {
		cmd.SetOptions(strings.Join(options, ","))
	}
	if *ack {
		client.SetAutoAck(true)
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
		if cmdType != amps.CommandPublish && cmdType != amps.CommandOOF {
			return nil
		}
		writeMessage(msg, *format, prettyVal)
		if err := copyClient.Publish(*topic, msg.Data(), *delta); err != nil {
			return err
		}
		if limit > 0 && atomic.AddInt32(&count, 1) >= limit {
			select {
			case done <- struct{}{}:
			default:
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	if limit > 0 {
		select {
		case <-done:
		case <-sigCh:
		}
	} else {
		<-sigCh
	}

	return nil
}
