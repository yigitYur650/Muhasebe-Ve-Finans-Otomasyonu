package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"deftersystem/backend/internal/domain"
)

func TestDecimalPrecision_NoFloatLoss(t *testing.T) {
	// Standard float 0.1 + 0.2 equals 0.30000000000000004 in IEEE 754.
	// decimal.Decimal MUST equal 0.3 exactly.
	d1 := decimal.NewFromFloat(0.1)
	d2 := decimal.NewFromFloat(0.2)
	expected := decimal.NewFromFloat(0.3)

	result := d1.Add(d2)

	assert.True(t, result.Equal(expected), "0.1 + 0.2 must equal 0.3 exactly without floating point representation loss")
	assert.Equal(t, "0.3", result.String())
}

func TestPeriodClosingBalanceCalculation(t *testing.T) {
	startingBalance := decimal.NewFromFloat(15000.50)

	transactions := []*domain.Transaction{
		{
			ID:        uuid.New(),
			Direction: domain.DirectionIn,
			Channel:   domain.ChannelEft,
			Amount:    decimal.NewFromFloat(5450.75),
		},
		{
			ID:        uuid.New(),
			Direction: domain.DirectionOut,
			Channel:   domain.ChannelKira,
			Amount:    decimal.NewFromFloat(3200.00),
		},
		{
			ID:        uuid.New(),
			Direction: domain.DirectionOut,
			Channel:   domain.ChannelMaasBanka,
			Amount:    decimal.NewFromFloat(4500.25),
		},
		{
			ID:        uuid.New(),
			Direction: domain.DirectionIn,
			Channel:   domain.ChannelPos,
			Amount:    decimal.NewFromFloat(1200.00),
		},
	}

	totalIn := decimal.Zero
	totalOut := decimal.Zero

	for _, tx := range transactions {
		assert.NoError(t, tx.Validate(), "Valid transaction should pass validation")
		if tx.Direction == domain.DirectionIn {
			totalIn = totalIn.Add(tx.Amount)
		} else if tx.Direction == domain.DirectionOut {
			totalOut = totalOut.Add(tx.Amount)
		}
	}

	// Starting: 15000.50, In: 6650.75, Out: 7700.25 => Closing: 13951.00
	expectedIn := decimal.NewFromFloat(6650.75)
	expectedOut := decimal.NewFromFloat(7700.25)
	expectedClosing := startingBalance.Add(totalIn).Sub(totalOut) // 15000.50 + 6650.75 - 7700.25 = 13951.00

	assert.True(t, totalIn.Equal(expectedIn), "Total IN should match expected")
	assert.True(t, totalOut.Equal(expectedOut), "Total OUT should match expected")
	assert.True(t, expectedClosing.Equal(decimal.NewFromFloat(13951.00)), "Closing balance calculation should be exact")
}

func TestTransactionValidation_NegativeOrZeroAmount(t *testing.T) {
	txZero := &domain.Transaction{
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelNakit,
		Amount:    decimal.Zero,
	}

	txNegative := &domain.Transaction{
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelNakit,
		Amount:    decimal.NewFromFloat(-100.50),
	}

	assert.ErrorIs(t, txZero.Validate(), domain.ErrInvalidAmount)
	assert.ErrorIs(t, txNegative.Validate(), domain.ErrInvalidAmount)
}

func TestTransactionValidation_InvalidDirectionAndChannel(t *testing.T) {
	txBadDirection := &domain.Transaction{
		Direction: domain.Direction("invalid_direction"),
		Channel:   domain.ChannelNakit,
		Amount:    decimal.NewFromFloat(100.00),
	}

	txBadChannel := &domain.Transaction{
		Direction: domain.DirectionIn,
		Channel:   domain.Channel("invalid_channel"),
		Amount:    decimal.NewFromFloat(100.00),
	}

	assert.ErrorIs(t, txBadDirection.Validate(), domain.ErrInvalidDirection)
	assert.ErrorIs(t, txBadChannel.Validate(), domain.ErrInvalidChannel)
}

func TestPeriod_IsLocked(t *testing.T) {
	now := time.Now()

	openPeriod := &domain.Period{
		Status: domain.PeriodStatusOpen,
	}

	lockedPeriod := &domain.Period{
		Status:   domain.PeriodStatusLocked,
		LockedAt: &now,
	}

	assert.False(t, openPeriod.IsLocked())
	assert.True(t, lockedPeriod.IsLocked())
}

func TestRoleValidation(t *testing.T) {
	assert.True(t, domain.RoleAdmin.IsValid())
	assert.True(t, domain.RoleMuhasebeci.IsValid())
	assert.True(t, domain.RoleStandart.IsValid())
	assert.False(t, domain.Role("super_admin").IsValid())
}
