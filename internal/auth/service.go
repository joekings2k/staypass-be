package auth

import (
	"context"
	"fmt"

	"github.com/joekings2k/staypass-be/internal/platform/config"
	"github.com/joekings2k/staypass-be/internal/platform/hash"
	"github.com/joekings2k/staypass-be/internal/user"
)

type AuthService struct {
	repo     AuthRepository
	userRepo user.UserRepository
	config   *config.Config
	hasher   hash.Hasher
}

var ()

func NewAuthService(repo AuthRepository, userRepo user.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:     repo,
		userRepo: userRepo,
		config:   cfg,
	}
}

func (s *AuthService) Login(email, password string) (Session, error) {
	user, err := s.userRepo.GetUserByEmail(context.Background(), email)
	fmt.Println("user", user)
	if err != nil {
		return Session{}, err
	}
	return Session{}, nil
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
	arg := user.CreateUserParams{
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
