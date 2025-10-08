package user_file

import (
	"context"
	"user-manager-api/internal/domain/user"
	"user-manager-api/internal/domain/user_file"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) user_file.Repository {
	return &Repository{db: db}
}

func (r *Repository) FetchUserFiles(ctx context.Context, userID user.ID, page int) (user_file.UserFiles, error) {
	rows, err := r.db.Query(ctx, SelectUserFiles, userID, page)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ufs UserFiles
	for rows.Next() {
		uf := new(UserFile)

		if err = rows.Scan(
			&uf.ID,
			&uf.UUID,
			&uf.UserID,

			&uf.Bucket,
			&uf.StorageKey,
			&uf.FileName,
			&uf.MimeType,
			&uf.SizeBytes,
			&uf.DownloadURL,

			&uf.CreatedAt,
			&uf.DeletedAt,
		); err != nil {
			return nil, err
		}

		ufs = append(ufs, uf)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return fromDBModels(&ufs), nil
}

func (r *Repository) CreateUserFile(ctx context.Context, userID user.ID, req *user_file.UserFile) (*user_file.UserFile, error) {
	uf := new(UserFile)

	err := r.db.QueryRow(
		ctx,
		InsertUserFile,
		userID, req.Bucket, req.StorageKey, req.FileName, req.MimeType, req.SizeBytes, req.DownloadURL,
	).Scan(
		&uf.ID,
		&uf.UUID,
		&uf.UserID,

		&uf.Bucket,
		&uf.StorageKey,
		&uf.FileName,
		&uf.MimeType,
		&uf.SizeBytes,
		&uf.DownloadURL,

		&uf.CreatedAt,
		&uf.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	return fromDBModel(uf), err
}

func (r *Repository) DeleteUserFiles(ctx context.Context, userID user.ID) error {
	_, err := r.db.Exec(ctx, SoftDeleteUserFiles, userID)
	return err
}
