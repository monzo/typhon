package service

import (
	"errors"
	"github.com/monzo/typhon/examples/stringsvc/pkg/stringsvc"
	"strings"
)

type service struct{}

var ErrEmpty = errors.New("empty string")

func New() stringsvc.Service {
	return service{}
}

func (service) Uppercase(str string) (string, error) {
	if str == "" {
		return "", ErrEmpty
	}

	return strings.ToUpper(str), nil
}

func (service) Count(str string) (int, error) {
	return len(str), nil
}
