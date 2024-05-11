package users

import (
	"arbokcore/core/database"
	"errors"

	"github.com/rs/zerolog/log"
)

type User struct {
	ID             string `db:"id"`
	Type           string `db:"user_type"`
	Email          string `db:"email"`
	HashedPassword string `db:"hashedpass"`

	database.Timestamp
}

const UserTypeNormal = "normal"

var (
	ErrIDGenerationFailed = errors.New("id_gen_failed")
	ErrPasswordHashFailed = errors.New("password_gen_failed")
)

func NewUser(email, password string) (*User, error) {
	id, err := database.NewID()
	if err != nil {
		log.Error().Err(err).Msg("failed to generate id")
		return nil, ErrIDGenerationFailed
	}

	hashedPassword, err := database.HashPassword(password)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash password")
		return nil, ErrPasswordHashFailed
	}

	now := database.Now()
	user := &User{
		ID:             id,
		Type:           UserTypeNormal,
		Email:          email,
		HashedPassword: hashedPassword,

		Timestamp: database.Timestamp{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	return user, nil
}
