package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"user-manager-api/internal/domain/user"
	"user-manager-api/internal/infrastructure/db/postgres"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) user.Repository {
	return &Repository{db: db}
}

func (r *Repository) FetchUsers(ctx context.Context, page int) (user.Users, error) {
	rows, err := r.db.Query(ctx, SelectUsers, page)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var us Users
	for rows.Next() {
		u := new(User)

		if err = rows.Scan(
			&u.ID,
			&u.UUID,
			&u.Email,
			&u.PasswordHash,
			&u.Role,
			&u.Name,
			&u.Lastname,
			&u.BirthDate,
			&u.Phone,

			&u.CreatedAt,
			&u.UpdatedAt,

			&u.DeletedAt,
			&u.DeletedReason,
			&u.DeletedBy,
		); err != nil {
			return nil, err
		}

		us = append(us, u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return fromDBModels(&us), nil
}

func (r *Repository) FetchUserByID(ctx context.Context, uuid user.UUID) (*user.User, error) {
	u := new(User)
	err := r.db.QueryRow(ctx, SelectUserByID, uuid.String()).Scan(
		&u.ID,
		&u.UUID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Name,
		&u.Lastname,
		&u.BirthDate,
		&u.Phone,

		&u.CreatedAt,
		&u.UpdatedAt,

		&u.DeletedAt,
		&u.DeletedReason,
		&u.DeletedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return fromDBModel(u), err
}

func (r *Repository) FetchUserByEmail(ctx context.Context, email string) (*user.User, error) {
	u := new(User)
	err := r.db.QueryRow(ctx, SelectUserByEmail, email).Scan(
		&u.ID,
		&u.UUID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Name,
		&u.Lastname,
		&u.BirthDate,
		&u.Phone,

		&u.CreatedAt,
		&u.UpdatedAt,

		&u.DeletedAt,
		&u.DeletedReason,
		&u.DeletedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return fromDBModel(u), err
}

func (r *Repository) CreateUser(ctx context.Context, req user.User) (*user.User, error) {
	u := new(User)

	err := r.db.QueryRow(
		ctx,
		InsertUser,
		req.Email, req.Name, req.Lastname, req.BirthDate, req.Phone,
	).Scan(
		&u.ID,
		&u.UUID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Name,
		&u.Lastname,
		&u.BirthDate,
		&u.Phone,

		&u.CreatedAt,
		&u.UpdatedAt,

		&u.DeletedAt,
		&u.DeletedReason,
		&u.DeletedBy,
	)
	if err != nil {
		if postgres.IsPgUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	return fromDBModel(u), err
}

func (r *Repository) UpdateUser(ctx context.Context, req user.User) (*user.User, error) {
	u := new(User)

	err := r.db.QueryRow(ctx, UpdateUserByUUID,
		req.Email, req.Name, req.Lastname, req.BirthDate, req.Phone, req.UUID,
	).Scan(
		&u.ID,
		&u.UUID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Name,
		&u.Lastname,
		&u.BirthDate,
		&u.Phone,

		&u.CreatedAt,
		&u.UpdatedAt,

		&u.DeletedAt,
		&u.DeletedReason,
		&u.DeletedBy,
	)
	if err != nil {
		if postgres.IsPgUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return fromDBModel(u), err
}

func (r *Repository) FetchInternalID(ctx context.Context, uuid user.UUID) (user.ID, error) {
	var id uint64
	if err := r.db.QueryRow(ctx, SelectIdByUUID, uuid.String()).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("user not found by uuid %s: %w", uuid.String(), err)
		}
		return 0, err
	}

	return user.ID(id), nil
}

func (r *Repository) DeleteUser(ctx context.Context, id user.ID) (*user.User, error) {
	u := new(User)
	err := r.db.QueryRow(ctx, SoftDeleteUserByID, id).Scan(
		&u.ID,
		&u.UUID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Name,
		&u.Lastname,
		&u.BirthDate,
		&u.Phone,

		&u.CreatedAt,
		&u.UpdatedAt,

		&u.DeletedAt,
		&u.DeletedReason,
		&u.DeletedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return fromDBModel(u), err
}
