package main

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/Thejuampi/amps-client-go/amps"
)

// writer is a buffered stdout writer for high-throughput output.
var writer = bufio.NewWriterSize(os.Stdout, 64*1024)

// flushOutput flushes the buffered writer to stdout.
func flushOutput() { _ = writer.Flush() }

// writeMessage writes a single message's data payload to stdout.
// When pretty is true and the data is valid JSON, it is indented.
func writeMessage(msg *amps.Message, pretty bool) {
	data := msg.Data()
	if len(data) == 0 {
		return
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
	_, _ = writer.Write(data)
	_ = writer.WriteByte('\n')
}
