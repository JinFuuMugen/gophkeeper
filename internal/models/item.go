package models

import (
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Encrypted []byte

	Metadata  string
	Version   int64
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
