package sdk

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/mr-tron/base58"
	models "github.com/whiteelite/superapp/services/internal/infrastructure/blockchain/solana/models"
)

type Client struct {
	c *client.Client
}

// Network defines Solana cluster
type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkDevnet  Network = "devnet"
	NetworkTestnet Network = "testnet"
)

func DefaultRPCURL(network Network) string {
	switch network {
	case NetworkMainnet:
		return "https://api.mainnet-beta.solana.com"
	case NetworkTestnet:
		return "https://api.testnet.solana.com"
	case NetworkDevnet:
		fallthrough
	default:
		return "https://api.devnet.solana.com"
	}
}

func NewClient(rpcURL string) *Client {
	return &Client{c: client.NewClient(rpcURL)}
}

func NewClientForNetwork(network Network) *Client {
	return NewClient(DefaultRPCURL(network))
}

func (c *Client) SwitchNetworkByURL(rpcURL string) {
	c.c = client.NewClient(rpcURL)
}

func (c *Client) SwitchNetwork(network Network) {
	c.SwitchNetworkByURL(DefaultRPCURL(network))
}

// GetBalance returns balance in lamports for a given public key (base58)
func (c *Client) GetBalance(ctx context.Context, req models.BalanceRequest) (uint64, error) {
	pub := common.PublicKeyFromString(req.PublicKey)
	bal, err := c.c.GetBalance(ctx, pub.ToBase58())
	if err != nil {
		return 0, err
	}
	return bal, nil
}

// RequestAirdrop requests airdrop to the given public key (base58) in lamports
func (c *Client) RequestAirdrop(ctx context.Context, req models.AirdropRequest) (string, error) {
	pub := common.PublicKeyFromString(req.PublicKey)
	sig, err := c.c.RequestAirdrop(ctx, pub.ToBase58(), req.Lamports)
	if err != nil {
		return "", err
	}
	return sig, nil
}

// TransferSOL sends lamports from a private key (base58 64 bytes) to recipient public key (base58)
func (c *Client) TransferSOL(ctx context.Context, req models.TransferSOLRequest) (string, error) {
	privBytes, err := base58.Decode(req.FromPrivateKey)
	if err != nil {
		return "", err
	}
	if len(privBytes) != 64 {
		return "", fmt.Errorf("invalid private key length: expected 64 bytes")
	}

	sender, err := types.AccountFromBytes(privBytes)
	if err != nil {
		return "", err
	}

	recent, err := c.c.GetLatestBlockhash(ctx)
	if err != nil {
		return "", err
	}

	to := common.PublicKeyFromString(req.ToPublicKey)

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        sender.PublicKey,
			RecentBlockhash: recent.Blockhash,
			Instructions: []types.Instruction{
				// SystemProgram Transfer
				{
					ProgramID: common.SystemProgramID,
					Accounts: []types.AccountMeta{
						{PubKey: sender.PublicKey, IsSigner: true, IsWritable: true},
						{PubKey: to, IsSigner: false, IsWritable: true},
					},
					Data: append([]byte{2}, // Transfer instruction index in System Program
						uint8(req.Lamports), uint8(req.Lamports>>8), uint8(req.Lamports>>16), uint8(req.Lamports>>24),
						uint8(req.Lamports>>32), uint8(req.Lamports>>40), uint8(req.Lamports>>48), uint8(req.Lamports>>56),
					),
				},
			},
		}),
		Signers: []types.Account{sender},
	})
	if err != nil {
		return "", err
	}

	sig, err := c.c.SendTransaction(ctx, tx)
	if err != nil {
		return "", err
	}
	return sig, nil
}

// GetMinimumBalanceForRentExemption returns required lamports for an account of given size
func (c *Client) GetMinimumBalanceForRentExemption(ctx context.Context, req models.RentRequest) (uint64, error) {
	return c.c.GetMinimumBalanceForRentExemption(ctx, req.DataLen)
}

// DeriveAssociatedTokenAddress derives ATA PDA for owner+mint
func (c *Client) DeriveAssociatedTokenAddress(req models.DeriveATARequest) (string, error) {
	owner := common.PublicKeyFromString(req.Owner)
	mint := common.PublicKeyFromString(req.Mint)
	seeds := [][]byte{
		owner.Bytes(),
		common.TokenProgramID.Bytes(),
		mint.Bytes(),
	}
	pda, _, err := common.FindProgramAddress(seeds, common.SPLAssociatedTokenAccountProgramID)
	if err != nil {
		return "", err
	}
	return pda.ToBase58(), nil
}

