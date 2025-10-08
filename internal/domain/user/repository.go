package user

import (
	"context"
)

type Repository interface {
	FetchUserByID(ctx context.Context, uuid UUID) (*User, error)
	FetchUserByEmail(ctx context.Context, email string) (*User, error)
	FetchUsers(ctx context.Context, page int) (Users, error)
	CreateUser(ctx context.Context, req User) (*User, error)
	UpdateUser(ctx context.Context, req User) (*User, error)
	FetchInternalID(ctx context.Context, uuid UUID) (ID, error)
	DeleteUser(ctx context.Context, uuid ID) (*User, error)
}
