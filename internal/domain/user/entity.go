package user

import (
	"time"

	"github.com/google/uuid"
)

type (
	ID   uint64
	UUID = uuid.UUID
	User struct {
		UUID         UUID
		Email        string
		PasswordHash *string
		Role         string
		Name         string
		Lastname     string
		BirthDate    time.Time
		Phone        string

		CreatedAt time.Time
		UpdatedAt time.Time

		DeletedAt     *time.Time
		DeletedReason string
		DeletedBy     *ID
	}
	Users []*User
)
