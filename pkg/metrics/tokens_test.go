package metrics

import "testing"

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "empty",
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "single word",
			text:    "hello",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "simple sentence",
			text:    "hello world",
			wantMin: 2,
			wantMax: 5,
		},
		{
			name:    "go function",
			text:    "func main() { fmt.Println() }",
			wantMin: 5,
			wantMax: 20,
		},
		{
			name:    "code with punctuation",
			text:    "if x := getValue(); x != nil { return x }",
			wantMin: 10,
			wantMax: 30,
		},
		{
			name:    "identifier with underscores",
			text:    "my_long_variable_name",
			wantMin: 1,
			wantMax: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateTokens(%q) = %d, want between %d and %d",
					tt.text, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestEstimateTokensInFile(t *testing.T) {
	content := []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}")
	tokens := EstimateTokensInFile(content)

	// Should have reasonable token count for this code
	if tokens < 5 || tokens > 30 {
		t.Errorf("EstimateTokensInFile() = %d, want between 5 and 30", tokens)
	}
}

func TestEstimateTokensConsistency(t *testing.T) {
	// Longer text should have more tokens
	short := EstimateTokens("func a() {}")
	long := EstimateTokens("func processUserRequest(ctx context.Context, req *Request) (*Response, error) {}")

	if long <= short {
		t.Errorf("longer code (%d tokens) should have more tokens than shorter code (%d tokens)",
			long, short)
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	// Sample Go code
	code := `package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	for _, arg := range args {
		fmt.Println(arg)
	}
}
`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(code)
	}
}
