package user_file

import (
	"github.com/google/uuid"
)

type (
	UserFile struct {
		UUID        uuid.UUID `json:"uuid"`
		FileName    string    `json:"file_name"`
		MimeType    string    `json:"mime_type"`
		SizeBytes   uint64    `json:"size_bytes"`
		StorageKey  string    `json:"storage_key"`
		DownloadURL string    `json:"download_url"`
	}
	UserFiles    []UserFile
	ResponseData struct {
		Data UserFiles `json:"data"`
	}
)
