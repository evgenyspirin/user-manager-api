package user_file

import (
	domain "user-manager-api/internal/domain/user_file"
)

func fromDBModel(model *UserFile) *domain.UserFile {
	var uf = &domain.UserFile{
		UUID:   model.UUID,
		UserID: model.UserID,

		Bucket:      model.Bucket,
		StorageKey:  model.StorageKey,
		FileName:    model.FileName,
		MimeType:    model.MimeType,
		SizeBytes:   model.SizeBytes,
		DownloadURL: model.DownloadURL,

		CreatedAt: model.CreatedAt,
		DeletedAt: model.DeletedAt,
	}

	return uf
}

func fromDBModels(models *UserFiles) domain.UserFiles {
	ufs := make(domain.UserFiles, len(*models))
	for idx, u := range *models {
		ufs[idx] = fromDBModel(u)
	}

	return ufs
}