// CreateAssociatedTokenAccountIfNotExists creates ATA for (owner,mint) if missing; returns ATA and optional signature
func (c *Client) CreateAssociatedTokenAccountIfNotExists(ctx context.Context, req models.CreateATARequest) (string, string, error) {
	ata, err := c.DeriveAssociatedTokenAddress(models.DeriveATARequest{Owner: req.Owner, Mint: req.Mint})
	if err != nil {
		return "", "", err
	}
	// if exists, nothing to do
	if _, err := c.c.GetAccountInfo(ctx, ata); err == nil {
		return ata, "", nil
	}

	payerPriv, err := base58.Decode(req.PayerPrivateKey)
	if err != nil {
		return "", "", err
	}
	payer, err := types.AccountFromBytes(payerPriv)
	if err != nil {
		return "", "", err
	}

	owner := common.PublicKeyFromString(req.Owner)
	mint := common.PublicKeyFromString(req.Mint)

	recent, err := c.c.GetLatestBlockhash(ctx)
	if err != nil {
		return "", "", err
	}

	inst := types.Instruction{
		ProgramID: common.SPLAssociatedTokenAccountProgramID,
		Accounts: []types.AccountMeta{
			{PubKey: payer.PublicKey, IsSigner: true, IsWritable: true},
			{PubKey: common.PublicKeyFromString(ata), IsSigner: false, IsWritable: true},
			{PubKey: owner, IsSigner: false, IsWritable: false},
			{PubKey: mint, IsSigner: false, IsWritable: false},
			{PubKey: common.SystemProgramID, IsSigner: false, IsWritable: false},
			{PubKey: common.TokenProgramID, IsSigner: false, IsWritable: false},
			{PubKey: common.SysVarRentPubkey, IsSigner: false, IsWritable: false},
		},
		Data: []byte{},
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        payer.PublicKey,
			RecentBlockhash: recent.Blockhash,
			Instructions:    []types.Instruction{inst},
		}),
		Signers: []types.Account{payer},
	})
	if err != nil {
		return "", "", err
	}
	sig, err := c.c.SendTransaction(ctx, tx)
	if err != nil {
		return "", "", err
	}
	return ata, sig, nil
}

// GetMintDecimals reads decimals from Mint account data at offset 44
func (c *Client) GetMintDecimals(ctx context.Context, req models.GetMintDecimalsRequest) (uint8, error) {
	acc, err := c.c.GetAccountInfo(ctx, req.Mint)
	if err != nil {
		return 0, err
	}
	if len(acc.Data) < 45 {
		return 0, fmt.Errorf("invalid mint account data")
	}
	return acc.Data[44], nil
}

// TransferTokenChecked performs SPL token transfer with decimals check
func (c *Client) TransferTokenChecked(ctx context.Context, req models.TransferTokenCheckedRequest) (string, error) {
	priv, err := base58.Decode(req.AuthorityPrivateKey)
	if err != nil {
		return "", err
	}
	authority, err := types.AccountFromBytes(priv)
	if err != nil {
		return "", err
	}

	recent, err := c.c.GetLatestBlockhash(ctx)
	if err != nil {
		return "", err
	}

	src := common.PublicKeyFromString(req.SourceATA)
	dst := common.PublicKeyFromString(req.DestinationATA)
	mint := common.PublicKeyFromString(req.Mint)

	// token.TransferChecked instruction layout
	data := make([]byte, 0, 1+8+1)
	data = append(data, byte(12)) // TransferChecked
	// amount little-endian u64
	data = append(data,
		byte(req.Amount), byte(req.Amount>>8), byte(req.Amount>>16), byte(req.Amount>>24),
		byte(req.Amount>>32), byte(req.Amount>>40), byte(req.Amount>>48), byte(req.Amount>>56),
	)
	data = append(data, byte(req.Decimals))

	inst := types.Instruction{
		ProgramID: common.TokenProgramID,
		Accounts: []types.AccountMeta{
			{PubKey: src, IsSigner: false, IsWritable: true},
			{PubKey: mint, IsSigner: false, IsWritable: false},
			{PubKey: dst, IsSigner: false, IsWritable: true},
			{PubKey: authority.PublicKey, IsSigner: true, IsWritable: false},
		},
		Data: data,
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        authority.PublicKey,
			RecentBlockhash: recent.Blockhash,
			Instructions:    []types.Instruction{inst},
		}),
		Signers: []types.Account{authority},
	})
	if err != nil {
		return "", err
	}
	return c.c.SendTransaction(ctx, tx)
}

