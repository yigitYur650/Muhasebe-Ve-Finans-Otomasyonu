package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"deftersystem/backend/internal/domain"
)

type DefaultAuthService struct {
	securityRepo domain.UserSecurityRepository
}

func NewAuthService(securityRepo domain.UserSecurityRepository) domain.AuthService {
	return &DefaultAuthService{securityRepo: securityRepo}
}

func (s *DefaultAuthService) SetSecurityQuestion(ctx context.Context, userID uuid.UUID, email, question, answer string) error {
	cleanQuestion := strings.TrimSpace(question)
	cleanAnswer := strings.ToLower(strings.TrimSpace(answer))
	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	if cleanQuestion == "" || cleanAnswer == "" {
		return domain.ErrInvalidAmount // Generic invalid payload error
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cleanAnswer), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	sec := &domain.UserSecurity{
		ID:                 uuid.New(),
		UserID:             userID,
		Email:              cleanEmail,
		SecurityQuestion:   cleanQuestion,
		SecurityAnswerHash: string(hash),
		UpdatedAt:          time.Now(),
	}

	return s.securityRepo.Save(ctx, sec)
}

func (s *DefaultAuthService) GetSecurityQuestion(ctx context.Context, email string) (string, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	sec, err := s.securityRepo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return "", domain.ErrSecurityQuestionNotFound
	}
	return sec.SecurityQuestion, nil
}

func (s *DefaultAuthService) ResetPasswordWithSecurityAnswer(ctx context.Context, email, answer, newPassword string) error {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanAnswer := strings.ToLower(strings.TrimSpace(answer))
	cleanNewPassword := strings.TrimSpace(newPassword)

	if len(cleanNewPassword) < 6 {
		return domain.ErrInvalidAmount
	}

	sec, err := s.securityRepo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return domain.ErrSecurityQuestionNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(sec.SecurityAnswerHash), []byte(cleanAnswer)); err != nil {
		return domain.ErrInvalidSecurityAnswer
	}

	// Security answer matches! Password reset authorized.
	return nil
}
