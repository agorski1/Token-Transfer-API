package wallet

import (
	"context"
	"log"
	"testing"

	"github.com/agorski1/token-transfer-api/db"
	"github.com/agorski1/token-transfer-api/internal/graph/model"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) (*gorm.DB, *WalletService) {
	t.Helper()

	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("Warning: .env file not found, using env variables if set")
	}

	dbConn, err := db.Connect()
	require.NoError(t, err)

	t.Cleanup(func() {
		resetTestWallets(t, dbConn)
	})

	repo := NewWalletRepository(dbConn)
	service := NewWalletService(repo)

	return dbConn, service
}

func TestTransfer_Success(t *testing.T) {
	tx, service := newTestService(t)
	ctx := context.Background()
	var fromWallet, toWallet model.Wallet

	// Given
	fromAddr := "0x0000000000000000000000000000000000000000"
	toAddr := "0x1000000000000000000000000000000000000000"
	amount := 500

	// When
	result, err := service.Transfer(ctx, fromAddr, toAddr, int32(amount))
	require.NoError(t, tx.Where("address = ?", fromAddr).First(&fromWallet).Error)
	require.NoError(t, tx.Where("address = ?", toAddr).First(&toWallet).Error)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(999500), result.Balance)
	assert.Equal(t, int32(999500), fromWallet.Balance)
	assert.Equal(t, int32(510), toWallet.Balance)
}

func TestTransfer_CreatesToWalletIfNotExists(t *testing.T) {
	tx, service := newTestService(t)
	ctx := context.Background()
	var toWallet model.Wallet

	// Given
	fromAddr := "0x0000000000000000000000000000000000000000"
	toAddr := "0x4000000000000000000000000000000000000000"
	amount := 500

	// When
	result, err := service.Transfer(ctx, fromAddr, toAddr, int32(amount))
	err2 := tx.Where("address = ?", toAddr).First(&toWallet).Error

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, err2)
	assert.Equal(t, int32(500), toWallet.Balance)
}

func TestTransfer_FailsWhenAmountIsZeroOrNegative(t *testing.T) {
	_, service := newTestService(t)
	ctx := context.Background()

	// Given
	fromAddr := "0x0000000000000000000000000000000000000000"
	toAddr := "0x1000000000000000000000000000000000000000"

	invalidAmounts := []int32{0, -10}

	for _, amount := range invalidAmounts {
		// When
		result, err := service.Transfer(ctx, fromAddr, toAddr, amount)

		// Then
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "amount must be greater than 0")
	}
}

func TestTransfer_FailsWhenSourceWalletNotFound(t *testing.T) {
	_, service := newTestService(t)
	ctx := context.Background()

	// Given
	fromAddr := "0xnonexistent_from"
	toAddr := "0x1000000000000000000000000000000000000000"
	amount := int32(10)

	// When
	result, err := service.Transfer(ctx, fromAddr, toAddr, amount)

	// Then
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "source wallet does not exist")
}

func TestTransfer_FailsWhenInsufficientBalance(t *testing.T) {
	_, service := newTestService(t)
	ctx := context.Background()

	// Given
	fromAddr := "0x2000000000000000000000000000000000000000"
	toAddr := "0x1000000000000000000000000000000000000000"
	amount := int32(20)

	// When
	result, err := service.Transfer(ctx, fromAddr, toAddr, amount)

	// Then
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insufficient balance")
}
