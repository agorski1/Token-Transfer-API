package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/agorski1/token-transfer-api/db"
	"github.com/agorski1/token-transfer-api/internal/graph/model"
	"github.com/agorski1/token-transfer-api/internal/wallet"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestServer(t *testing.T) (*gorm.DB, *httptest.Server) {
	t.Helper()

	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("Warning: .env file not found, using env variables if set")
	}

	dbConn, err := db.Connect()
	require.NoError(t, err)

	resetTestWallets(t, dbConn)

	t.Cleanup(func() {
		resetTestWallets(t, dbConn)
	})

	repo := wallet.NewWalletRepository(dbConn)
	service := wallet.NewWalletService(repo)
	resolver := Resolver{WalletService: service}

	// Utworzenie serwera gqlgen
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{
		Resolvers: &resolver,
	}))

	// Start httptest.Server
	server := httptest.NewServer(srv)

	// Cleanup serwera po zakończeniu testu
	t.Cleanup(func() {
		server.Close()
	})

	return dbConn, server
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

func buildTransferQuery(from, to string, amount int) string {
	return fmt.Sprintf(`
		mutation {
			transfer(srcAddress: "%s", dstAddress: "%s", amount: %d) {
				balance
			}
		}
	`, from, to, amount)
}

func buildCanAffordQuery(address string, amount int) string {
	return fmt.Sprintf(`
		query {
			canAfford(address: "%s", amount: %d)
		}
	`, address, amount)
}

func doGraphQLRequest(t *testing.T, url string, query string) (*http.Response, error) {
	payload := map[string]any{"query": query}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	return resp, err
}

type TransferResponse struct {
	Data struct {
		Transfer struct {
			Balance int `json:"balance"`
		} `json:"transfer"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type CanAffordResponse struct {
	Data struct {
		CanAfford bool `json:"canAfford"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func TestGraphMutationTransfer_Success(t *testing.T) {
	// Given
	db, server := newTestServer(t)
	defer server.Close()
	srcAddress := "0x0000000000000000000000000000000000000000"
	dstAddress := "0x1000000000000000000000000000000000000000"
	amount := 1000
	query := buildTransferQuery(srcAddress, dstAddress, amount)
	var result TransferResponse
	var srcWallet, dstWallet model.Wallet

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err, "Failed to perform GraphQL request")
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode response")

	err = db.Find(&srcWallet, "address = ?", srcAddress).Error
	require.NoError(t, err, "Failed to find srcWallet")

	err = db.Find(&dstWallet, "address = ?", dstAddress).Error
	require.NoError(t, err, "Failed to find dstWallet")

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP status OK")
	assert.Empty(t, result.Errors, "Expected no GraphQL errors")
	assert.Equal(t, 999000, result.Data.Transfer.Balance, "Expected correct transfer balance")
	assert.Equal(t, int32(999000), srcWallet.Balance, "Expected correct srcWallet balance")
	assert.Equal(t, int32(1010), dstWallet.Balance, "Expected correct dstWallet balance")
}

func TestGraphMutationTransfer_NegativeAmount(t *testing.T) {
	// Given
	db, server := newTestServer(t)
	defer server.Close()
	srcAddress := "0x0000000000000000000000000000000000000000"
	dstAddress := "0xBob"
	amount := -1000
	query := buildTransferQuery(srcAddress, dstAddress, amount)
	var result TransferResponse
	var srcWallet, dstWallet model.Wallet

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err, "Failed to perform GraphQL request")
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode response")

	err = db.Find(&srcWallet, "address = ?", srcAddress).Error
	require.NoError(t, err, "Failed to find srcWallet")

	err = db.Find(&dstWallet, "address = ?", dstAddress).Error
	require.NoError(t, err, "Failed to find dstWallet")

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP status OK")
	assert.NotEmpty(t, result.Errors, "Expected GraphQL errors")
	assert.Contains(t, result.Errors[0].Message, "amount must be greater than 0", "Expected error message about negative amount")
	assert.Equal(t, int32(1000000), srcWallet.Balance, "Expected unchanged srcWallet balance")
	assert.Equal(t, int32(0), dstWallet.Balance, "Expected unchanged dstWallet balance")
}

