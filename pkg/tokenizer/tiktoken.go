package tokenizer

import (
	"encoding/json"
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

// TiktokenTokenizer implements Tokenizer using tiktoken-go BPE encoding.
type TiktokenTokenizer struct {
	encodingName string
	bpe          *tiktoken.Tiktoken
}

// NewTiktokenTokenizer creates a new TiktokenTokenizer for the given encoding (e.g. "cl100k_base", "o200k_base").
func NewTiktokenTokenizer(encodingName string) (*TiktokenTokenizer, error) {
	if encodingName == "" {
		encodingName = "cl100k_base"
	}

	bpe, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tiktoken encoding %q: %w", encodingName, err)
	}

	return &TiktokenTokenizer{
		encodingName: encodingName,
		bpe:          bpe,
	}, nil
}

// CountTokens returns the exact number of tokens in the given text string.
func (t *TiktokenTokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	tokens := t.bpe.Encode(text, nil, nil)
	return len(tokens)
}

// CountJSONTokens serializes a value to JSON and counts its tokens.
func (t *TiktokenTokenizer) CountJSONTokens(v any) int {
	if v == nil {
		return 0
	}
	data, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return t.CountTokens(string(data))
}

// Name returns the tokenizer name.
func (t *TiktokenTokenizer) Name() string {
	return t.encodingName
}
