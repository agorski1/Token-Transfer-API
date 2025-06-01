package wallet

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/agorski1/token-transfer-api/db"
	"github.com/agorski1/token-transfer-api/internal/graph/model"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetWalletByAddress_Success(t *testing.T) {
	withTestTx(t, func(tx *gorm.DB) {
		repo := NewWalletRepository(tx)
		ctx := context.Background()

		// Given
		address := "0x1000000000000000000000000000000000000000"

		// When
		fetched, err := repo.GetWalletByAddress(ctx, address)

		// Then
		assert.NoError(t, err)
		assert.NotNil(t, fetched)
		assert.Equal(t, address, fetched.Address)
	})
}

func TestGetWalletByAddress_NotFound(t *testing.T) {
	withTestTx(t, func(tx *gorm.DB) {
		repo := NewWalletRepository(tx)
		ctx := context.Background()

		// Given
		address := "0x123"

		// When
		fetched, err := repo.GetWalletByAddress(ctx, address)

		// Then
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrWalletNotFound))
		assert.Nil(t, fetched)
	})
}

func TestCreateWallet_Success(t *testing.T) {
	withTestTx(t, func(tx *gorm.DB) {
		repo := NewWalletRepository(tx)
		ctx := context.Background()

		// Given
		address := "0x123"

		// When
		fetched, err := repo.CreateWallet(ctx, address)

		// Then
		assert.NoError(t, err)
		assert.NotNil(t, fetched)
	})
}

func TestCreateWallet_FailedWalletAlreadyExist(t *testing.T) {
	withTestTx(t, func(tx *gorm.DB) {
		repo := NewWalletRepository(tx)
		ctx := context.Background()

		// Given
		address := "0x1000000000000000000000000000000000000000"

		// When
		wallet, err := repo.CreateWallet(ctx, address)

		// Then
		assert.Error(t, err)
		assert.Nil(t, wallet)
		assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
	})
}

func TestSaveWallet_Success(t *testing.T) {
	withTestTx(t, func(tx *gorm.DB) {
		repo := NewWalletRepository(tx)
		ctx := context.Background()

		// Given
		address := "0x1000000000000000000000000000000000000000"
		balance := 200
		wallet := &model.Wallet{Address: address, Balance: int32(balance)}
		var saved model.Wallet

		// When
		err1 := repo.SaveWallet(ctx, wallet)
		err2 := tx.Where("address = ?", address).First(&saved).Error

		// Then
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, address, saved.Address)
		assert.Equal(t, int32(balance), saved.Balance)
	})
}

func TestWithTransaction_CommitsOnSuccess(t *testing.T) {
	tx, repo := newTestRepo(t)

	// Given
	address := "0x1000000000000000000000000000000000000000"
	var count int64

	// When
	err := repo.WithTransaction(context.Background(), func(txRepo WalletRepository) error {
		db := txRepo.(*walletRepository).DB
		return db.Where("address = ?", address).Delete(&model.Wallet{}).Error
	})

	// Then
	assert.NoError(t, err)
	err = tx.Model(&model.Wallet{}).Where("address = ?", address).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "wallet should be deleted within transaction")
}

func TestWithTransaction_RollbackOnError(t *testing.T) {
	tx, repo := newTestRepo(t)

	// Given
	address := "0xdef"
	err := tx.Create(&model.Wallet{Address: address}).Error
	require.NoError(t, err)

	// When
	err = repo.WithTransaction(context.Background(), func(txRepo WalletRepository) error {
		db := txRepo.(*walletRepository).DB
		delErr := db.Where("address = ?", address).Delete(&model.Wallet{}).Error
		require.NoError(t, delErr)
		return errors.New("force rollback")
	})

	// Then
	assert.Error(t, err)

	var count int64
	err = tx.Model(&model.Wallet{}).Where("address = ?", address).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "wallet should NOT be deleted due to rollback")
}

func newTestRepo(t *testing.T) (*gorm.DB, WalletRepository) {
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

	return dbConn, repo
}

func withTestTx(t *testing.T, testFunc func(tx *gorm.DB)) {
	t.Helper()

	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("Warning: .env file not found, using env variables if set")
	}
	dbConn, err := db.Connect()
	require.NoError(t, err)

	tx := dbConn.Begin()
	require.NoError(t, tx.Error)

	defer func() {
		_ = tx.Rollback()
	}()

	testFunc(tx)
}

func resetTestWallets(t *testing.T, db *gorm.DB) {
	t.Helper()

	err := db.Exec(`DELETE FROM wallets`).Error
	require.NoError(t, err)

	err = db.Exec(`
		INSERT INTO wallets (address, balance) VALUES 
		('0x0000000000000000000000000000000000000000', 1000000),
		('0x1000000000000000000000000000000000000000', 10),
		('0x2000000000000000000000000000000000000000', 1)
	`).Error
	require.NoError(t, err)
}
