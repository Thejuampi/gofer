package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Thejuampi/amps-client-go/amps"
	kerberosauth "github.com/Thejuampi/amps-client-go/amps/auth/kerberos"
)

type compatBool struct {
	set   bool
	value bool
}

func (value *compatBool) String() string {
	if value == nil {
		return "false"
	}
	if value.value {
		return "true"
	}
	return "false"
}

func (value *compatBool) Set(raw string) error {
	parsed, err := parseCompatBool(raw)
	if err != nil {
		return err
	}
	value.set = true
	value.value = parsed
	return nil
}

func parseCompatBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "t", "true", "y", "yes":
		return true, nil
	case "0", "f", "false", "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", raw)
	}
}

type transportOptions struct {
	server           string
	proto            string
	messageType      string
	uriopts          string
	urischeme        string
	authenticatorRaw string
	secure           compatBool
}

func addTransportFlags(fs *flag.FlagSet, options *transportOptions, includeAuthenticator bool) {
	fs.StringVar(&options.server, "server", "", "AMPS server to connect to")
	fs.StringVar(&options.proto, "proto", "", "protocol to use")
	fs.StringVar(&options.proto, "prot", "", "protocol alias")
	fs.StringVar(&options.messageType, "type", "", "message type to use")
	fs.Var(&options.secure, "secure", "whether to use tcps (true/false/yes/no/1/0)")
	fs.StringVar(&options.uriopts, "uriopts", "", "custom AMPS URI query parameters")
	fs.StringVar(&options.urischeme, "urischeme", "", "custom URI scheme")
	if includeAuthenticator {
		fs.StringVar(&options.authenticatorRaw, "authenticator", "", "Go authenticator name")
	}
}

func (options transportOptions) canonicalURI() (string, error) {
	if strings.TrimSpace(options.server) == "" {
		return "", fmt.Errorf("server URI is required (-server flag)")
	}

	parsed, err := parseServerReference(options.server)
	if err != nil {
		return "", err
	}

	scheme, err := options.effectiveScheme(parsed.Scheme)
	if err != nil {
		return "", err
	}
	parsed.Scheme = scheme
	parsed.Path = options.effectivePath(parsed.Path)
	parsed.RawQuery = mergeQueryString(parsed.RawQuery, options.uriopts)

	if parsed.Host == "" {
		return "", fmt.Errorf("server URI %q is missing host", options.server)
	}

	return parsed.String(), nil
}

func (options transportOptions) effectiveScheme(current string) (string, error) {
	var scheme = strings.ToLower(strings.TrimSpace(current))
	if scheme == "" {
		scheme = "tcp"
	}

	if options.secure.set {
		if options.secure.value {
			scheme = "tcps"
		} else {
			scheme = "tcp"
		}
	}

	if strings.TrimSpace(options.urischeme) != "" {
		scheme = strings.ToLower(strings.TrimSpace(options.urischeme))
	}

	if scheme != "tcp" && scheme != "tcps" {
		return "", fmt.Errorf("unsupported URI scheme %q (supported: tcp, tcps)", scheme)
	}
	return scheme, nil
}

func (options transportOptions) effectivePath(current string) string {
	if normalized := options.normalizedMessageType(); normalized != "" {
		return "/amps/" + normalized
	}
	if strings.TrimSpace(current) != "" {
		return current
	}
	return "/amps"
}

func (options transportOptions) normalizedMessageType() string {
	var messageType = strings.ToLower(strings.TrimSpace(options.messageType))
	if messageType == "" {
		messageType = strings.ToLower(strings.TrimSpace(options.proto))
	}
	switch messageType {
	case "", "amps":
		return ""
	case "json":
		return "json"
	default:
		return messageType
	}
}

func (options transportOptions) authenticator() (amps.Authenticator, error) {
	return resolveAuthenticator(options.authenticatorRaw)
}

func resolveAuthenticator(name string) (amps.Authenticator, error) {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "", "default", "com.crankuptheamps.spark.defaultauthenticatorfactory":
		return nil, nil
	case "kerberos":
		return kerberosauth.NewAuthenticator(kerberosauth.Config{}), nil
	default:
		return nil, errors.New("unsupported authenticator; supported values: default, kerberos")
	}
}

func parseServerReference(raw string) (*url.URL, error) {
	var trimmed = strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("server URI is required (-server flag)")
	}
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return nil, fmt.Errorf("parse server URI: %w", err)
		}
		return parsed, nil
	}

	var user *url.Userinfo
	var host = trimmed
	if at := strings.LastIndex(trimmed, "@"); at > 0 {
		host = trimmed[at+1:]
		userInfo := trimmed[:at]
		if colon := strings.Index(userInfo, ":"); colon >= 0 {
			user = url.UserPassword(userInfo[:colon], userInfo[colon+1:])
		} else {
			user = url.User(userInfo)
		}
	}

	return &url.URL{
		Host: host,
		User: user,
	}, nil
}

func mergeQueryString(existing string, extra string) string {
	existing = strings.TrimSpace(existing)
	extra = strings.TrimSpace(extra)
	switch {
	case existing == "":
		return extra
	case extra == "":
		return existing
	default:
		return existing + "&" + extra
	}
}

type copyPublisher struct {
	client *amps.Client
}

func newCopyPublisher(copyServer string, options transportOptions, timeout time.Duration) (*copyPublisher, error) {
	if strings.TrimSpace(copyServer) == "" {
		return nil, nil
	}
	var copyOptions = options
	copyOptions.server = copyServer
	client, _, err := connect(copyOptions, timeout)
	if err != nil {
		return nil, err
	}
	return &copyPublisher{client: client}, nil
}

func (publisher *copyPublisher) Close() {
	if publisher == nil || publisher.client == nil {
		return
	}
	_ = publisher.client.Close()
}

func (publisher *copyPublisher) Publish(topic string, data []byte, delta bool) error {
	if publisher == nil || publisher.client == nil {
		return nil
	}
	if delta {
		return publisher.client.DeltaPublishBytes(topic, data)
	}
	return publisher.client.Publish(topic, string(data))
}

func parseUintOrDefault(raw string, defaultValue uint) (uint, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(value), nil
}
