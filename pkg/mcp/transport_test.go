package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStdioDrainsStderr(t *testing.T) {
	if os.Getenv("MCPQ_STDERR_HELPER") == "1" {
		fmt.Fprint(os.Stderr, strings.Repeat("log", 1024*1024))
		var req Request
		if json.NewDecoder(os.Stdin).Decode(&req) != nil {
			os.Exit(2)
		}
		json.NewEncoder(os.Stdout).Encode(Response{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(`{}`)})
		os.Exit(0)
	}
	c, err := NewStdioClient(os.Args[0], []string{"-test.run=^TestStdioDrainsStderr$"}, []string{"MCPQ_STDERR_HELPER=1"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := c.SendRequest(ctx, "initialize", nil); err != nil {
		t.Fatal(err)
	}
}

func TestSSEInitializeNotification(t *testing.T) {
	messages := make(chan []byte, 4)
	initialized := make(chan Request, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: endpoint\ndata: /messages\n\n")
			w.(http.Flusher).Flush()
			for {
				select {
				case msg := <-messages:
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
					w.(http.Flusher).Flush()
				case <-r.Context().Done():
					return
				}
			}
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		if req.Method == "notifications/initialized" {
			initialized <- req
		} else {
			b, _ := json.Marshal(Response{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"test","version":"1"}}`)})
			messages <- b
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	c, err := NewSSEClient(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := c.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case req := <-initialized:
		if req.ID != nil {
			t.Fatal("notification has request ID")
		}
	case <-ctx.Done():
		t.Fatal("initialized notification missing")
	}
}
