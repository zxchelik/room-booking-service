package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

const (
	DummyAdminID    = "11111111-1111-1111-1111-111111111111"
	DummyUserID     = "22222222-2222-2222-2222-222222222222"
	dummyAdminEmail = "admin@example.com"
	dummyUserEmail  = "user@example.com"
)

type AuthService struct {
	users                  *repository.UserRepository
	db                     repository.DBTX
	jwtSecret              []byte
	jwtTTL                 time.Duration
	allowAdminRegistration bool
	now                    func() time.Time
}

func NewAuthService(db repository.DBTX, users *repository.UserRepository, jwtSecret string, jwtTTL time.Duration, allowAdminRegistration bool) *AuthService {
	return &AuthService{
		db:                     db,
		users:                  users,
		jwtSecret:              []byte(jwtSecret),
		jwtTTL:                 jwtTTL,
		allowAdminRegistration: allowAdminRegistration,
		now:                    time.Now,
	}
}

func (s *AuthService) DummyLogin(ctx context.Context, role model.Role) (string, error) {
	var user model.User
	now := s.now().UTC()
	switch role {
	case model.RoleAdmin:
		user = model.User{ID: DummyAdminID, Email: dummyAdminEmail, Role: model.RoleAdmin, CreatedAt: &now}
	case model.RoleUser:
		user = model.User{ID: DummyUserID, Email: dummyUserEmail, Role: model.RoleUser, CreatedAt: &now}
	default:
		return "", InvalidRequest("role must be admin or user")
	}

	if err := s.users.Upsert(ctx, s.db, user); err != nil {
		return "", err
	}

	return s.issueToken(user)
}

func (s *AuthService) Register(ctx context.Context, email, password string, role model.Role) (model.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)

	if !isValidEmail(email) {
		return model.User{}, InvalidRequest("email must be valid")
	}
	if password == "" {
		return model.User{}, InvalidRequest("password is required")
	}
	if role != model.RoleAdmin && role != model.RoleUser {
		return model.User{}, InvalidRequest("role must be admin or user")
	}
	if role == model.RoleAdmin && !s.allowAdminRegistration {
		return model.User{}, InvalidRequest("admin registration is disabled outside development and test environments")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}

	now := s.now().UTC()
	hashValue := string(hashedPassword)
	user := model.User{
		ID:           uuid.NewString(),
		Email:        email,
		Role:         role,
		PasswordHash: &hashValue,
		CreatedAt:    &now,
	}

	created, err := s.users.Create(ctx, s.db, user)
	if err != nil {
		if repository.IsUniqueViolation(err) {
			return model.User{}, InvalidRequest("email is already registered")
		}
		return model.User{}, err
	}
	created.PasswordHash = nil
	return created, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)

	if !isValidEmail(email) {
		return "", InvalidRequest("email must be valid")
	}
	if password == "" {
		return "", InvalidRequest("password is required")
	}

	user, err := s.users.GetByEmail(ctx, s.db, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", Unauthorized("invalid email or password")
		}
		return "", err
	}
	if user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)) != nil {
		return "", Unauthorized("invalid email or password")
	}

	return s.issueToken(user)
}

func (s *AuthService) issueToken(user model.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    string(user.Role),
		"exp":     s.now().Add(s.jwtTTL).Unix(),
		"iat":     s.now().Unix(),
	})

	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	return signed, nil
}

func (s *AuthService) JWTSecret() []byte {
	return s.jwtSecret
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
