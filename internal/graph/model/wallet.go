package model

type Wallet struct {
	Address string `gorm:"primaryKey;size:42"`
	Balance int32  `gorm:"check:balance >= 0"`
}
