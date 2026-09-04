package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"deftersystem/backend/internal/domain"
)

type PostgresUserSecurityRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserSecurityRepository initializes user security repository using PostgreSQL.
func NewPostgresUserSecurityRepository(pool *pgxpool.Pool) domain.UserSecurityRepository {
	return &PostgresUserSecurityRepository{pool: pool}
}

func (r *PostgresUserSecurityRepository) Save(ctx context.Context, sec *domain.UserSecurity) error {
	query := `
		INSERT INTO public.user_security (id, user_id, email, security_question, security_answer_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			security_question = EXCLUDED.security_question,
			security_answer_hash = EXCLUDED.security_answer_hash,
			updated_at = EXCLUDED.updated_at
	`
	sec.Email = strings.ToLower(strings.TrimSpace(sec.Email))
	if sec.ID == uuid.Nil {
		sec.ID = uuid.New()
	}
	now := time.Now()
	if sec.CreatedAt.IsZero() {
		sec.CreatedAt = now
	}
	sec.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query, sec.ID, sec.UserID, sec.Email, sec.SecurityQuestion, sec.SecurityAnswerHash, sec.CreatedAt, sec.UpdatedAt)
	return MapSQLError(err)
}

func (r *PostgresUserSecurityRepository) GetByEmail(ctx context.Context, email string) (*domain.UserSecurity, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	query := `SELECT id, user_id, email, security_question, security_answer_hash, created_at, updated_at FROM public.user_security WHERE email = $1`
	var sec domain.UserSecurity
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&sec.ID, &sec.UserID, &sec.Email, &sec.SecurityQuestion, &sec.SecurityAnswerHash, &sec.CreatedAt, &sec.UpdatedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &sec, nil
}

func (r *PostgresUserSecurityRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserSecurity, error) {
	query := `SELECT id, user_id, email, security_question, security_answer_hash, created_at, updated_at FROM public.user_security WHERE user_id = $1`
	var sec domain.UserSecurity
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&sec.ID, &sec.UserID, &sec.Email, &sec.SecurityQuestion, &sec.SecurityAnswerHash, &sec.CreatedAt, &sec.UpdatedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &sec, nil
}

// MockUserSecurityRepository for unit tests
type MockUserSecurityRepository struct {
	mu    sync.RWMutex
	store map[string]*domain.UserSecurity
}

func NewMockUserSecurityRepository() *MockUserSecurityRepository {
	return &MockUserSecurityRepository{store: make(map[string]*domain.UserSecurity)}
}

func (m *MockUserSecurityRepository) Save(ctx context.Context, sec *domain.UserSecurity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	email := strings.ToLower(strings.TrimSpace(sec.Email))
	sec.Email = email
	m.store[email] = sec
	m.store[sec.UserID.String()] = sec
	return nil
}

func (m *MockUserSecurityRepository) GetByEmail(ctx context.Context, email string) (*domain.UserSecurity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	email = strings.ToLower(strings.TrimSpace(email))
	if sec, ok := m.store[email]; ok {
		return sec, nil
	}
	return nil, domain.ErrSecurityQuestionNotFound
}

func (m *MockUserSecurityRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserSecurity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sec, ok := m.store[userID.String()]; ok {
		return sec, nil
	}
	return nil, domain.ErrSecurityQuestionNotFound
}
