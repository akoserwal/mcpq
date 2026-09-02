package tokenizer

import (
	"encoding/json"
	"math"
	"unicode"
)

// FastTokenizer implements a fast heuristic token estimation.
type FastTokenizer struct{}

// NewFastTokenizer creates a new FastTokenizer.
func NewFastTokenizer() *FastTokenizer {
	return &FastTokenizer{}
}

// CountTokens estimates token count using a calibrated character/word ratio.
func (f *FastTokenizer) CountTokens(text string) int {
	if len(text) == 0 {
		return 0
	}

	charCount := len(text)
	wordCount := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			if inWord {
				wordCount++
				inWord = false
			}
		} else {
			inWord = true
		}
	}
	if inWord {
		wordCount++
	}

	// Blend character-based (~3.8 chars/token) and word-based (~1.3 tokens/word) estimations
	charEstimate := float64(charCount) / 3.8
	wordEstimate := float64(wordCount) * 1.25

	estimated := (charEstimate*0.6 + wordEstimate*0.4)
	return int(math.Ceil(estimated))
}

// CountJSONTokens serializes a value to JSON and counts its estimated tokens.
func (f *FastTokenizer) CountJSONTokens(v any) int {
	if v == nil {
		return 0
	}
	data, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return f.CountTokens(string(data))
}

// Name returns the tokenizer name.
func (f *FastTokenizer) Name() string {
	return "fast_heuristic"
}
