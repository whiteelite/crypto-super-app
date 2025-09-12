package models

type BalanceRequest struct {
	PublicKey string
}

type AirdropRequest struct {
	PublicKey string
	Lamports  uint64
}

type TransferSOLRequest struct {
	FromPrivateKey string
	ToPublicKey    string
	Lamports       uint64
}

type RentRequest struct {
	DataLen uint64
}

type DeriveATARequest struct {
	Owner string
	Mint  string
}

type CreateATARequest struct {
	PayerPrivateKey string
	Owner           string
	Mint            string
}

type GetMintDecimalsRequest struct {
	Mint string
}

type TransferTokenCheckedRequest struct {
	AuthorityPrivateKey string
	SourceATA           string
	DestinationATA      string
	Mint                string
	Amount              uint64
	Decimals            uint8
}

type CreateMintRequest struct {
	PayerPrivateKey string
	MintAuthority   string
	Decimals        uint8
}

type MintToRequest struct {
	MintAuthorityPrivateKey string
	Mint                    string
	DestinationATA          string
	Amount                  uint64
}

type GetTokenMetadataRequest struct {
	Mint string
}

type SetTokenMetadataRequest struct {
	UpdateAuthorityPrivateKey string
	Mint                      string
	Name                      string
	Symbol                    string
	URI                       string
}

type GetTransactionTransfersRequest struct {
	Signature string
}

type GetTokenAccountRequest struct {
	ATA string
}

type GetTokenMintFromATARequest struct {
	ATA string
}
