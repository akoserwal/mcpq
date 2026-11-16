package proxy

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/akoserwal/mcpq/pkg/mcp"
)

// recordingBody observes bytes as the reverse proxy reads them, without changing
// framing or buffering delivery to the client. Oversized events still pass through.
type recordingBody struct {
	io.ReadCloser
	recorder   *Recorder
	sse        bool
	frame      []byte
	lineLength int
	oversized  bool
}

const maxRecordedResponse = 16 * 1024 * 1024

func (b *recordingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	for _, c := range p[:n] {
		if !b.oversized {
			if len(b.frame) == maxRecordedResponse {
				b.frame = nil
				b.oversized = true
			} else {
				b.frame = append(b.frame, c)
			}
		}
		if b.sse {
			if c == '\n' {
				if b.lineLength == 0 {
					if !b.oversized {
						b.recordEvent()
					}
					b.frame = b.frame[:0]
					b.oversized = false
				}
				b.lineLength = 0
			} else if c != '\r' {
				b.lineLength++
			}
		}
	}
	if err == io.EOF && !b.sse && !b.oversized && len(b.frame) > 0 {
		b.recordResponse(b.frame)
		b.frame = nil
	}
	return n, err
}

func (b *recordingBody) recordEvent() {
	var data []byte
	event := ""
	for _, line := range bytes.Split(b.frame, []byte("\n")) {
		line = bytes.TrimSuffix(line, []byte("\r"))
		field, value, _ := bytes.Cut(line, []byte(":"))
		value = bytes.TrimPrefix(value, []byte(" "))
		switch string(field) {
		case "event":
			event = string(value)
		case "data":
			data = append(data, value...)
			data = append(data, '\n')
		}
	}
	if event == "" || event == "message" {
		b.recordResponse(data)
	}
}

func (b *recordingBody) recordResponse(data []byte) {
	var message struct {
		mcp.Response
		Method string `json:"method"`
	}
	if json.Unmarshal(data, &message) == nil && message.JSONRPC == mcp.JSONRPCVersion && message.ID != nil && message.Method == "" && (len(message.Result) > 0 || message.Error != nil) {
		b.recorder.RecordResponse(&message.Response)
	}
}
