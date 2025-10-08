package ports

import (
	"context"

	"user-manager-api/internal/domain/user"
)

type UserService interface {
	FindUserByID(ctx context.Context, uuid user.UUID) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	FindUsers(ctx context.Context, page int) (user.Users, error)
	CreateUser(ctx context.Context, u user.User) (*user.User, error)
	UpdateUser(ctx context.Context, u user.User) (*user.User, error)
	DeleteUser(ctx context.Context, uuid user.UUID) error
}