// CreateMint creates a new SPL Mint and initializes it
func (c *Client) CreateMint(ctx context.Context, req models.CreateMintRequest) (string, string, error) {
	payerPriv, err := base58.Decode(req.PayerPrivateKey)
	if err != nil {
		return "", "", err
	}
	payer, err := types.AccountFromBytes(payerPriv)
	if err != nil {
		return "", "", err
	}

	mintAccount := types.NewAccount()
	mintPub := mintAccount.PublicKey.ToBase58()

	// Rent for Mint account; typical size is 82 bytes
	const mintAccountSize = 82
	rent, err := c.GetMinimumBalanceForRentExemption(ctx, models.RentRequest{DataLen: mintAccountSize})
	if err != nil {
		return "", "", err
	}

	recent, err := c.c.GetLatestBlockhash(ctx)
	if err != nil {
		return "", "", err
	}

	// SystemProgram CreateAccount for Mint
	createMint := types.Instruction{
		ProgramID: common.SystemProgramID,
		Accounts: []types.AccountMeta{
			{PubKey: payer.PublicKey, IsSigner: true, IsWritable: true},
			{PubKey: mintAccount.PublicKey, IsSigner: true, IsWritable: true},
		},
		Data: func() []byte {
			// CreateAccount: 0, lamports u64, space u64, owner pubkey
			data := []byte{0}
			lam := rent
			space := uint64(mintAccountSize)
			data = append(data,
				byte(lam), byte(lam>>8), byte(lam>>16), byte(lam>>24), byte(lam>>32), byte(lam>>40), byte(lam>>48), byte(lam>>56),
			)
			data = append(data,
				byte(space), byte(space>>8), byte(space>>16), byte(space>>24), byte(space>>32), byte(space>>40), byte(space>>48), byte(space>>56),
			)
			data = append(data, common.TokenProgramID.Bytes()...)
			return data
		}(),
	}

	// token.InitializeMint2
	mintAuthority := common.PublicKeyFromString(req.MintAuthority)
	initMint := types.Instruction{
		ProgramID: common.TokenProgramID,
		Accounts: []types.AccountMeta{
			{PubKey: mintAccount.PublicKey, IsSigner: false, IsWritable: true},
		},
		Data: func() []byte {
			// InitializeMint2: 20, decimals u8, mintAuthority pubkey, freezeAuthorityOption u8(0 = none)
			data := []byte{20}
			data = append(data, byte(req.Decimals))
			data = append(data, mintAuthority.Bytes()...)
			data = append(data, 0) // no freeze authority
			return data
		}(),
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        payer.PublicKey,
			RecentBlockhash: recent.Blockhash,
			Instructions:    []types.Instruction{createMint, initMint},
		}),
		Signers: []types.Account{payer, mintAccount},
	})
	if err != nil {
		return "", "", err
	}
	sig, err := c.c.SendTransaction(ctx, tx)
	if err != nil {
		return "", "", err
	}
	return mintPub, sig, nil
}

// MintTo mints tokens to a destination ATA
func (c *Client) MintTo(ctx context.Context, req models.MintToRequest) (string, error) {
	priv, err := base58.Decode(req.MintAuthorityPrivateKey)
	if err != nil {
		return "", err
	}
	authority, err := types.AccountFromBytes(priv)
	if err != nil {
		return "", err
	}

	recent, err := c.c.GetLatestBlockhash(ctx)
	if err != nil {
		return "", err
	}

	mint := common.PublicKeyFromString(req.Mint)
	dest := common.PublicKeyFromString(req.DestinationATA)

	data := make([]byte, 0, 1+8)
	data = append(data, byte(7)) // MintTo
	data = append(data,
		byte(req.Amount), byte(req.Amount>>8), byte(req.Amount>>16), byte(req.Amount>>24),
		byte(req.Amount>>32), byte(req.Amount>>40), byte(req.Amount>>48), byte(req.Amount>>56),
	)
	inst := types.Instruction{
		ProgramID: common.TokenProgramID,
		Accounts: []types.AccountMeta{
			{PubKey: mint, IsSigner: false, IsWritable: true},
			{PubKey: dest, IsSigner: false, IsWritable: true},
			{PubKey: authority.PublicKey, IsSigner: true, IsWritable: false},
		},
		Data: data,
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        authority.PublicKey,
			RecentBlockhash: recent.Blockhash,
			Instructions:    []types.Instruction{inst},
		}),
		Signers: []types.Account{authority},
	})
	if err != nil {
		return "", err
	}
	return c.c.SendTransaction(ctx, tx)
}

var tokenMetadataProgramID = common.PublicKeyFromString("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")

