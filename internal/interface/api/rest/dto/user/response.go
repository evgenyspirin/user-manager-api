package user

import (
	"time"

	"github.com/google/uuid"
)

type (
	User struct {
		UUID      uuid.UUID `json:"uuid"`
		Email     string    `json:"email"`
		Name      string    `json:"name"`
		Lastname  string    `json:"lastname"`
		BirthDate time.Time `json:"birth_date"`
		Phone     string    `json:"phone"`
	}
	Users        []User
	ResponseData struct {
		Data Users `json:"data"`
	}
)
