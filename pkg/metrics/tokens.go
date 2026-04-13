// Package metrics provides measurement utilities for source code analysis.
package metrics

import "unicode"

// EstimateTokens estimates the token count for text using heuristics.
//
// The estimation accounts for:
// - Word boundaries (identifiers, keywords)
// - Code punctuation (braces, parentheses, operators)
// - GPT tokenizer behavior (~1.5 tokens per word for code)
//
// This is an approximation; actual token counts depend on the specific
// tokenizer used by the target LLM.
func EstimateTokens(text string) int {
	// Count words (sequences of letters, digits, underscores)
	words := 0
	inWord := false

	for _, r := range text {
		isWordChar := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		if isWordChar && !inWord {
			words++
			inWord = true
		} else if !isWordChar {
			inWord = false
		}
	}

	// Count punctuation that typically becomes separate tokens
	punctuation := 0
	for _, r := range text {
		if unicode.IsPunct(r) || r == '{' || r == '}' || r == '(' || r == ')' {
			punctuation++
		}
	}

	// Code typically has ~1.5 tokens per word plus half the punctuation
	return int(float64(words)*1.5) + punctuation/2
}

// EstimateTokensInFile estimates tokens in a source file.
// This is a convenience wrapper that accepts byte content.
func EstimateTokensInFile(content []byte) int {
	return EstimateTokens(string(content))
}
