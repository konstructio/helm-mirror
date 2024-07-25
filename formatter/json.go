package formatter

import (
	"bytes"
	jsonencoding "encoding/json"
	"fmt"
	"log"
	"strings"
)

type json struct {
	fileName string
	l        *log.Logger
}

func newJSONFormatter(fileName string, logger *log.Logger) *json {
	return &json{
		fileName: fileName,
		l:        logger,
	}
}

func (f *json) Output(b bytes.Buffer) error {
	imgs := strings.Split(b.String(), "\n")
	var images Images
	for _, i := range imgs {
		if i != "" {
			images.Names = append(images.Names, i)
		}
	}
	j, err := jsonencoding.Marshal(images)
	if err != nil {
		f.l.Printf("error: cannot encode json")
		return fmt.Errorf("cannot encode json: %w", err)
	}
	err = writeFile(f.fileName, j, f.l)
	if err != nil {
		return err
	}
	return nil
}
