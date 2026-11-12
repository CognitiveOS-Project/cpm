package daemon

import (
	"fmt"

	"github.com/CognitiveOS-Project/cogsdk/client/daemon"
)

// DefaultSocketPath returns the cognitiveosd socket path, honoring the
// COGNITIVEOS_SOCKET environment override.
func DefaultSocketPath() string {
	return daemon.DefaultSocketPath()
}

// Client is a thin wrapper around cogsdk/client/daemon.Client that preserves
// the existing SendMessage API for cpm.
type Client struct {
	*daemon.Client
}

// NewClient connects to the cognitiveosd socket.
func NewClient() (*Client, error) {
	c, err := daemon.Dial()
	if err != nil {
		return nil, err
	}
	return &Client{Client: c}, nil
}

// SendMessage sends a single envelope to the daemon, fire-and-forget.
func SendMessage(msgType string, payload interface{}) error {
	c, err := NewClient()
	if err != nil {
		return fmt.Errorf("daemon connection failed: %w", err)
	}
	defer c.Close()
	return c.SendMessage(msgType, payload)
}