package ports

import (
	"user-manager-api/internal/domain/user"
)

type Auth interface {
	GenerateToken(u *user.User, requestPassword string) (string, error)
}
