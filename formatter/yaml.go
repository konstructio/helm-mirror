package formatter

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	yamlencoder "gopkg.in/yaml.v3"
)

type yaml struct {
	fileName string
	l        *log.Logger
}

func newYamlFormatter(fileName string, logger *log.Logger) *yaml {
	return &yaml{
		fileName: fileName,
		l:        logger,
	}
}

func (f *yaml) Output(b bytes.Buffer) error {
	imgs := strings.Split(b.String(), "\n")
	var images Images
	for _, i := range imgs {
		if i != "" {
			images.Names = append(images.Names, i)
		}
	}
	encoded, err := yamlencoder.Marshal(images)
	if err != nil {
		f.l.Printf("error: cannot encode yaml")
		return fmt.Errorf("cannot encode yaml: %w", err)
	}

	if err := writeFile(f.fileName, encoded, f.l); err != nil {
		return fmt.Errorf("cannot write to file %q: %w", f.fileName, err)
	}
	return nil
}
