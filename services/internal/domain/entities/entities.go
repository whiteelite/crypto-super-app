package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/whiteelite/superapp/pkg/shared/domain/entities"
)

type (
	IDCard      string
	DigitalSign string
)

type User struct {
	entities.Entity

	IDCard      IDCard
	DigitalSign DigitalSign
}

type (
	PrivateKey string
	PublicKey  string
	Amount     decimal.Decimal
)

type CryptoWallet struct {
	entities.Entity

	PrivateKey PrivateKey
	PublicKey  PublicKey
	Amount     Amount
}

type (
	CVV        string
	CardNumber string
	OwnerName  string
	DueDate    time.Time
	Currency   string
)

type FiatWallet struct {
	entities.Entity

	CardNumber CardNumber
	OwnerName  OwnerName
	CVV        CVV
	DueDate    DueDate
	Amount     Amount
	Currency   Currency
}

type (
	WalletAddress interface{ PublicKey | CardNumber }
)

type WalletTransfer[T WalletAddress] struct {
	entities.Entity

	From   T
	To     T
	Amount Amount
}

type CryptoTransfer struct {
	entities.Entity

	From   PublicKey
	To     PublicKey
	Amount Amount
}

type CryptoExchangeTransfer struct{ CryptoTransfer }

type CryptoFundWallet struct {
	entities.Entity

	Owner      PublicKey
	PublicKey  PublicKey
	PrivateKey PrivateKey
	Amount     Amount
}

type CryptoFundContract struct {
	entities.Entity

	ContractContent string
	ShareAmount     *Amount
	FounderSign     DigitalSign
	MemberSign      DigitalSign
	DueDate         time.Time
}

type (
	Content          string
	IsKYC            bool
	ImageContentType string
)

const (
	ImageContentTypePrivate ImageContentType = "private"
	ImageContentTypePublic  ImageContentType = "public"
)

type Image struct {
	entities.Entity

	PublicImage  Content
	PrivateImage Content
	IsKYC        IsKYC
}

type Position string

const (
	PositionFounder Position = "founder"
	PositionMember  Position = "member"
)

type Roles string

const (
	RolesFounder Roles = "founder"
	RolesMember  Roles = "member"
)

type Score int32

type Profile struct {
	entities.Entity

	FirstName       string
	LastName        string
	Image           ImageContentType
	Position        Position
	Roles           Roles
	ReputationScore Score
}

type GeoPoint struct {
	entities.Entity

	Latitude  decimal.Decimal
	Longitude decimal.Decimal
}

type RealEstateOwnerType string

const (
	RealEstateOwnerTypeUser       RealEstateOwnerType = "user"
	RealEstateOwnerTypeCompany    RealEstateOwnerType = "company"
	RealEstateOwnerTypeGovernment RealEstateOwnerType = "government"
)

type RealEstateType string
type RealEstateSubType string

type RealEstateTypeContent struct {
	Content           Content
	RealEstateSubType *RealEstateSubType
}

type RealEstateInfo map[RealEstateType]*RealEstateTypeContent

type RealEstatePrice struct {
	Amount   Amount
	Currency Currency
}

type RealEstateRooms struct {
	Images []Image
	Count  uint8
}

type RealEstate struct {
	entities.Entity

	Owner     PublicKey
	Rent      bool
	GeoPoint  GeoPoint
	OwnerType RealEstateOwnerType
	Images    []Image
	Info      RealEstateInfo
	Rooms     *RealEstateRooms
}

type ContractID uuid.UUID

type Contract struct {
	entities.Entity

	ContractID      ContractID
	ContractContent Content
}

type RealEstateWallet struct {
	entities.Entity

	Contract   *Contract
	PublicKey  PublicKey
	PrivateKey PrivateKey
	Amount     Amount
}

type RealEstateContract struct {
	entities.Entity

	FounderSign DigitalSign
	MemberSign  DigitalSign
	Contract    Contract
	Amount      Amount
	Currency    Currency
	DueDate     time.Time
}