// GetTokenMetadata fetches Metaplex metadata (name, symbol, uri) and mint decimals
func (c *Client) GetTokenMetadata(ctx context.Context, req models.GetTokenMetadataRequest) (*models.TokenMetadata, error) {
	mint := common.PublicKeyFromString(req.Mint)
	metaPDA, err := deriveMetadataPDA(mint)
	if err != nil {
		return nil, err
	}
	acc, err := c.c.GetAccountInfo(ctx, metaPDA.ToBase58())
	if err != nil {
		return nil, err
	}
	if len(acc.Data) < 1+32+32+4 {
		return nil, fmt.Errorf("invalid metadata account data")
	}
	// Skip: key(1) + updateAuthority(32) + mint(32)
	offset := 1 + 32 + 32
	name, off, err := parseBorshString(acc.Data, offset)
	if err != nil {
		return nil, err
	}
	symbol, off2, err := parseBorshString(acc.Data, off)
	if err != nil {
		return nil, err
	}
	uri, _, err := parseBorshString(acc.Data, off2)
	if err != nil {
		return nil, err
	}
	dec, err := c.GetMintDecimals(ctx, models.GetMintDecimalsRequest{Mint: req.Mint})
	if err != nil {
		return nil, err
	}
	return &models.TokenMetadata{
		Name:     name,
		Symbol:   symbol,
		URI:      uri,
		Decimals: dec,
	}, nil
}

// SetTokenMetadata updates name/symbol/uri via Metaplex UpdateMetadataAccountV2
func (c *Client) SetTokenMetadata(ctx context.Context, req models.SetTokenMetadataRequest) (string, error) {
	priv, err := base58.Decode(req.UpdateAuthorityPrivateKey)
	if err != nil {
		return "", err
	}
	updateAuth, err := types.AccountFromBytes(priv)
	if err != nil {
		return "", err
	}

	mint := common.PublicKeyFromString(req.Mint)
	metadataPDA, err := deriveMetadataPDA(mint)
	if err != nil {
		return "", err
	}

	// Build instruction data for UpdateMetadataAccountV2
	var b bytes.Buffer
	b.WriteByte(15) // UpdateMetadataAccountV2 discriminator
	b.WriteByte(1)  // Some(DataV2)
	writeBorshString(&b, req.Name)
	writeBorshString(&b, req.Symbol)
	writeBorshString(&b, req.URI)
	_ = binary.Write(&b, binary.LittleEndian, uint16(0)) // seller_fee_basis_points
	b.WriteByte(0)                                       // creators None
	b.WriteByte(0)                                       // collection None
	b.WriteByte(0)                                       // uses None
	b.WriteByte(0)                                       // new_update_authority None
	b.WriteByte(0)                                       // primary_sale_happened None
	b.WriteByte(0)                                       // is_mutable None

	inst := types.Instruction{
		ProgramID: tokenMetadataProgramID,
		Accounts: []types.AccountMeta{
			{PubKey: metadataPDA, IsSigner: false, IsWritable: true},
			{PubKey: updateAuth.PublicKey, IsSigner: true, IsWritable: false},
		},
		Data: b.Bytes(),
	}

	recent, err := c.c.GetLatestBlockhash(ctx)
	if err != nil {
		return "", err
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        updateAuth.PublicKey,
			RecentBlockhash: recent.Blockhash,
			Instructions:    []types.Instruction{inst},
		}),
		Signers: []types.Account{updateAuth},
	})
	if err != nil {
		return "", err
	}

	return c.c.SendTransaction(ctx, tx)
}

func deriveMetadataPDA(mint common.PublicKey) (common.PublicKey, error) {
	seeds := [][]byte{
		[]byte("metadata"),
		tokenMetadataProgramID.Bytes(),
		mint.Bytes(),
	}
	pda, _, err := common.FindProgramAddress(seeds, tokenMetadataProgramID)
	if err != nil {
		return common.PublicKey{}, err
	}
	return pda, nil
}

func parseBorshString(data []byte, offset int) (string, int, error) {
	if len(data) < offset+4 {
		return "", 0, fmt.Errorf("unexpected end of data while reading string length")
	}
	ln := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	end := offset + int(ln)
	if end > len(data) {
		return "", 0, fmt.Errorf("unexpected end of data while reading string bytes")
	}
	return string(data[offset:end]), end, nil
}

func writeBorshString(b *bytes.Buffer, s string) {
	length := uint32(len(s))
	_ = binary.Write(b, binary.LittleEndian, length)
	b.WriteString(s)
}

