package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/user"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/auth"
)

// Sentinel errors for user service.
var (
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong      = errors.New("password must be at most 128 characters")
	ErrPasswordInvalidChars = errors.New("password contains invalid characters")
	ErrPasswordNonASCII     = errors.New("password contains non-ASCII characters")
	ErrUsernameExists       = errors.New("username already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrIncorrectPassword    = errors.New("incorrect password")
)

const (
	minPasswordLen = 8
	maxPasswordLen = 128
	maxASCIIChar   = 126
	tokenExpiry    = 24 * time.Hour
)

// RegisterResponse is the result of a successful user registration.
type RegisterResponse struct {
	UserID   string
	Username string
	Token    string
}

// LoginResponse is the result of a successful login.
type LoginResponse struct {
	UserID   string
	Username string
	Token    string
}

// ValidatePassword checks password complexity requirements.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLen {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLen {
		return ErrPasswordTooLong
	}
	for _, c := range password {
		if c <= 32 || c == 127 {
			return ErrPasswordInvalidChars
		}
		if c > maxASCIIChar {
			return ErrPasswordNonASCII
		}
	}
	return nil
}

// Service manages user-related operations.
type Service struct {
	client *ent.Client
	secret string
}

// NewService creates a new user Service.
func NewService(client *ent.Client, jwtSecret string) *Service {
	return &Service{client: client, secret: jwtSecret}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, username, password, displayName string) (*RegisterResponse, error) {
	if err := ValidatePassword(password); err != nil {
		return nil, fmt.Errorf("validate password: %w", err)
	}

	exists, err := s.client.User.Query().Where(user.Username(username)).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return nil, ErrUsernameExists
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.client.User.Create().
		SetID(uuid.New().String()).
		SetUsername(username).
		SetDisplayName(displayName).
		SetPasswordHash(hash).
		SetRole("user").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := auth.GenerateToken(s.secret, u.ID, "", tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &RegisterResponse{UserID: u.ID, Username: u.Username, Token: token}, nil
}

// Login authenticates a user and returns a JWT token.
func (s *Service) Login(ctx context.Context, username, password, deviceID string) (*LoginResponse, error) {
	u, err := s.client.User.Query().Where(user.Username(username)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	if !auth.CheckPassword(u.PasswordHash, password) {
		return nil, ErrIncorrectPassword
	}

	token, err := auth.GenerateToken(s.secret, u.ID, deviceID, tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResponse{UserID: u.ID, Username: u.Username, Token: token}, nil
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, userID string) (*ent.User, error) {
	u, err := s.client.User.Get(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}
