package user_file

import (
	"context"
	"user-manager-api/internal/domain/user"
)

type Repository interface {
	FetchUserFiles(ctx context.Context, userID user.ID, page int) (UserFiles, error)
	CreateUserFile(ctx context.Context, userID user.ID, req *UserFile) (*UserFile, error)
	DeleteUserFiles(ctx context.Context, userID user.ID) error
}
