package cgminer

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Dialer is abstract network dialer
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	Dial(network, address string) (net.Conn, error)
}

// ConnectError represents API connection error
type ConnectError struct {
	err error
}

// NewConnectError wraps error as ConnectError
func NewConnectError(baseError error) ConnectError {
	return ConnectError{err: baseError}
}

// Error implements error
func (err ConnectError) Error() string {
	return fmt.Sprintf("connect error (%s)", err.err)
}

// Unwrap implements error
func (err ConnectError) Unwrap() error {
	return err.err
}

// CGMiner is cgminer API client
type CGMiner struct {
	// Address is API endpoint address (host:port)
	Address string

	// Timeout is request timeout
	Timeout time.Duration

	// Dialer is network dialer
	Dialer Dialer

	// Transport is request and response decoder.
	//
	// CGMiner might have one of two API formats - JSON or plain text.
	// JSON is default one.
	Transport Transport
}

// Call sends command to cgminer API and writes result to passed response output
// or returns error.
//
// If command doesn't returns any response, nil "out" value should be passed.
//
// For context-based requests, use `CallContext()`
func (c *CGMiner) Call(cmd Command, out AbstractResponse) error {
	return c.CallContext(context.Background(), cmd, out)
}

// CallContext sends command to cgminer API using the provided context.
//
// If command doesn't returns any response, nil "out" value should be passed.
func (c *CGMiner) CallContext(ctx context.Context, cmd Command, out AbstractResponse) error {
	conn, err := c.Dialer.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return ConnectError{err: err}
	}

	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	if err = c.Transport.SendCommand(conn, cmd); err != nil {
		return fmt.Errorf("failed to send cgminer command: %w", err)
	}

	return c.Transport.DecodeResponse(conn, cmd, out)
}

// RawCall sends command to CGMiner API and returns raw response as slice of bytes.
//
// Response error check should be performed manually.
func (c *CGMiner) RawCall(ctx context.Context, cmd Command) ([]byte, error) {
	conn, err := c.Dialer.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return nil, ConnectError{err: err}
	}

	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	if err = c.Transport.SendCommand(conn, cmd); err != nil {
		return nil, err
	}

	return readWithNullTerminator(conn)
}

// NewCGMiner returns a CGMiner client with JSON API transport
func NewCGMiner(hostname string, port int, timeout time.Duration) *CGMiner {
	return &CGMiner{
		Address:   fmt.Sprintf("%s:%d", hostname, port),
		Timeout:   timeout,
		Transport: NewJSONTransport(),
		Dialer: &net.Dialer{
			Timeout: timeout,
		},
	}
}
