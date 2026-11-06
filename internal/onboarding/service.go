package onboarding

import (
	"context"

	"github.com/google/uuid"

	"github.com/joekings2k/staypass-be/internal/estate"
	"github.com/joekings2k/staypass-be/internal/membership"
	"github.com/joekings2k/staypass-be/internal/platform/database"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/joekings2k/staypass-be/internal/platform/hash"
	"github.com/joekings2k/staypass-be/internal/platform/logger"
	"github.com/joekings2k/staypass-be/internal/user"
)

type OnboardingService struct {
	Tx     database.TransactionRunner
	Hasher hash.Hasher
}

func NewOnboardingService(tx database.TransactionRunner, hasher hash.Hasher) *OnboardingService {
	return &OnboardingService{
		Tx:     tx,
		Hasher: hasher,
	}
}

type RegisterParams struct {
	Phone      string
	FullName   string
	Password   string
	Email      string
	EstateName string
	Slug       string
	Address    string
	City       string
	State      string
}

func (s *OnboardingService) Register(ctx context.Context, params RegisterParams) (*estate.Estate, *user.User, *membership.Membership, error) {
	log := logger.LoggerFrom(ctx).With("op", "onboarding.Register", "slug", params.Slug)
	log.Info("starting estate onboarding", "estate_name", params.EstateName, "email", params.Email)

	hashedPassword, err := s.Hasher.Hash(params.Password)
	if err != nil {
		log.Error("failed to hash password", "error", err)
		return nil, nil, nil, err
	}

	var resultEstate *estate.Estate
	var resultUser *user.User
	var resultMembership *membership.Membership

	err = s.Tx.Run(ctx, func(q *dbgen.Queries) error {
		estateRepo := estate.NewPgEstateRepository(q)
		userRepo := user.NewPgUserRepository(q)
		membershipRepo := membership.NewMembershipRepository(q)

		estateArg := estate.CreateEstateParams{
			Name:    params.EstateName,
			Slug:    params.Slug,
			City:    &params.City,
			State:   &params.State,
			Address: &params.Address,
		}
		resultEstate, err = estateRepo.CreateEstate(ctx, estateArg)
		if err != nil {
			return err
		}

		userArg := user.CreateUserParams{
			Phone:        params.Phone,
			Email:        &params.Email,
			FullName:     params.FullName,
			PasswordHash: &hashedPassword,
		}
		resultUser, err = userRepo.CreateUser(ctx, userArg)
		if err != nil {
			return err
		}

		membershipArg := membership.CreateMembershipParams{
			EstateID: resultEstate.ID,
			UserID:   uuid.UUID(resultUser.ID),
			Role:     membership.MembershipRoleEstateAdmin,
		}
		resultMembership, err = membershipRepo.CreateMembership(ctx, membershipArg)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("estate onboarding failed, transaction rolled back", "error", err)
		return nil, nil, nil, err
	}

	log.Info("estate onboarding completed",
		"estate_id", resultEstate.ID,
		"user_id", resultUser.ID,
		"membership_id", resultMembership.ID,
	)
	return resultEstate, resultUser, resultMembership, nil

}

func (s *OnboardingService) Login(ctx context.Context, params RegisterParams) (*estate.Estate, *user.User, *membership.Membership, error) {
	return nil, nil, nil, nil
}
