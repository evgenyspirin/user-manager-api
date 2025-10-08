package user_file

import (
	"user-manager-api/internal/domain/user_file"
)

func ToResponseUserFile(uDomain user_file.UserFile) UserFile {
	var uf = UserFile{
		UUID:        uDomain.UUID,
		FileName:    uDomain.FileName,
		MimeType:    uDomain.MimeType,
		SizeBytes:   uDomain.SizeBytes,
		StorageKey:  uDomain.StorageKey,
		DownloadURL: uDomain.DownloadURL,
	}

	return uf
}

func ToResponseUserFiles(ufDomain user_file.UserFiles) UserFiles {
	ufs := make(UserFiles, len(ufDomain))
	for idx, u := range ufDomain {
		ufs[idx] = ToResponseUserFile(*u)
	}

	return ufs
}
