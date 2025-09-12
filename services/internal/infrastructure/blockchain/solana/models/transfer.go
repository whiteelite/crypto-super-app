package models

type Transfer struct {
	Type        string
	Source      string
	Destination string
	Authority   string
	TokenMint   string
	Amount      string
	IsInner     bool
}
