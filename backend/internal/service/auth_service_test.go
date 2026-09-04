package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/repository"
	"deftersystem/backend/internal/service"
)

func TestAuthService_SetAndGetSecurityQuestion(t *testing.T) {
	mockRepo := repository.NewMockUserSecurityRepository()
	authSvc := service.NewAuthService(mockRepo)

	userID := uuid.New()
	email := "test@oncuotogaz.com"
	question := "İlk evcil hayvanınızın adı nedir?"
	answer := "Karabaş"

	ctx := context.Background()

	// Set question
	err := authSvc.SetSecurityQuestion(ctx, userID, email, question, answer)
	assert.NoError(t, err)

	// Fetch question
	retrievedQ, err := authSvc.GetSecurityQuestion(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, question, retrievedQ)
}

func TestAuthService_ResetPasswordWithSecurityAnswer(t *testing.T) {
	mockRepo := repository.NewMockUserSecurityRepository()
	authSvc := service.NewAuthService(mockRepo)

	userID := uuid.New()
	email := "user@oncuotogaz.com"
	question := "En sevdiğiniz öğretmen kimdir?"
	answer := "Ahmet Hoca"

	ctx := context.Background()
	_ = authSvc.SetSecurityQuestion(ctx, userID, email, question, answer)

	// Correct answer case (case-insensitive test)
	err := authSvc.ResetPasswordWithSecurityAnswer(ctx, email, "ahmet hoca", "newpassword123")
	assert.NoError(t, err)

	// Wrong answer case
	errWrong := authSvc.ResetPasswordWithSecurityAnswer(ctx, email, "yanlis cevap", "newpassword123")
	assert.ErrorIs(t, errWrong, domain.ErrInvalidSecurityAnswer)
}
