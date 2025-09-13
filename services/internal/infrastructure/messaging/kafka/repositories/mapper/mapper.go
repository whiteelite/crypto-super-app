package mapper

import (
	"encoding/base64"

	json "github.com/goccy/go-json"

	"github.com/google/uuid"
	"github.com/whiteelite/superapp/internal/infrastructure/messaging/kafka/repositories/models"
	shared "github.com/whiteelite/superapp/pkg/shared/domain/entities"
)

func ToMessage[T shared.Entity](entity *T) (*models.Message, error) {
	serialized, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}

	hash := base64.StdEncoding.EncodeToString(serialized)

	return &models.Message{
		ID:      uuid.New(),
		Content: string(serialized),
		Hash:    hash,
	}, nil
}

func FromMessage[T shared.Entity](message *models.Message) (*T, error) {
	entity := new(T)
	if err := json.Unmarshal([]byte(message.Content), entity); err != nil {
		return nil, err
	}

	return entity, nil
}
