package user

const (
	SelectUsers = `
		SELECT id, uuid, email, password_hash, role, name, lastname, birth_date, phone, created_at, updated_at, deleted_at, deleted_reason, deleted_by
		FROM users
		WHERE deleted_at IS NULL
		LIMIT 50 OFFSET ( ($1 - 1) * 50 )
	`
	SelectUserByID = `
		SELECT id, uuid, email, password_hash, role, name, lastname, birth_date, phone, created_at, updated_at, deleted_at, deleted_reason, deleted_by 
		FROM users 
		WHERE uuid = $1 AND deleted_at IS NULL
	`
	SelectUserByEmail = `
		SELECT id, uuid, email, password_hash, role, name, lastname, birth_date, phone, created_at, updated_at, deleted_at, deleted_reason, deleted_by 
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL
	`
	InsertUser = `
		INSERT INTO users (email, name, lastname, birth_date, phone, deleted_reason)
		VALUES ($1, $2, $3, $4, $5, '')
		RETURNING
		  id, uuid, email, password_hash, role, name, lastname, birth_date, phone, created_at, updated_at, deleted_at, deleted_reason, deleted_by
	`
	UpdateUserByUUID = `
		UPDATE users
		SET email = $1,
		    name = $2,
		    lastname = $3,
		    birth_date = $4,
		    phone = $5,
		    updated_at = now()
		WHERE uuid = $6 AND deleted_at IS NULL
		RETURNING
		  id, uuid, email, password_hash, role, name, lastname, birth_date, phone, created_at, updated_at, deleted_at, deleted_reason, deleted_by
	`
	SelectIdByUUID     = `SELECT id FROM users WHERE uuid = $1::uuid`
	SoftDeleteUserByID = `
		UPDATE users
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
		  id, uuid, email, password_hash, role, name, lastname, birth_date, phone, created_at, updated_at, deleted_at, deleted_reason, deleted_by
	`
)
