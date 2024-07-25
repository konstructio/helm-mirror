package service

import (
	"bytes"

	"github.com/pkg/errors"
)

type mockFormatter struct{}

func (m *mockFormatter) Output(buffer bytes.Buffer) error {
	if buffer.String() == "test" {
		return errors.New("not implemented")
	}
	return nil
}
