package wallet

import (
	"context"

	"github.com/agorski1/token-transfer-api/internal/graph/model"
)

type mutationResolver struct {
	*WalletService
}

type queryResolver struct {
	*WalletService
}

func NewMutationResolver(s *WalletService) *mutationResolver {
	return &mutationResolver{WalletService: s}
}

func NewQueryResolver(s *WalletService) *queryResolver {
	return &queryResolver{WalletService: s}
}

func (r *mutationResolver) Transfer(ctx context.Context, srcAddress string, dstAddress string, amount int32) (*model.TransferResult, error) {
	return r.WalletService.Transfer(ctx, srcAddress, dstAddress, amount)
}

func (r *queryResolver) CanAfford(ctx context.Context, address string, amount int32) (bool, error) {
	return r.WalletService.CanAfford(ctx, address, amount)
}
