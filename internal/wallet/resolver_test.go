package wallet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMutationResolver_Transfer(t *testing.T) {
	_, service := newTestService(t)
	resolver := NewMutationResolver(service)
	ctx := context.Background()

	// Given
	srcAddress := "0x0000000000000000000000000000000000000000"
	dstAddress := "0x1000000000000000000000000000000000000000"
	amount := int32(10)

	// When
	result, err := resolver.Transfer(ctx, srcAddress, dstAddress, amount)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.Balance, int32(0))
}

func TestQueryResolver_CanAfford(t *testing.T) {
	_, service := newTestService(t)
	resolver := NewQueryResolver(service)
	ctx := context.Background()

	// Given
	addr := "0x0000000000000000000000000000000000000000"
	amount := 100

	// When
	canAfford, err := resolver.CanAfford(ctx, addr, int32(amount))
	require.NoError(t, err)

	// Then
	assert.True(t, canAfford)
}
