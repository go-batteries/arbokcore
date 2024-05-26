package tokens

import (
	"arbokcore/core/database"
	"time"
)

var AdminToken = Token{
	ResourceID:   "U11223455",
	ResouceType:  "user",
	AccessToken:  "trial_access_token",
	RefreshToken: "trial_refresh_token",
	TokenType:    "user",
	DeviceID:     "1",

	AccessExpiresAt:  time.Now().Add(1 * time.Hour),
	RefreshExpiresAt: time.Now().Add(5 * time.Hour),

	Timestamp: database.NewTimestamp(),
}

var AnotherToken = Token{
	ResourceID:   "U11223456",
	ResouceType:  "user",
	AccessToken:  "trial_access_token_1",
	RefreshToken: "trial_refresh_token_1",
	TokenType:    "user",
	DeviceID:     "2",

	AccessExpiresAt:  time.Now().Add(1 * time.Hour),
	RefreshExpiresAt: time.Now().Add(5 * time.Hour),

	Timestamp: database.NewTimestamp(),
}

var StreamToken = Token{
	ResourceID:   "S11223455",
	ResouceType:  "stream",
	AccessToken:  "trial_stream_access_token",
	RefreshToken: "trial_stream_refresh_token",
	TokenType:    "stream",
	UserID:       &AdminToken.ResourceID,
	DeviceID:     "1",

	AccessExpiresAt:  time.Now().Add(1 * time.Hour),
	RefreshExpiresAt: time.Now().Add(5 * time.Hour),

	Timestamp: database.NewTimestamp(),
}

var InvalidStreamToken = Token{
	ResourceID:   "S11223455",
	ResouceType:  "stream",
	AccessToken:  "trial_stream_access_token_invalid",
	RefreshToken: "trial_stream_refresh_token_invalid",
	TokenType:    "stream",

	AccessExpiresAt:  time.Now().Add(-1 * time.Hour),
	RefreshExpiresAt: time.Now().Add(5 * time.Hour),

	Timestamp: database.NewTimestamp(),
}
