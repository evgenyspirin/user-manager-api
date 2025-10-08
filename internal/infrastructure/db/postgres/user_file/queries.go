package user_file

const (
	SelectUserFiles = `
		SELECT id, uuid, user_id, bucket, storage_key, file_name, mime_type, size_bytes, download_url, created_at, deleted_at
		FROM user_files
		WHERE user_id = $1 AND deleted_at IS NULL
		LIMIT 50 OFFSET ( ($2 - 1) * 50 )
	`
	InsertUserFile = `
		INSERT INTO user_files (user_id, bucket, storage_key, file_name, mime_type, size_bytes, download_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
		  id, uuid, user_id, bucket, storage_key, file_name, mime_type, size_bytes, download_url, created_at, deleted_at
	`
	SoftDeleteUserFiles = `
		UPDATE user_files
		SET deleted_at = now()
		WHERE user_id = $1 AND deleted_at IS NULL
	`
)
