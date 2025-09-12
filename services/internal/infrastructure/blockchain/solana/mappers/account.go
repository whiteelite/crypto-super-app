package mappers

import (
	entities "github.com/whiteelite/superapp/services/internal/domain/entities/solana"
	"github.com/whiteelite/superapp/services/internal/infrastructure/blockchain/solana/models"
)

func ToAccount(entity entities.Account) models.Account {
	return models.Account{
		PublicKey:  entity.PublicKey,
		PrivateKey: entity.PrivateKey,
	}
}

func FromAccount(model models.Account) entities.Account {
	return entities.Account{
		PublicKey:  model.PublicKey,
		PrivateKey: model.PrivateKey,
	}
}
