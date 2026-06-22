package wsx

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const defaultQueueSize = 16

type Client struct {
	conn         *websocket.Conn
	writeTimeout time.Duration
	queue        chan []byte
	done         chan struct{}
	once         sync.Once
}

func NewClient(ctx context.Context, conn *websocket.Conn, writeTimeout time.Duration, queueSize int) *Client {
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	if writeTimeout <= 0 {
		writeTimeout = 2 * time.Second
	}
	client := &Client{
		conn:         conn,
		writeTimeout: writeTimeout,
		queue:        make(chan []byte, queueSize),
		done:         make(chan struct{}),
	}
	go client.writeLoop(ctx)
	return client
}

func (c *Client) SendJSON(value any) bool {
	return c.Send(MustMarshal(value))
}

func (c *Client) SendError(message string) bool {
	return c.SendJSON(map[string]string{
		"type":  "error",
		"error": message,
	})
}

func (c *Client) SendPong(payload json.RawMessage) bool {
	if len(payload) == 0 {
		payload = MustMarshal(map[string]any{})
	}
	return c.SendJSON(map[string]any{
		"type":    "pong",
		"payload": payload,
	})
}

func (c *Client) Send(data []byte) bool {
	if c == nil || len(data) == 0 {
		return false
	}
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case <-c.done:
		return false
	case c.queue <- append([]byte{}, data...):
		return true
	default:
		c.Close(websocket.StatusPolicyViolation, "write queue full")
		return false
	}
}

func (c *Client) Close(code websocket.StatusCode, reason string) {
	if c == nil {
		return
	}
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close(code, reason)
	})
}

func (c *Client) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.Close(websocket.StatusNormalClosure, "bye")
			return
		case <-c.done:
			return
		case data := <-c.queue:
			writeCtx, cancel := context.WithTimeout(context.Background(), c.writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				c.Close(websocket.StatusPolicyViolation, "write failed")
				return
			}
		}
	}
}

func MustMarshal(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
