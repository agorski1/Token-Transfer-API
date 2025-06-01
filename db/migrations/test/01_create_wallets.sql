CREATE TABLE wallets (
    address VARCHAR(42) PRIMARY KEY,
    balance BIGINT NOT NULL CHECK (balance >= 0)
);

CREATE INDEX idx_wallet_address ON wallets (address);