// GetTransactionTransfersSPL parses SPL token transfers from a confirmed transaction
func (c *Client) GetTransactionTransfersSPL(ctx context.Context, req models.GetTransactionTransfersRequest) ([]*models.Transfer, error) {
	tx, err := c.c.GetTransaction(ctx, req.Signature)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, fmt.Errorf("transaction not found")
	}

	indexAccountMap := map[int]string{}
	for i, acc := range tx.Transaction.Message.Accounts {
		indexAccountMap[i] = acc.String()
	}

	var transfers []*models.Transfer

	// outer instructions
	for i, inst := range tx.Transaction.Message.Instructions {
		programID := indexAccountMap[int(inst.ProgramIDIndex)]
		if programID == common.TokenProgramID.String() {
			if t := tryDecodeTransfer(inst, indexAccountMap); t != nil {
				transfers = append(transfers, t)
				transfers[len(transfers)-1].IsInner = false
			}
		}
		// inner for this outer index
		if tx.Meta != nil {
			for _, inner := range tx.Meta.InnerInstructions {
				if int(inner.Index) != i {
					continue
				}
				for _, inInst := range inner.Instructions {
					p := indexAccountMap[inInst.ProgramIDIndex]
					if p == common.TokenProgramID.String() {
						if t := tryDecodeTransfer(inInst, indexAccountMap); t != nil {
							t.IsInner = true
							transfers = append(transfers, t)
						}
					}
				}
			}
		}
	}

	return transfers, nil
}

func tryDecodeTransfer(inst types.CompiledInstruction, indexAccountMap map[int]string) *models.Transfer {
	if len(inst.Data) == 0 {
		return nil
	}
	instrType := inst.Data[0]
	switch instrType {
	case 3: // Transfer
		if len(inst.Data) < 9 || len(inst.Accounts) < 3 {
			return nil
		}
		amount := uint64(inst.Data[1]) |
			uint64(inst.Data[2])<<8 |
			uint64(inst.Data[3])<<16 |
			uint64(inst.Data[4])<<24 |
			uint64(inst.Data[5])<<32 |
			uint64(inst.Data[6])<<40 |
			uint64(inst.Data[7])<<48 |
			uint64(inst.Data[8])<<56
		return &models.Transfer{
			Type:        "transfer",
			Source:      indexAccountMap[inst.Accounts[0]],
			Destination: indexAccountMap[inst.Accounts[1]],
			Authority:   indexAccountMap[inst.Accounts[2]],
			TokenMint:   "",
			Amount:      fmt.Sprintf("%d", amount),
		}
	case 12: // TransferChecked
		if len(inst.Data) < 10 || len(inst.Accounts) < 4 {
			return nil
		}
		amount := uint64(inst.Data[1]) |
			uint64(inst.Data[2])<<8 |
			uint64(inst.Data[3])<<16 |
			uint64(inst.Data[4])<<24 |
			uint64(inst.Data[5])<<32 |
			uint64(inst.Data[6])<<40 |
			uint64(inst.Data[7])<<48 |
			uint64(inst.Data[8])<<56
		return &models.Transfer{
			Type:        "transferChecked",
			Source:      indexAccountMap[inst.Accounts[0]],
			Destination: indexAccountMap[inst.Accounts[2]],
			Authority:   indexAccountMap[inst.Accounts[3]],
			TokenMint:   indexAccountMap[inst.Accounts[1]],
			Amount:      fmt.Sprintf("%d", amount),
		}
	default:
		return nil
	}
}

// GetTokenAccount returns minimal parsed info of a token account (ATA)
func (c *Client) GetTokenAccount(ctx context.Context, req models.GetTokenAccountRequest) (*models.TokenAccount, error) {
	acc, err := c.c.GetAccountInfo(ctx, req.ATA)
	if err != nil {
		return nil, err
	}
	if len(acc.Data) < 72 {
		return nil, fmt.Errorf("invalid token account data")
	}
	mint := base58.Encode(acc.Data[0:32])
	owner := base58.Encode(acc.Data[32:64])
	amount := binary.LittleEndian.Uint64(acc.Data[64:72])
	return &models.TokenAccount{Mint: mint, Owner: owner, Amount: amount}, nil
}

// GetTokenMintFromATA returns mint address for a given token account (ATA)
func (c *Client) GetTokenMintFromATA(ctx context.Context, req models.GetTokenMintFromATARequest) (string, error) {
	ta, err := c.GetTokenAccount(ctx, models.GetTokenAccountRequest{ATA: req.ATA})
	if err != nil {
		return "", err
	}
	return ta.Mint, nil
}
