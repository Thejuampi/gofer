package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
)

func TestTransportOptionsCanonicalURIWithType(t *testing.T) {
	options := transportOptions{
		server:      "localhost:9007",
		messageType: "json",
	}

	uri, err := options.canonicalURI()
	if err != nil {
		t.Fatalf("canonicalURI returned error: %v", err)
	}
	if uri != "tcp://localhost:9007/amps/json" {
		t.Fatalf("canonicalURI = %q, want %q", uri, "tcp://localhost:9007/amps/json")
	}
}

func TestTransportOptionsCanonicalURIUsesProtoAlias(t *testing.T) {
	options := transportOptions{
		server: "localhost:9007",
		proto:  "json",
	}

	uri, err := options.canonicalURI()
	if err != nil {
		t.Fatalf("canonicalURI returned error: %v", err)
	}
	if uri != "tcp://localhost:9007/amps/json" {
		t.Fatalf("canonicalURI = %q, want %q", uri, "tcp://localhost:9007/amps/json")
	}
}

func TestTransportOptionsCanonicalURIUsesQueryAndUserInfo(t *testing.T) {
	options := transportOptions{
		server:      "alice:secret@localhost:9007",
		messageType: "json",
		uriopts:     "tcp_nodelay=true&tcp_sndbuf=8192",
	}

	uri, err := options.canonicalURI()
	if err != nil {
		t.Fatalf("canonicalURI returned error: %v", err)
	}
	if uri != "tcp://alice:secret@localhost:9007/amps/json?tcp_nodelay=true&tcp_sndbuf=8192" {
		t.Fatalf("canonicalURI = %q", uri)
	}
}

func TestTransportOptionsCanonicalURIURISchemeOverridesSecure(t *testing.T) {
	options := transportOptions{
		server:      "localhost:9007",
		messageType: "json",
		secure:      sparkBool{set: true, value: true},
		urischeme:   "tcp",
	}

	uri, err := options.canonicalURI()
	if err != nil {
		t.Fatalf("canonicalURI returned error: %v", err)
	}
	if uri != "tcp://localhost:9007/amps/json" {
		t.Fatalf("canonicalURI = %q, want %q", uri, "tcp://localhost:9007/amps/json")
	}
}

func TestTransportOptionsCanonicalURIRejectsUnsupportedScheme(t *testing.T) {
	options := transportOptions{
		server:      "localhost:9007",
		messageType: "json",
		urischeme:   "protected",
	}

	_, err := options.canonicalURI()
	if err == nil {
		t.Fatalf("canonicalURI should reject unsupported custom schemes")
	}
}

func TestResolveAuthenticatorDefaultAndAlias(t *testing.T) {
	if authenticator, err := resolveAuthenticator(""); err != nil || authenticator != nil {
		t.Fatalf("resolveAuthenticator empty = (%v, %v), want (nil, nil)", authenticator, err)
	}
	if authenticator, err := resolveAuthenticator("default"); err != nil || authenticator != nil {
		t.Fatalf("resolveAuthenticator default = (%v, %v), want (nil, nil)", authenticator, err)
	}
	if authenticator, err := resolveAuthenticator("com.crankuptheamps.spark.DefaultAuthenticatorFactory"); err != nil || authenticator != nil {
		t.Fatalf("resolveAuthenticator java alias = (%v, %v), want (nil, nil)", authenticator, err)
	}
}

func TestResolveAuthenticatorRejectsUnknown(t *testing.T) {
	_, err := resolveAuthenticator("mystery")
	if err == nil {
		t.Fatalf("resolveAuthenticator should reject unknown names")
	}
	if !strings.Contains(err.Error(), "supported") {
		t.Fatalf("resolveAuthenticator error = %q, want supported-values hint", err)
	}
}

func TestSplitRecordsTrimsAndSkipsEmpty(t *testing.T) {
	records := splitRecords([]byte(" first | |second| third |"), byte('|'))
	if len(records) != 3 {
		t.Fatalf("splitRecords len = %d, want 3", len(records))
	}
	if string(records[0]) != "first" || string(records[1]) != "second" || string(records[2]) != "third" {
		t.Fatalf("splitRecords = %q, want trimmed non-empty records", records)
	}
}

func TestPublishPacerWaitsForRate(t *testing.T) {
	current := time.Unix(0, 0)
	var sleeps []time.Duration
	pacer := newPublishPacer(2, func() time.Time {
		return current
	}, func(delay time.Duration) {
		sleeps = append(sleeps, delay)
		current = current.Add(delay)
	})

	pacer.Wait()
	pacer.Wait()

	if len(sleeps) != 1 {
		t.Fatalf("sleep calls = %d, want 1", len(sleeps))
	}
	if sleeps[0] != 500*time.Millisecond {
		t.Fatalf("sleep[0] = %v, want %v", sleeps[0], 500*time.Millisecond)
	}
}

func TestRenderMessageUsesSparkTokens(t *testing.T) {
	message := amps.NewCommand("sow").
		SetTopic("orders").
		SetBookmark("1|1|").
		SetData([]byte(`{"id":1}`)).
		GetMessage()

	output := string(renderMessage(message, "{command}:{topic}:{bookmark}:{length}:{data}", false))
	expected := `sow:orders:1|1|:8:{"id":1}`
	if output != expected {
		t.Fatalf("renderMessage = %q, want %q", output, expected)
	}
}
