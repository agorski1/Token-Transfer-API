package wallet

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/agorski1/token-transfer-api/db"
	"github.com/agorski1/token-transfer-api/internal/graph/model"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testCase struct {
	name   string
	from   string
	to     string
	amount int32
}

type result struct {
	success bool
	balance int32
	err     error
}

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

func TestMutationTransfer_ParallelTransactions(t *testing.T) {
	tx, service := newTestService(t)
	ctx := context.Background()

	// Given
	tests := []testCase{
		{"plus1", "0x2000000000000000000000000000000000000000", "0x1000000000000000000000000000000000000000", 1},
		{"minus4", "0x1000000000000000000000000000000000000000", "0x3000000000000000000000000000000000000000", 4},
		{"minus7", "0x1000000000000000000000000000000000000000", "0x4000000000000000000000000000000000000000", 7},
	}

	expectedBalances := map[[3]bool]int32{
		{true, true, false}: 7,
		{true, false, true}: 4,
		{true, true, true}:  0,
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]result, len(tests))

	for i, tc := range tests {
		wg.Add(1)
		go func(i int, tc testCase) {
			defer wg.Done()
			canAfford, err := service.CanAfford(ctx, tc.from, tc.amount) // Ensure precondition (can afford)
			require.NoError(t, err, "CanAfford failed for test case %q", tc.name)
			require.True(t, canAfford, "Sender cannot afford amount in %q", tc.name)

			<-start // wait for all goroutines to start

			// When
			transfer, err := service.Transfer(ctx, tc.from, tc.to, tc.amount)
			if err != nil {
				t.Logf("Transfer failed for %q: %v", tc.name, err)
				results[i] = result{false, 0, err}
			} else {
				results[i] = result{true, transfer.Balance, nil}
			}
		}(i, tc)
	}

	time.Sleep(200 * time.Millisecond)
	close(start)
	wg.Wait()

	// Then
	var wallet model.Wallet
	require.NoError(t, tx.Find(&wallet, "address = ?", "0x1000000000000000000000000000000000000000").Error)

	resultKey := [3]bool{results[0].success, results[1].success, results[2].success}

	expectedBalance, ok := expectedBalances[resultKey]
	require.True(t, ok, "Unexpected transfer result combination: %+v", resultKey)
	assert.Equal(t, expectedBalance, int32(wallet.Balance), "Final balance mismatch")
}
