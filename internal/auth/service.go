package auth

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	membershipType "github.com/joekings2k/staypass-be/internal/membership"
	"github.com/joekings2k/staypass-be/internal/platform/config"
	"github.com/joekings2k/staypass-be/internal/platform/database"
	"github.com/joekings2k/staypass-be/internal/platform/hash"
	"github.com/joekings2k/staypass-be/internal/platform/logger"
	"github.com/joekings2k/staypass-be/internal/platform/token"
	userType "github.com/joekings2k/staypass-be/internal/user"
)

type AuthService struct {
	repo           AuthRepository
	userRepo       userType.UserRepository
	membershipRepo membershipType.MembershipRepository
	config         *config.Config
	hasher         hash.Hasher
	tokeMaker      token.Maker
	Tx             database.TransactionRunner
}

func NewAuthService(repo AuthRepository, userRepo userType.UserRepository, membershipRepo membershipType.MembershipRepository, cfg *config.Config, hasher hash.Hasher, tokeMaker token.Maker, tx database.TransactionRunner) *AuthService {
	return &AuthService{
		repo:           repo,
		userRepo:       userRepo,
		membershipRepo: membershipRepo,
		config:         cfg,
		hasher:         hasher,
		tokeMaker:      tokeMaker,
		Tx:             tx,
	}
}

type RegisterParams struct {
	Email     string
	Password  string
	Phone     string
	FirstName string
	LastName  string
}

func (s *AuthService) Register(ctx context.Context, registerParams RegisterParams) error {
	hashedPassword, err := s.hasher.Hash(registerParams.Password)
	if err != nil {
		return err
	}
	arg := userType.CreateUserParams{
		Phone:        registerParams.Phone,
		PasswordHash: &hashedPassword,
		Email:        &registerParams.Email,
		FullName:     fmt.Sprintf("%s %s", registerParams.FirstName, registerParams.LastName),
	}
	_, err = s.userRepo.CreateUser(ctx, arg)
	if err != nil {
		return err
	}

	return nil
}

type LoginParams struct {
	Identifier string
	Password   string
	UserAgent  *string
	IP         *netip.Addr
}
type GetUserAndEstatesResponse struct {
	User    LoginUserResponse
	Estates []membershipType.UserEstatesAndRoles
}
type LoginResult struct {
	AccessToken            string
	AccessTokenExpiresAt   time.Time
	RefereshToken          string
	RefereshTokenExpiresAt time.Time
}
type LoginUserResponse struct {
	Phone        string
	FullName     string
	Email        string
	IsSuperAdmin bool
}

func newUserResponse(userRes userType.User) LoginUserResponse {
	var email string
	if userRes.Email != nil {
		email = *userRes.Email
	}
	return LoginUserResponse{
		Phone:        userRes.Phone,
		FullName:     userRes.FullName,
		Email:        email,
		IsSuperAdmin: userRes.IsSuperAdmin,
	}
}

// identifier can be either email or phone number
func (s *AuthService) Login(ctx context.Context, params LoginParams) (LoginResult, error) {

	log := logger.LoggerFrom(ctx).With("op", "auth.Login")
	var user *userType.User
	var err error

	if isEmail(params.Identifier) {
		user, err = s.userRepo.GetUserByEmail(ctx, params.Identifier)
	} else {
		user, err = s.userRepo.GetUserByPhone(ctx, params.Identifier)
	}
	if err != nil {
		return LoginResult{}, ErrUnAuthorized
	}

	if user == nil || user.PasswordHash == nil {
		log.Warn("user not found")
		return LoginResult{}, ErrInvalidCredentials
	}
	err = s.hasher.Compare(*user.PasswordHash, params.Password)
	if err != nil {
		log.Warn("invalid password")
		return LoginResult{}, ErrInvalidCredentials
	}

	accessToken, accessPayload, err := s.tokeMaker.CreateToken(user.ID, s.config.AccessTokenTTL)
	if err != nil {
		return LoginResult{}, err
	}
	refreshToken, refreshPayload, err := s.tokeMaker.CreateToken(user.ID, s.config.RefreshTokenTTL)
	if err != nil {
		return LoginResult{}, err
	}
	hashedRefreshToken := hash.HashToken(refreshToken)

	sessionArgs := CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: hashedRefreshToken,
		UserAgent:        params.UserAgent,
		IP:               params.IP,
		ExpiresAt:        refreshPayload.ExpiresAt,
	}
	_, err = s.repo.CreateSession(ctx, sessionArgs)
	if err != nil {
		return LoginResult{}, err
	}

	response := LoginResult{
		AccessToken:            accessToken,
		AccessTokenExpiresAt:   accessPayload.ExpiresAt,
		RefereshToken:          refreshToken,
		RefereshTokenExpiresAt: refreshPayload.ExpiresAt,
	}

	return response, nil
}

