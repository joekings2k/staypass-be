package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joekings2k/staypass-be/internal/platform/config"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/joekings2k/staypass-be/internal/platform/hash"
	"github.com/joekings2k/staypass-be/internal/user"
	"golang.org/x/term"
)

func main() {
	email := flag.String("email", "", "super admin email")
	phone := flag.String("phone", "", "super admin phone number")
	name := flag.String("name", "", "super admin name")

	flag.Parse()

	password := ReadPassword()

	if *email == "" && *phone == "" {
		slog.Error("email or phone is required")
	}

	if password == "" {
		slog.Error("password is required")
	}

	if *name == "" {
		slog.Error("name is required")
	}
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create postgres pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	hasher := hash.NewBcryptHasher()

	hashedpassword, err := hasher.Hash(password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		os.Exit(1)
	}
	queries := dbgen.New(pool)

	userRepo := user.NewPgUserRepository(queries)
	userRes, err := userRepo.GetUserByPhone(ctx, *phone)
	if err != nil {
		slog.Error("failed to get user by phone", "error", err)
	}
	if userRes != nil {
		slog.Info("user already exists", "phone", *phone)
		os.Exit(0)
	}
	userRes, err = userRepo.GetUserByEmail(ctx, *email)
	if err != nil {
		slog.Error("failed to get user by email", "error", err)
	}
	if userRes != nil {
		slog.Info("user already exists", "email", *email)
		os.Exit(0)
	}
	params := user.CreateUserParams{
		Phone:        *phone,
		Email:        email,
		FullName:     *name,
		PasswordHash: &hashedpassword,
	}
	userRes, err = userRepo.CreateUser(ctx, params)
	if err != nil {
		slog.Error("failed to create user", "error", err)
		os.Exit(1)
	}
	slog.Info("user created", "phone", *phone, "email", *email, "name", *name)
	os.Exit(0)
}

func ReadPassword() string {
	slog.Info("Enter password:")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))

	if err != nil {
		slog.Error("failed to read password", "error", err)
		os.Exit(1)
	}
	return string(password)
}
