package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
)

// writer is a buffered stdout writer for high-throughput output.
var writer = bufio.NewWriterSize(os.Stdout, 64*1024)

// flushOutput flushes the buffered writer to stdout.
func flushOutput() { _ = writer.Flush() }

func writeLine(line string) {
	_, _ = writer.WriteString(line)
	_ = writer.WriteByte('\n')
}

func writeSummary(prefix string, count int, started time.Time) {
	var elapsed = time.Since(started)
	var rate = 0.0
	if elapsed > 0 {
		rate = float64(count) / elapsed.Seconds()
	}
	fmt.Fprintf(writer, "%s %d (%.2f/s)\n", prefix, count, rate)
}

func renderMessage(msg *amps.Message, format string, pretty bool) []byte {
	if msg == nil {
		return nil
	}
	if strings.TrimSpace(format) != "" {
		return []byte(replaceFormatTokens(format, msg))
	}

	data := msg.Data()
	if len(data) == 0 {
		return nil
	}
	if pretty {
		var raw json.RawMessage
		if json.Unmarshal(data, &raw) == nil {
			indented, err := json.MarshalIndent(raw, "", "  ")
			if err == nil {
				data = indented
			}
		}
	}
	return data
}

func writeMessage(msg *amps.Message, format string, pretty bool) {
	data := renderMessage(msg, format, pretty)
	if len(data) == 0 {
		return
	}
	_, _ = writer.Write(data)
	_ = writer.WriteByte('\n')
}

func replaceFormatTokens(format string, msg *amps.Message) string {
	var replacer = strings.NewReplacer(
		"{bookmark}", fieldString(msg.Bookmark()),
		"{command}", commandName(msg),
		"{correlation_id}", fieldString(msg.CorrelationID()),
		"{data}", string(msg.Data()),
		"{expiration}", uintString(msg.Expiration()),
		"{lease_period}", fieldString(msg.LeasePeriod()),
		"{length}", fmt.Sprintf("%d", len(msg.Data())),
		"{sowkey}", fieldString(msg.SowKey()),
		"{timestamp}", fieldString(msg.Timestamp()),
		"{topic}", fieldString(msg.Topic()),
		"{user_id}", fieldString(msg.UserID()),
	)
	return replacer.Replace(format)
}

func fieldString(value string, ok bool) string {
	if !ok {
		return ""
	}
	return value
}

func uintString(value uint, ok bool) string {
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", value)
}

func commandName(msg *amps.Message) string {
	command, ok := msg.Command()
	if !ok {
		return ""
	}
	switch command {
	case amps.CommandAck:
		return "ack"
	case amps.CommandDeltaPublish:
		return "delta_publish"
	case amps.CommandDeltaSubscribe:
		return "delta_subscribe"
	case amps.CommandFlush:
		return "flush"
	case amps.CommandGroupBegin:
		return "group_begin"
	case amps.CommandGroupEnd:
		return "group_end"
	case amps.CommandOOF:
		return "oof"
	case amps.CommandPublish:
		return "publish"
	case amps.CommandSOW:
		return "sow"
	case amps.CommandSOWAndDeltaSubscribe:
		return "sow_and_delta_subscribe"
	case amps.CommandSOWAndSubscribe:
		return "sow_and_subscribe"
	case amps.CommandSOWDelete:
		return "sow_delete"
	case amps.CommandSubscribe:
		return "subscribe"
	case amps.CommandUnsubscribe:
		return "unsubscribe"
	default:
		return ""
	}
}

func splitRecords(payload []byte, delimiter byte) [][]byte {
	var parts = bytes.Split(payload, []byte{delimiter})
	var records = make([][]byte, 0, len(parts))
	for _, part := range parts {
		part = bytes.TrimSpace(part)
		if len(part) == 0 {
			continue
		}
		records = append(records, bytes.Clone(part))
	}
	return records
}

type publishPacer struct {
	rate  float64
	now   func() time.Time
	sleep func(time.Duration)
	start time.Time
	sent  int
	armed bool
}

func newPublishPacer(rate float64, now func() time.Time, sleep func(time.Duration)) *publishPacer {
	if now == nil {
		now = time.Now
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	return &publishPacer{
		rate:  rate,
		now:   now,
		sleep: sleep,
	}
}

func (pacer *publishPacer) Wait() {
	if pacer == nil || pacer.rate <= 0 {
		return
	}
	if !pacer.armed {
		pacer.start = pacer.now()
		pacer.armed = true
		pacer.sent = 1
		return
	}
	pacer.sent++
	target := pacer.start.Add(time.Duration(float64(time.Second) * float64(pacer.sent-1) / pacer.rate))
	delay := target.Sub(pacer.now())
	if delay > 0 {
		pacer.sleep(delay)
	}
}
