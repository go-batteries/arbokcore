package database

import (
	"crypto/rand"

	"github.com/oklog/ulid/v2"
)

func NewID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(Now()), rand.Reader)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
