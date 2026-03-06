package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
)

// defaultTimeout is applied when no explicit timeout is provided.
const defaultTimeout = 10 * time.Second

// connect creates a new AMPS client, connects to the given URI, and performs
// logon. On success the caller owns the client and must call Close.
func connect(options transportOptions, timeout time.Duration) (*amps.Client, string, error) {
	uri, err := options.canonicalURI()
	if err != nil {
		return nil, "", err
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	// Use a unique client name per process so command-ID deduplication in
	// servers/fakeamps does not confuse separate CLI invocations that happen
	// to generate the same auto-incremented commandID.
	clientName := fmt.Sprintf("gofer-%d-%d", os.Getpid(), time.Now().UnixNano())
	client := amps.NewClient(clientName)
	authenticator, err := options.authenticator()
	if err != nil {
		return nil, "", err
	}
	if err := client.Connect(uri); err != nil {
		return nil, "", fmt.Errorf("connect: %w", err)
	}

	logonDone := make(chan error, 1)
	go func() {
		if authenticator != nil {
			logonDone <- client.Logon(amps.LogonParams{Authenticator: authenticator})
			return
		}
		logonDone <- client.Logon()
	}()

	select {
	case err := <-logonDone:
		if err != nil {
			_ = client.Close()
			return nil, "", fmt.Errorf("logon: %w", err)
		}
	case <-time.After(timeout):
		_ = client.Close()
		return nil, "", fmt.Errorf("logon timed out after %v", timeout)
	}

	return client, uri, nil
}
