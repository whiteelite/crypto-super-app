package models

import "github.com/google/uuid"

type Message struct {
	ID      uuid.UUID `json:"id"`
	Content string    `json:"content"`
	Hash    string    `json:"hash"`
}
