package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Envelope struct {
	Type    string          `json:"type"`
	From    string          `json:"from"`
	Payload json.RawMessage `json:"payload"`
}

func SendMessage(msgType string, payload interface{}) error {
	socketPath := "/cognitiveos/run/daemon.sock"
	if d := os.Getenv("COGNITIVEOS_SOCKET"); d != "" {
		socketPath = d
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("daemon connection failed: %w", err)
	}
	defer conn.Close()

	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	env := Envelope{
		Type:    msgType,
		From:    "cpm",
		Payload: payloadData,
	}

	envData, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope failed: %w", err)
	}

	_, err = conn.Write(append(envData, '\n'))
	return err
}
