package graph

import "github.com/agorski1/token-transfer-api/internal/wallet"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	WalletService *wallet.WalletService
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver {
	return wallet.NewMutationResolver(r.WalletService)
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver {
	return wallet.NewQueryResolver(r.WalletService)
}
