package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/zxchelik/room-booking-service/internal/model"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Upsert(ctx context.Context, db DBTX, user model.User) error {
	_, err := db.Exec(ctx, `
		INSERT INTO users (id, email, role, password_hash, created_at)
		VALUES ($1, $2, $3, $4, COALESCE($5, NOW()))
		ON CONFLICT (id) DO UPDATE
		SET email = EXCLUDED.email,
		    role = EXCLUDED.role,
		    password_hash = COALESCE(users.password_hash, EXCLUDED.password_hash)
	`, user.ID, user.Email, user.Role, user.PasswordHash, user.CreatedAt)
	return err
}

func (r *UserRepository) Create(ctx context.Context, db DBTX, user model.User) (model.User, error) {
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		INSERT INTO users (id, email, role, password_hash, created_at)
		VALUES ($1, $2, $3, $4, COALESCE($5, NOW()))
		RETURNING created_at
	`, user.ID, user.Email, user.Role, user.PasswordHash, user.CreatedAt).Scan(&createdAt)
	if err != nil {
		return model.User{}, err
	}
	user.CreatedAt = &createdAt
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, db DBTX, email string) (model.User, error) {
	var user model.User
	var passwordHash sql.NullString
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		SELECT id, email, role, password_hash, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Role, &passwordHash, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, err
	}
	if passwordHash.Valid {
		value := passwordHash.String
		user.PasswordHash = &value
	}
	user.CreatedAt = &createdAt
	return user, nil
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
