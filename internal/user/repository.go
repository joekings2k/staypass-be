package user

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	uuid "github.com/vgarvardt/pgx-google-uuid/v5"
)

type User struct {
	ID           uuid.UUID
	Phone        string
	Email        *string
	FullName     string
	IsSuperAdmin bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
type CreateUserParams struct {
	Phone        string
	FullName     string
	Email        *string
	PasswordHash *string
}

type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, params CreateUserParams) (*User, error)
}

type pgUserRepository struct {
	db *dbgen.Queries
}

func NewPgUserRepository(db *dbgen.Queries) UserRepository {
	return &pgUserRepository{
		db: db,
	}
}

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already registered")
)

func (r *pgUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	dbUser, err := r.db.GetUserByID(ctx, toGetUserParams(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return ToDomainUser(dbUser), nil
}
func toGetUserParams(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

func ToDomainUser(dbUser dbgen.User) *User {
	var deletedAt *time.Time
	if dbUser.DeletedAt.Valid {
		dt := dbUser.DeletedAt.Time
		deletedAt = &dt
	}
	return &User{
		ID:           dbUser.ID.Bytes,
		Phone:        dbUser.Phone,
		Email:        dbUser.Email,
		FullName:     dbUser.FullName,
		IsSuperAdmin: dbUser.IsSuperAdmin,
		CreatedAt:    dbUser.CreatedAt.Time,
		UpdatedAt:    dbUser.UpdatedAt.Time,
		DeletedAt:    deletedAt,
	}

}

func (r *pgUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	dbUser, err := r.db.GetUserByEmail(ctx, &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return ToDomainUser(dbUser), nil
}

func (r *pgUserRepository) CreateUser(ctx context.Context, params CreateUserParams) (*User, error) {
	createdUser, err := r.db.CreateUser(ctx, toCreateUserParams(params))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return ToDomainUser(createdUser), nil
}

func toCreateUserParams(params CreateUserParams) dbgen.CreateUserParams {
	return dbgen.CreateUserParams{
		Phone:        params.Phone,
		Email:        params.Email,
		FullName:     params.FullName,
		PasswordHash: params.PasswordHash,
	}
}
