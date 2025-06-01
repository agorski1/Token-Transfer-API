package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/agorski1/token-transfer-api/internal/graph/model"
)

const maxRetries = 10

var (
	ErrInvalidAmount        = errors.New("amount must be greater than 0")
	ErrSourceWalletNotFound = errors.New("source wallet does not exist")
	ErrInsufficientBalance  = errors.New("insufficient balance")
)

type WalletService struct {
	repo WalletRepository
}

func NewWalletService(repo WalletRepository) *WalletService {
	return &WalletService{repo: repo}
}

func (s *WalletService) Transfer(ctx context.Context, srcAddress, dstAddress string, amount int32) (*model.TransferResult, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		result, err := s.tryTransfer(ctx, srcAddress, dstAddress, amount)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, ErrRetryableTransaction) {
			lastErr = err
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("transaction failed after %d retries: %w", maxRetries, lastErr)
}

func (s *WalletService) tryTransfer(ctx context.Context, srcAddress, dstAddress string, amount int32) (*model.TransferResult, error) {
	var responseSrcWallet *model.Wallet
	err := s.repo.WithTransaction(ctx, func(r WalletRepository) error {
		srcWallet, err := r.GetWalletByAddress(ctx, srcAddress)
		if err != nil {
			if errors.Is(err, ErrWalletNotFound) {
				return ErrSourceWalletNotFound
			}
			return err
		}

		if !s.canAfford(srcWallet, amount) {
			return ErrInsufficientBalance
		}

		dstWallet, err := r.GetWalletByAddress(ctx, dstAddress)
		if err != nil && !errors.Is(err, ErrWalletNotFound) {
			return err
		}
		if errors.Is(err, ErrWalletNotFound) {
			dstWallet, err = r.CreateWallet(ctx, dstAddress)
			if err != nil {
				return err
			}
		}

		if err := s.transferAmountFromSrcWalletToDstWallet(srcWallet, dstWallet, amount); err != nil {
			return err
		}

		if err := r.SaveWallet(ctx, srcWallet); err != nil {
			return err
		}
		if err := r.SaveWallet(ctx, dstWallet); err != nil {
			return err
		}

		responseSrcWallet = srcWallet
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &model.TransferResult{Balance: int32(responseSrcWallet.Balance)}, nil
}

func (s *WalletService) transferAmountFromSrcWalletToDstWallet(srcWallet, dstWallet *model.Wallet, transferAmount int32) error {
	srcWallet.Balance -= transferAmount
	dstWallet.Balance += transferAmount

	return nil
}

func (s *WalletService) canAfford(srcWallet *model.Wallet, amount int32) bool {
	return srcWallet.Balance >= amount
}
