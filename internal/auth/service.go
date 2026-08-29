package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	membershipType "github.com/joekings2k/staypass-be/internal/membership"
	"github.com/joekings2k/staypass-be/internal/platform/config"
	"github.com/joekings2k/staypass-be/internal/platform/hash"
	"github.com/joekings2k/staypass-be/internal/platform/token"
	"github.com/joekings2k/staypass-be/internal/platform/web"
	userType "github.com/joekings2k/staypass-be/internal/user"
)

type AuthService struct {
	repo           AuthRepository
	userRepo       userType.UserRepository
	membershipRepo membershipType.MembershipRepository
	config         *config.Config
	hasher         hash.Hasher
	tokeMaker      token.Maker
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func NewAuthService(repo AuthRepository, userRepo userType.UserRepository, membershipRepo membershipType.MembershipRepository, cfg *config.Config, hasher hash.Hasher, tokeMaker token.Maker) *AuthService {
	return &AuthService{
		repo:           repo,
		userRepo:       userRepo,
		membershipRepo: membershipRepo,
		config:         cfg,
		hasher:         hasher,
		tokeMaker:      tokeMaker,
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
type LoginResult struct {
	AccessToken            string
	AccessTokenExpiresAt   time.Time
	RefereshToken          string
	RefereshTokenExpiresAt time.Time
	User                   LoginUserResponse
	Estates                []membershipType.UserEstatesAndRoles
}
type LoginUserResponse struct {
	Phone        string
	FullName     string
	Email        string
	IsSuperAdmin bool
}

func newUserResponse(userRes userType.User) LoginUserResponse {
	return LoginUserResponse{
		Phone:        userRes.Phone,
		FullName:     userRes.FullName,
		Email:        *userRes.Email,
		IsSuperAdmin: userRes.IsSuperAdmin,
	}
}

// identifier can be either email or phone number
func (s *AuthService) Login(ctx context.Context, params LoginParams) (LoginResult, error) {

	log := web.LoggerFrom(ctx).With("op", "auth.Login", "identifier", params.Identifier)
	var user *userType.User
	var err error

	if isEmail(params.Identifier) {
		user, err = s.userRepo.GetUserByEmail(ctx, params.Identifier)
	} else {
		user, err = s.userRepo.GetUserByPhone(ctx, params.Identifier)
	}

	if err != nil {
		return LoginResult{}, err
	}
	if user == nil {
		log.Warn("user not found", "identifier", params.Identifier, "error", err)
		return LoginResult{}, ErrInvalidCredentials
	}
	err = s.hasher.Compare(*user.PasswordHash, params.Password)
	if err != nil {
		log.Warn("invalid password", "identifier", params.Identifier, "error", err)
		return LoginResult{}, ErrInvalidCredentials
	}

	estatesGroup, err := s.membershipRepo.ListUserEstateMemberships(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
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
		User:                   newUserResponse(*user),
		Estates:                estatesGroup,
	}

	return response, nil
}

func isEmail(identifier string) bool {
	return strings.Contains(identifier, "@")
}
