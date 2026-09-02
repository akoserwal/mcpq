package tokenizer

import (
	"encoding/json"
)

// Tokenizer is the interface for counting tokens in strings and JSON payloads.
type Tokenizer interface {
	// CountTokens counts the number of tokens in a plain text string.
	CountTokens(text string) int

	// CountJSONTokens serializes a value to compact JSON and counts its tokens.
	CountJSONTokens(v any) int

	// Name returns the identifier of this tokenizer.
	Name() string
}

// DefaultTokenizer returns the standard BPE tokenizer (cl100k_base), falling back to FastTokenizer on error.
func DefaultTokenizer() Tokenizer {
	t, err := NewTiktokenTokenizer("cl100k_base")
	if err != nil {
		return NewFastTokenizer()
	}
	return t
}

// CountJSONHelper is a utility method to marshal data compactly and count with a given tokenizer.
func CountJSONHelper(t Tokenizer, v any) int {
	if v == nil {
		return 0
	}
	data, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return t.CountTokens(string(data))
}
