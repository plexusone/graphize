// Package output provides formatters for CLI output.
// TOON format is the default for agent-friendly, token-efficient output.
package output

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Format represents an output format.
type Format string

const (
	FormatTOON Format = "toon"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// Formatter formats data for output.
type Formatter interface {
	Format(v any) ([]byte, error)
}

// NewFormatter creates a formatter for the specified format.
func NewFormatter(format Format) (Formatter, error) {
	switch format {
	case FormatTOON, "":
		return &TOONFormatter{}, nil
	case FormatJSON:
		return &JSONFormatter{}, nil
	case FormatYAML:
		return &YAMLFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// TOONFormatter outputs in TOON format (token-efficient for LLMs).
type TOONFormatter struct{}

// Format formats data as TOON.
// TODO: Use github.com/toon-format/toon-go when available.
// For now, we implement a simple TOON-like format.
func (f *TOONFormatter) Format(v any) ([]byte, error) {
	return formatTOON(v, 0)
}

// JSONFormatter outputs in JSON format.
type JSONFormatter struct {
	Indent bool
}

// Format formats data as JSON.
func (f *JSONFormatter) Format(v any) ([]byte, error) {
	if f.Indent {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}

// YAMLFormatter outputs in YAML format.
type YAMLFormatter struct{}

// Format formats data as YAML.
func (f *YAMLFormatter) Format(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

// formatTOON converts a value to TOON format.
// TOON is a token-efficient format designed for LLM consumption.
func formatTOON(v any, indent int) ([]byte, error) {
	// For now, use a simple implementation
	// TODO: integrate with toon-go library
	switch val := v.(type) {
	case map[string]any:
		return formatTOONMap(val, indent)
	case []any:
		return formatTOONSlice(val, indent)
	default:
		// Primitive values
		return []byte(fmt.Sprintf("%v", v)), nil
	}
}

func formatTOONMap(m map[string]any, indent int) ([]byte, error) {
	var result []byte
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	for k, v := range m {
		result = append(result, []byte(prefix+k+": ")...)
		valBytes, err := formatTOON(v, indent+1)
		if err != nil {
			return nil, err
		}
		result = append(result, valBytes...)
		result = append(result, '\n')
	}
	return result, nil
}

func formatTOONSlice(s []any, indent int) ([]byte, error) {
	var result []byte
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	for _, v := range s {
		result = append(result, []byte(prefix+"- ")...)
		valBytes, err := formatTOON(v, indent+1)
		if err != nil {
			return nil, err
		}
		result = append(result, valBytes...)
		result = append(result, '\n')
	}
	return result, nil
}
