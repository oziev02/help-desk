package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/oziev02/help-desk/internal/config"
	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/repository"
)

type AuthService struct {
	users repository.UserStore
	cfg   config.Config
}

func NewAuthService(users repository.UserStore, cfg config.Config) *AuthService {
	return &AuthService{users: users, cfg: cfg}
}

type TokenPair struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type claims struct {
	UserID string        `json:"user_id"`
	Roles  []domain.Role `json:"roles"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, domain.User, error) {
	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return TokenPair{}, domain.User{}, domain.ErrInvalidCredentials
		}
		return TokenPair{}, domain.User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, domain.User{}, domain.ErrInvalidCredentials
	}
	token, err := s.issueToken(user)
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	return token, user, nil
}

func (s *AuthService) Register(ctx context.Context, email, password, fullName string) (TokenPair, domain.User, error) {
	email = strings.TrimSpace(email)
	fullName = strings.TrimSpace(fullName)
	if email == "" || password == "" || fullName == "" {
		return TokenPair{}, domain.User{}, domain.ErrInvalidInput
	}
	if !strings.Contains(email, "@") || len(password) < 8 {
		return TokenPair{}, domain.User{}, domain.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	user, err := s.users.CreateUser(ctx, email, string(hash), fullName, []domain.Role{domain.RoleUser})
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	token, err := s.issueToken(user)
	if err != nil {
		return TokenPair{}, domain.User{}, err
	}
	return token, user, nil
}

func (s *AuthService) issueToken(user domain.User) (TokenPair, error) {
	now := time.Now()
	exp := now.Add(s.cfg.JWTExpiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: user.ID,
		Roles:  user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign token: %w", err)
	}
	return TokenPair{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.cfg.JWTExpiry.Seconds()),
	}, nil
}

func ParseToken(secret, tokenStr string) (domain.Actor, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	return domain.Actor{UserID: c.UserID, Roles: c.Roles}, nil
}
