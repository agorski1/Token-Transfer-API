# BTP Token Transfer API

This project is a GraphQL API written in Go for transferring BTP tokens between wallets, inspired by ERC20 token transfers. The API runs locally and connects to a PostgreSQL database managed by Docker Compose. It provides a single `transfer` mutation to move tokens between wallets, with an initial wallet (`0x0000000000000000000000000000000000000000`) holding 1,000,000 BTP tokens. The project includes tests for core functionality and race condition handling.

## Prerequisites

Before starting, ensure you have the following tools installed:
- **Go** (version 1.21 or later): [Download and install Go](https://golang.org/dl/)
- **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose**: [Install Docker Compose](https://docs.docker.com/compose/install/)
- **Git**: [Install Git](https://git-scm.com/downloads)

## Project Structure

```
proj2/
├── db/
│   ├── migrations/
│   │   ├── prod/
│   │   │   ├── 01_create_wallets.sql
│   │   │   └── 02_insert_wallets.sql
│   │   └── test/
│   │       ├── 01_create_wallets.sql
│   │       └── 02_insert_wallets.sql
│   └── db.go
├── internal/
│   ├── graph/
│   │   ├── model/
│   │   │   ├── transfer_result.go
│   │   │   └── wallet.go
│   │   ├── resolver.go
│   │   ├── resolver_test.go
│   │   ├── schema.graphqls
│   │   └── generated.go
│   └── wallet/
│       ├── repository.go
│       ├── repository_test.go
│       ├── service.go
│       ├── service_test.go
│       ├── resolver.go
│       └── resolver_test.go
├── .env
├── .gitignore
├── go.mod
├── go.sum
├── docker-compose.yml
├── gqlgen.yml
├── README.markdown
└── server.go
```

## Functionality

The API provides the following GraphQL operations:
- **Mutation: `transfer`**:
  - **Input**: `srcAddress` (sender's wallet address), `dstAddress` (recipient's wallet address), `amount` (number of tokens to transfer).
  - **Output**: `TransferResult` with the sender's updated balance (`balance`).
  - **Validations**:
    - The amount must be greater than 0 (returns error: "amount must be greater than 0").
    - The sender's wallet must exist (returns error: "source wallet does not exist").
    - The sender must have sufficient tokens (returns error: "insufficient balance").
  - **Race Conditions**: The mutation handles concurrent transfers using the `SERIALIZABLE` isolation level to ensure data consistency. If a transaction fails due to concurrency, it is retried up to 10 times (returns error: "transaction failed after 10 retries" if unsuccessful). For example, if a wallet has 10 tokens and three simultaneous transfers are requested (+1, -4, -7), possible outcomes are:
    - -4 accepted, -7 rejected, +1 accepted (balance: 7).
    - -7 accepted, -4 rejected, +1 accepted (balance: 4).
    - -4, +1, and -7 accepted (balance: 0).
- **Query: `canAfford`**:
  - **Input**: `address` (wallet address), `amount` (number of tokens to check).
  - **Output**: Boolean indicating whether the wallet has sufficient funds.
  - **Validations**:
    - The wallet must exist (returns error: "wallet not found").
    - The address must not be empty (returns error: "address is required").
### Branches
The project uses three branches with different database initializations:
- **master**: Production-ready branch. Uses migrations from `db/migrations/prod/`. The database contains one wallet (`0x0000000000000000000000000000000000000000`) holding 1,000,000 BTP tokens.
- **staging**: Pre-production testing branch. Uses migrations from `db/migrations/test/`, same as `develop`.
- **develop**: Development and testing branch. Uses migrations from `db/migrations/test/`. The database is initialized with three wallets:
  - `0x0000000000000000000000000000000000000000`: 1,000,000 BTP
  - `0x1000000000000000000000000000000000000000`: 10 BTP
  - `0x2000000000000000000000000000000000000000`: 1 BTP
## Configuration

The project requires a `.env` file to configure the database and server settings. The following environment variables must be defined:

```
DB_USER=user
DB_PASSWORD=user123
DB_NAME=btp
DB_HOST=localhost
DB_PORT=5432
PORT_GRAPHQL=8080
```
> **Note:** The included `.env` file is for local development only and should **not** be committed with sensitive or production credentials.

### Database Connection
- **PostgreSQL Settings**:
  - The database runs in a Docker container with the following defaults (defined in `docker-compose.yml`):
    - User: `user`
    - Password: `user123`
    - Database name: `btp`
  - For local development:
    - Set `DB_HOST=localhost` if the Go app runs outside Docker.
    - Set `DB_HOST=postgres` if both the app and database run in Docker.
  - Port: `5432` (mapped to the host's `DB_PORT`).

### GraphQL Server
- The server listens on the port specified in `PORT_GRAPHQL` (default: `8080`).
## Project Setup

### 1. Clone the Repository
Clone the project to your local machine:

```bash
git clone https://github.com/agorski1/Token-Transfer-API
cd Token-Transfer-API
git checkout master  # or git checkout develop for test data
```

### 2. Install Go Dependencies
Ensure all dependencies are installed:

```bash
go mod tidy
```

### 3. Start the PostgreSQL Database
The PostgreSQL database runs in a Docker container using Docker Compose.

#### a) Run the Database
```bash
docker-compose up
```

This starts the postgres service. On the `master` branch, migrations are applied from `db/migrations/prod/`. On the `develop` and `staging` branches, migrations are applied from `db/migrations/test/`.
#### b) Stopping the Database
To stop and remove the container:

```bash
docker-compose down
```

### 4. Run the Go Application Locally
Run the GraphQL server locally:

```bash
go run server.go
```

The GraphQL server will be available at `http://localhost:8080`.

### 5. Testing the GraphQL API
1. Open a browser and navigate to `http://localhost:8080` to access the **GraphQL Playground**.
2. Execute a sample `transfer` mutation:

```graphql
mutation {
  transfer(srcAddress: "0x0000000000000000000000000000000000000000", dstAddress: "0x1000000000000000000000000000000000000000", amount: 1000) {
    balance
  }
}
```

Expected response:
```json
{
  "data": {
    "transfer": {
      "balance": 999000
    }
  }
}
```

**Error Examples**:
- Insufficient balance:
```graphql
mutation {
  transfer(srcAddress: "0x0000000000000000000000000000000000000000", dstAddress: "0x1000000000000000000000000000000000000000", amount: 2000000) {
    balance
  }
}
```
Response:
```json
{
  "errors": [
    {
      "message": "insufficient balance",
      "path": [
        "transfer"
      ]
    }
  ],
  "data": null
}
```

- Negative amount:
```graphql
mutation {
  transfer(srcAddress: "0x0000000000000000000000000000000000000000", dstAddress: "0x1000000000000000000000000000000000000000", amount: -1000) {
    balance
  }
}
```
Response:
```json
{
  "errors": [
    {
      "message": "amount must be greater than 0",
      "path": [
        "transfer"
      ]
    }
  ],
  "data": null
}
```

### 6. Running Tests

The project includes integration and end-to-end tests in the `wallet/` and `graph/` directories, covering:
- Transfer correctness with various amounts.
- Error cases (negative amount, insufficient balance, non-existent wallet).
- Race conditions for concurrent transfers.

To run the tests, switch to the `develop` branch and execute:

```bash
git checkout develop
go test ./internal/graph ./internal/wallet
```

## License
This project is licensed under the MIT License. See the `LICENSE` file for details.