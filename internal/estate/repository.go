package estate

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
)

var (
	ErrEstateAlreadyExists     = errors.New("estate already exsists")
	ErrinternalServerError     = errors.New("internal server error")
	ErrEstateNotFound          = errors.New("estate not found")
	ErrInvalidEstateData       = errors.New("invalid estate data")
	ErrReferenceRecordNotFound = errors.New("reference record not found")
)

type Estate struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Address   *string
	City      *string
	State     *string
	Timezone  string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
type CreateEstateParams struct {
	Name     string
	Slug     string
	Address  *string
	City     *string
	State    *string
	Timezone string
}
type EstateRepository interface {
	CreateEstate(ctx context.Context, params CreateEstateParams) (*Estate, error)
	FindEstateByID(ctx context.Context, id uuid.UUID) (*Estate, error)
}

type pgEstateRepository struct {
	db *dbgen.Queries
}

func NewPgEstateRepository(db *dbgen.Queries) EstateRepository {
	return &pgEstateRepository{
		db: db,
	}
}

func (r *pgEstateRepository) CreateEstate(ctx context.Context, params CreateEstateParams) (*Estate, error) {
	dbEstate, err := r.db.CreateEstate(ctx, toCreateEstateParams(params))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation
				return nil, ErrEstateAlreadyExists
			case "23514": // check_violation
				return nil, ErrInvalidEstateData
			case "23503": // foreign_key_violation
				return nil, ErrReferenceRecordNotFound
			}
		}
		return nil, err
	}
	return toDomainEstate(dbEstate), nil
}

func toCreateEstateParams(params CreateEstateParams) dbgen.CreateEstateParams {
	timezone := params.Timezone
	if timezone == "" {
		timezone = "Africa/Lagos"
	}

	return dbgen.CreateEstateParams{
		Name:     params.Name,
		Slug:     params.Slug,
		Address:  params.Address,
		City:     params.City,
		State:    params.State,
		Timezone: timezone,
	}
}
func toDomainEstate(dbEstate dbgen.Estate) *Estate {
	return &Estate{
		ID:        dbEstate.ID.Bytes,
		Name:      dbEstate.Name,
		Slug:      dbEstate.Slug,
		Address:   dbEstate.Address,
		City:      dbEstate.City,
		State:     dbEstate.State,
		Timezone:  dbEstate.Timezone,
		IsActive:  dbEstate.IsActive,
		CreatedAt: dbEstate.CreatedAt.Time,
		UpdatedAt: dbEstate.UpdatedAt.Time,
	}
}

func (r *pgEstateRepository) FindEstateByID(ctx context.Context, id uuid.UUID) (*Estate, error) {
	return nil, nil
}
