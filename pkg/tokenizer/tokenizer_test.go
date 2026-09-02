package tokenizer

import (
	"testing"
)

func TestTiktokenTokenizer(t *testing.T) {
	tok, err := NewTiktokenTokenizer("cl100k_base")
	if err != nil {
		t.Fatalf("failed to create TiktokenTokenizer: %v", err)
	}

	text := "Hello, world! Welcome to Model Context Protocol."
	count := tok.CountTokens(text)
	if count <= 0 {
		t.Errorf("expected count > 0, got %d", count)
	}

	jsonObj := map[string]any{
		"tool":   "list_items",
		"limit":  20,
		"cursor": "xyz",
	}
	jsonTokens := tok.CountJSONTokens(jsonObj)
	if jsonTokens <= 0 {
		t.Errorf("expected json tokens > 0, got %d", jsonTokens)
	}
}

func TestFastTokenizer(t *testing.T) {
	tok := NewFastTokenizer()
	text := "Hello, world! Welcome to Model Context Protocol."
	count := tok.CountTokens(text)
	if count <= 0 {
		t.Errorf("expected fast count > 0, got %d", count)
	}

	jsonObj := map[string]any{
		"tool":  "list_items",
		"limit": 20,
	}
	jsonTokens := tok.CountJSONTokens(jsonObj)
	if jsonTokens <= 0 {
		t.Errorf("expected fast json tokens > 0, got %d", jsonTokens)
	}
}