// to return the user and the estates

func (s *AuthService) GetUserAndEstates(ctx context.Context, userID uuid.UUID) (*GetUserAndEstatesResponse, error) {
	// get the user from the context
	log := logger.LoggerFrom(ctx).With("op", "auth.GetUserAndEstates")
	userRes, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		log.Error("failed to get user", "error", err)
		return nil, ErrUserNotFound
	}

	estates, err := s.membershipRepo.ListUserEstateMemberships(ctx, userID)
	if err != nil {
		log.Error("failed to get estates", "error", err)
		return nil, err
	}

	return &GetUserAndEstatesResponse{
		User:    newUserResponse(*userRes),
		Estates: estates,
	}, nil
}

type RefreshParams struct {
	RefreshToken string
	UserAgent    *string
	IP           *netip.Addr
}

func (s *AuthService) Refresh(ctx context.Context, params RefreshParams) (LoginResult, error) {
	log := logger.LoggerFrom(ctx).With("op", "auth.Refresh")

	refreshTokenPayload, err := s.tokeMaker.VerifyToken(params.RefreshToken)
	if err != nil {
		log.Error("failed to verify refresh token", "error", err)
		return LoginResult{}, ErrInvalidCredentials
	}
	user, err := s.userRepo.GetUserByID(ctx, refreshTokenPayload.UserID)
	if err != nil {
		return LoginResult{}, ErrUserNotFound
	}
	// hash the refresh token
	hashedRefreshToken := hash.HashToken(params.RefreshToken)

	sessionResp, err := s.repo.GetActiveSessionUserByTokenHash(ctx, hashedRefreshToken, refreshTokenPayload.UserID)
	if err != nil {
		log.Error("failed to get active session by token hash", "error", err)
		return LoginResult{}, ErrInvalidCredentials
	}

	if sessionResp.ExpiresAt.Before(time.Now()) {
		log.Error("session expired")
		return LoginResult{}, ErrInvalidCredentials
	}

	accessToken, accessPayload, err := s.tokeMaker.CreateToken(user.ID, s.config.AccessTokenTTL)
	if err != nil {
		return LoginResult{}, err
	}

	newRefreshToken, newRefreshPayload, err := s.tokeMaker.CreateToken(user.ID, s.config.RefreshTokenTTL)
	if err != nil {
		return LoginResult{}, err
	}

	if err := s.repo.RevokeSession(ctx, sessionResp.ID); err != nil {
		log.Error("failed to revoke rotated session", "error", err, "session_id", sessionResp.ID)
		return LoginResult{}, err
	}

	if _, err := s.repo.CreateSession(ctx, CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: hash.HashToken(newRefreshToken),
		UserAgent:        params.UserAgent,
		IP:               params.IP,
		ExpiresAt:        newRefreshPayload.ExpiresAt,
	}); err != nil {
		log.Error("failed to create rotated session", "error", err)
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken:            accessToken,
		AccessTokenExpiresAt:   accessPayload.ExpiresAt,
		RefereshToken:          newRefreshToken,
		RefereshTokenExpiresAt: newRefreshPayload.ExpiresAt,
	}, nil

}

func (s *AuthService) Claim(ctx context.Context, tokenHash string, password string) error {
	return nil

}
func isEmail(identifier string) bool {
	return strings.Contains(identifier, "@")
}
