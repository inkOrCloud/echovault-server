package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/user"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/auth"
)

type RegisterResponse struct {
	UserID   string
	Username string
	Token    string
}

type LoginResponse struct {
	UserID   string
	Username string
	Token    string
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return errors.New("password must be at most 128 characters")
	}
	for _, c := range password {
		if c <= 32 || c == 127 {
			return errors.New("password contains invalid characters")
		}
		if c > 126 {
			return errors.New("password contains non-ASCII characters")
		}
	}
	return nil
}

type Service struct {
	client *ent.Client
	secret string
}

func NewService(client *ent.Client, jwtSecret string) *Service {
	return &Service{client: client, secret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, username, password, displayName string) (*RegisterResponse, error) {
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	exists, err := s.client.User.Query().Where(user.Username(username)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	token, err := auth.GenerateToken(s.secret, u.ID, "", 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &RegisterResponse{UserID: u.ID, Username: u.Username, Token: token}, nil
}

func (s *Service) Login(ctx context.Context, username, password, deviceID string) (*LoginResponse, error) {
	u, err := s.client.User.Query().Where(user.Username(username)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if !auth.CheckPassword(u.PasswordHash, password) {
		return nil, errors.New("incorrect password")
	}

	token, err := auth.GenerateToken(s.secret, u.ID, deviceID, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{UserID: u.ID, Username: u.Username, Token: token}, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (*ent.User, error) {
	u, err := s.client.User.Get(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return u, nil
}
