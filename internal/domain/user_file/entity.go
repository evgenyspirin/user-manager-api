package user_file

import (
	"time"

	"github.com/google/uuid"

	userDB "user-manager-api/internal/infrastructure/db/postgres/user"
)

type (
	UserFile struct {
		UUID   uuid.UUID
		UserID *userDB.ID

		Bucket      string
		StorageKey  string
		FileName    string
		MimeType    string
		SizeBytes   uint64
		DownloadURL string

		CreatedAt time.Time
		DeletedAt *time.Time
	}
	UserFiles []*UserFile
)
