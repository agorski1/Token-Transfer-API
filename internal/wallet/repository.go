package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/agorski1/token-transfer-api/internal/graph/model"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrWalletNotFound = errors.New("wallet not found")
var ErrRetryableTransaction = errors.New("retryable transaction error")

type WalletRepository interface {
	WithTransaction(ctx context.Context, fn func(r WalletRepository) error) error
	CreateWallet(ctx context.Context, address string) (*model.Wallet, error)
	SaveWallet(ctx context.Context, wallet *model.Wallet) error
	GetWalletByAddress(ctx context.Context, address string) (*model.Wallet, error)
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{DB: db}
}

type walletRepository struct {
	DB *gorm.DB
}

func (r *walletRepository) CreateWallet(ctx context.Context, address string) (*model.Wallet, error) {
	wallet := &model.Wallet{Address: address, Balance: 0}
	if err := r.DB.Create(wallet).Error; err != nil {
		return nil, err
	}
	return wallet, nil
}

func (r *walletRepository) SaveWallet(ctx context.Context, wallet *model.Wallet) error {
	return r.DB.Save(wallet).Error
}

func (r *walletRepository) GetWalletByAddress(ctx context.Context, address string) (*model.Wallet, error) {
	var wallet model.Wallet
	err := r.DB.Where("address = ?", address).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) WithTransaction(ctx context.Context, fn func(repo WalletRepository) error) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE").Error; err != nil {
			return fmt.Errorf("failed to set isolation level: %v", err)
		}
		txRepo := &walletRepository{DB: tx}
		err := fn(txRepo)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "40001" || pgErr.Code == "40P01" {
				return ErrRetryableTransaction
			}
		}

		return err
	})
}
