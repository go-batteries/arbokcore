package tokens

import (
	"arbokcore/core/database"
	"time"
)

var AdminToken = Token{
	ResouceID:    "U11223455",
	ResouceType:  "user",
	AccessToken:  "trial_access_token",
	RefreshToken: "trial_refresh_token",
	TokenType:    "user",

	AccessExpiresAt:  time.Now().Add(1 * time.Hour),
	RefreshExpiresAt: time.Now().Add(5 * time.Hour),

	Timestamp: database.NewTimestamp(),
}

var StreamToken = Token{
	ResouceID:    "S11223455",
	ResouceType:  "stream",
	AccessToken:  "trial_stream_access_token",
	RefreshToken: "trial_stream_refresh_token",
	TokenType:    "stream",
	UserID:       &AdminToken.ResouceID,

	AccessExpiresAt:  time.Now().Add(1 * time.Hour),
	RefreshExpiresAt: time.Now().Add(5 * time.Hour),

	Timestamp: database.NewTimestamp(),
}

var InvalidStreamToken = Token{
	ResouceID:    "S11223455",
	ResouceType:  "stream",
	AccessToken:  "trial_stream_access_token_invalid",
	RefreshToken: "trial_stream_refresh_token_invalid",
	TokenType:    "stream",

	AccessExpiresAt:  time.Now().Add(-1 * time.Hour),
	RefreshExpiresAt: time.Now().Add(5 * time.Hour),

	Timestamp: database.NewTimestamp(),
}
