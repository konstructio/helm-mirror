package formatter

import (
	"bytes"
	"fmt"
	"log"
	"os"
)

// Formatter defines the behavior for a Formatter
type Formatter interface {
	Output(buffer bytes.Buffer) error
}

// Type definition of formatter type enum
type Type int

// Enum for Formatter
const (
	StdoutType Type = 1 << iota
	FileType
	JSONType
	YamlType
	SkopeoType
)

// NewFormatter returns a new instance of formatter
//
//nolint:ireturn
func NewFormatter(t Type, fileName string, logger *log.Logger) Formatter {
	switch t {
	case StdoutType:
		return newStdoutFormatter(logger)
	case FileType:
		return newFileFormatter(fileName, logger)
	case JSONType:
		return newJSONFormatter(fileName, logger)
	case YamlType:
		return newYamlFormatter(fileName, logger)
	case SkopeoType:
		return newSkopeoFormatter(fileName, logger)
	default:
		return newStdoutFormatter(logger)
	}
}

func writeFile(name string, content []byte, log *log.Logger) error {
	err := os.WriteFile(name, content, 0o600)
	if err != nil {
		log.Printf("cannot write files %s: %s", name, err)
		return fmt.Errorf("cannot write files %s: %w", name, err)
	}
	return nil
}

// Images struct for YAML and JSON output
type Images struct {
	Names []string `json:"names,omitempty" yaml:"names,omitempty"`
}
