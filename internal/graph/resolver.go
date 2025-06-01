package graph

import (
	"context"
	"fmt"

	model1 "github.com/agorski1/token-transfer-api/internal/graph/model"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

func (r *mutationResolver) Transfer(ctx context.Context, fromAddress string, toAddress string, amount int32) (*model1.TransferResult, error) {
	panic(fmt.Errorf("not implemented: Transfer - transfer"))
}

// CanAfford is the resolver for the canAfford field.
func (r *queryResolver) CanAfford(ctx context.Context, address string, amount int32) (bool, error) {
	panic(fmt.Errorf("not implemented: CanAfford - canAfford"))
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
