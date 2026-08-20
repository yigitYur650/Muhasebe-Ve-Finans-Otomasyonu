package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// IdempotencyKey stores cached API response details to ensure request deduplication.
type IdempotencyKey struct {
	Key            string          `json:"key"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ResponseBody   json.RawMessage `json:"response_body,omitempty"`
	ResponseStatus int             `json:"response_status"`
	CreatedAt      time.Time       `json:"created_at"`
}
