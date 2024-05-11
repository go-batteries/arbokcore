package tokens

import (
	"arbokcore/core/database"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type Token struct {
	ResouceID    string  `db:"resource_id"`
	ResouceType  string  `db:"resource_type"`
	AccessToken  string  `db:"access_token"`
	RefreshToken string  `db:"refresh_token"`
	TokenType    string  `db:"token_type"`
	UserID       *string `db:"user_id"`

	AccessExpiresAt  time.Time `db:"access_expires_at"`
	RefreshExpiresAt time.Time `db:"refresh_expires_at"`

	database.Timestamp
}

func (token *Token) IsExpired() bool {
	return token.AccessExpiresAt.Before(database.Now()) ||
		token.RefreshExpiresAt.Before(database.Now())
}

func (token *Token) HasAccessExpired() bool {
	return token.AccessExpiresAt.Before(database.Now())
}

const (
	ResourceTypeUser   string = "user"
	ResourceTypeStream string = "stream"
)

func ValidateResourceType(resourceType string) bool {
	return resourceType == ResourceTypeUser || resourceType == ResourceTypeStream
}

func GenerateToken(tokenLen int) (string, error) {
	b := make([]byte, tokenLen)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

var (
	ErrTokenValidationFailed = errors.New("validation_failed")
)

const (
	AccessExpiryDuration = 24 * time.Hour
	ShortExpiryDuration  = 20 * time.Minute
	LongExpiryDuration   = 30 * 24 * time.Hour
)

type TokenOpts func(*Token)

func WithResource(resourceID, resourceType string) TokenOpts {
	return func(t *Token) {
		t.ResouceID = resourceID
		t.ResouceType = resourceType
	}
}

func WithShortExpiry() TokenOpts {
	return func(token *Token) {
		token.AccessExpiresAt = database.Now().Add(ShortExpiryDuration)
	}
}

func WithDefaultExpiry() TokenOpts {
	return func(token *Token) {
		token.AccessExpiresAt = database.Now().Add(AccessExpiryDuration)
	}
}

func WithUserID(userID *string) TokenOpts {
	return func(t *Token) {
		t.UserID = userID
	}
}

func NewToken(resourceID, resourceType string, opts ...TokenOpts) (*Token, error) {
	resourceType = strings.ToLower(resourceType)
	if !ValidateResourceType(resourceType) {
		return nil, ErrTokenValidationFailed
	}

	accessToken, err := GenerateToken(32)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateToken(64)
	if err != nil {
		return nil, err
	}

	token := &Token{
		ResouceID:        resourceID,
		ResouceType:      resourceType,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: database.Now().Add(LongExpiryDuration),
		Timestamp:        database.NewTimestamp(),
	}

	for _, opt := range opts {
		opt(token)
	}

	return token, nil
}
