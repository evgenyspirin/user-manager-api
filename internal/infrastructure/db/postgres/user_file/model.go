package user_file

import (
	"time"
	userDB "user-manager-api/internal/infrastructure/db/postgres/user"

	"github.com/google/uuid"
)

type (
	UserFile struct {
		ID     uint64
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