func TestGraphMutationTransfer_InsufficientBalance(t *testing.T) {
	// Given
	db, server := newTestServer(t)
	defer server.Close()
	srcAddress := "0x0000000000000000000000000000000000000000"
	dstAddress := "0xBob"
	amount := 2000000
	query := buildTransferQuery(srcAddress, dstAddress, amount)
	var result TransferResponse
	var srcWallet, dstWallet model.Wallet

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err, "Failed to perform GraphQL request")
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode response")

	err = db.Find(&srcWallet, "address = ?", srcAddress).Error
	require.NoError(t, err, "Failed to find srcWallet")

	err = db.Find(&dstWallet, "address = ?", dstAddress).Error
	require.NoError(t, err, "Failed to find dstWallet")

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP status OK")
	assert.NotEmpty(t, result.Errors, "Expected GraphQL errors")
	assert.Contains(t, result.Errors[0].Message, "insufficient balance", "Expected error message about insufficient balance")
	assert.Equal(t, int32(1000000), srcWallet.Balance, "Expected unchanged srcWallet balance")
	assert.Equal(t, int32(0), dstWallet.Balance, "Expected unchanged dstWallet balance")
}

func TestGraphMutationTransfer_srcWalletNotFound(t *testing.T) {
	// Given
	_, server := newTestServer(t)
	defer server.Close()
	srcAddress := "0x5000000000000000000000000000000000000000"
	dstAddress := "0x1000000000000000000000000000000000000000"
	amount := 100
	query := buildTransferQuery(srcAddress, dstAddress, amount)
	var result TransferResponse

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err, "Failed to perform GraphQL request")
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode response")

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP status OK")
	assert.NotEmpty(t, result.Errors, "Expected GraphQL errors")
	assert.Contains(t, result.Errors[0].Message, "source wallet does not exist", "Expected error message about missing from wallet")
}

func TestGraphMutationTransfer_MissingField(t *testing.T) {
	// Given
	_, server := newTestServer(t)
	defer server.Close()
	query := `
		mutation {
			transfer(srcAddress: "0xAlice", dstAddress: "0xBob") {
				balance
			}
		}
	`
	var result TransferResponse

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err, "Failed to perform GraphQL request")
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode response")

	// Then
	assert.Equal(t, 422, resp.StatusCode, "Expected HTTP status 422 for invalid query")
	assert.NotEmpty(t, result.Errors, "Expected GraphQL errors")
	assert.Contains(t, result.Errors[0].Message, "amount", "Expected error message about missing amount field")
}

func TestGraphMutationTransfer_ResponseStructure(t *testing.T) {
	// Given
	_, server := newTestServer(t)
	defer server.Close()
	srcAddress := "0x0000000000000000000000000000000000000000"
	dstAddress := "0x1000000000000000000000000000000000000000"
	amount := 100
	query := buildTransferQuery(srcAddress, dstAddress, amount)
	var response map[string]interface{}

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err, "Failed to perform GraphQL request")
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP status OK")
	assert.Contains(t, response, "data", "Response should contain 'data'")
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Data should be a map")
	assert.Contains(t, data, "transfer", "Data should contain 'transfer'")
	transfer, ok := data["transfer"].(map[string]interface{})
	assert.True(t, ok, "Transfer should be a map")
	assert.Contains(t, transfer, "balance", "Transfer should contain 'balance'")
}

func TestGraphQueryCanAfford_Success(t *testing.T) {
	// Given
	_, server := newTestServer(t)
	defer server.Close()
	addr := "0x0000000000000000000000000000000000000000"
	amount := 1000
	query := buildCanAffordQuery(addr, amount)
	var result CanAffordResponse

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err)
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, result.Errors)
	assert.True(t, result.Data.CanAfford, "Expected CanAfford to be true")
}

func TestGraphQueryCanAfford_InsufficientFunds(t *testing.T) {
	// Given
	_, server := newTestServer(t)
	defer server.Close()
	addr := "0x2000000000000000000000000000000000000000"
	amount := 5
	query := buildCanAffordQuery(addr, amount)
	var result CanAffordResponse

	// When
	resp, err := doGraphQLRequest(t, server.URL, query)
	require.NoError(t, err)
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, result.Errors)
	assert.False(t, result.Data.CanAfford, "Expected CanAfford to be false")
}
