package database

import "time"

type Timestamp struct {
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

const DateFormat = time.RFC3339

func Now() time.Time {
	return time.Now().UTC()
}

func NewTimestamp() Timestamp {
	now := Now()

	return Timestamp{
		CreatedAt: now,
		UpdatedAt: now,
	}
}
