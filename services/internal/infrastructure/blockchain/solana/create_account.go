package sdk

import (
	"github.com/blocto/solana-go-sdk/types"
	"github.com/mr-tron/base58"
	entities "github.com/whiteelite/superapp/internal/domain/entities/solana"
	"github.com/whiteelite/superapp/internal/infrastructure/blockchain/solana/mappers"
	"github.com/whiteelite/superapp/internal/infrastructure/blockchain/solana/models"
)

func (c *Client) CreateAccount() entities.Account {
	account := types.NewAccount()

	return mappers.FromAccount(models.Account{
		PrivateKey: base58.Encode(account.PrivateKey),
		PublicKey:  base58.Encode(account.PrivateKey[32:]),
	})
}
