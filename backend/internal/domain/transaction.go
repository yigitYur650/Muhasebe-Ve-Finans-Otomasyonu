package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Direction string

const (
	DirectionIn  Direction = "in"
	DirectionOut Direction = "out"
)

func (d Direction) IsValid() bool {
	switch d {
	case DirectionIn, DirectionOut:
		return true
	default:
		return false
	}
}

type Channel string

const (
	ChannelEft        Channel = "eft"
	ChannelPos        Channel = "pos"
	ChannelNakit      Channel = "nakit"
	ChannelKredi      Channel = "kredi"
	ChannelKira       Channel = "kira"
	ChannelMaasBanka  Channel = "maas_banka"
	ChannelMaasElden  Channel = "maas_elden"
	ChannelKrediKarti Channel = "kredi_karti"
	ChannelKartus     Channel = "kartus"
	ChannelYemek      Channel = "yemek"
	ChannelYakit      Channel = "yakit"
	ChannelDiger      Channel = "diger"
)

func (c Channel) IsValid() bool {
	switch c {
	case ChannelEft, ChannelPos, ChannelNakit, ChannelKredi,
		ChannelKira, ChannelMaasBanka, ChannelMaasElden, ChannelKrediKarti,
		ChannelKartus, ChannelYemek, ChannelYakit, ChannelDiger:
		return true
	default:
		return false
	}
}

// Transaction represents an append-only entry in the ledger.
type Transaction struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	PeriodID    uuid.UUID       `json:"period_id"`
	Direction   Direction       `json:"direction"`
	Channel     Channel         `json:"channel"`
	Amount      decimal.Decimal `json:"amount"`
	Description *string         `json:"description,omitempty"`
	CreatedBy   uuid.UUID       `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	ReversedBy  *uuid.UUID      `json:"reversed_by,omitempty"`
}

// Validate checks domain level integrity rules for a transaction.
func (t *Transaction) Validate() error {
	if !t.Direction.IsValid() {
		return ErrInvalidDirection
	}
	if !t.Channel.IsValid() {
		return ErrInvalidChannel
	}
	if t.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}
	return nil
}
