package ports

import (
	"context"
	"mime/multipart"

	"user-manager-api/internal/domain/user"
	"user-manager-api/internal/domain/user_file"
)

type UserFileService interface {
	FindUserFiles(ctx context.Context, userUUID user.UUID, page int) (user_file.UserFiles, error)
	CreateUserFile(ctx context.Context, userUUID user.UUID, in *multipart.FileHeader) (*user_file.UserFile, error)
	DeleteUserFiles(ctx context.Context, userUUID user.UUID) error
}
