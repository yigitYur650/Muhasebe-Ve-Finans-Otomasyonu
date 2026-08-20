package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSecurityQuestionNotFound = errors.New("güvenlik sorusu bulunamadı")
	ErrInvalidSecurityAnswer   = errors.New("güvenlik sorusu cevabı hatalı")
)

// UserSecurity represents a user's password recovery security question and hashed answer.
type UserSecurity struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	Email              string    `json:"email"`
	SecurityQuestion   string    `json:"security_question"`
	SecurityAnswerHash string    `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// UserSecurityRepository defines database operations for user security data.
type UserSecurityRepository interface {
	Save(ctx context.Context, sec *UserSecurity) error
	GetByEmail(ctx context.Context, email string) (*UserSecurity, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*UserSecurity, error)
}

// AuthService defines business logic for password management and security questions.
type AuthService interface {
	SetSecurityQuestion(ctx context.Context, userID uuid.UUID, email, question, answer string) error
	GetSecurityQuestion(ctx context.Context, email string) (string, error)
	ResetPasswordWithSecurityAnswer(ctx context.Context, email, answer, newPassword string) error
}
