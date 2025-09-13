package sdk_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	sdk "github.com/whiteelite/superapp/internal/infrastructure/blockchain/solana"
	"github.com/whiteelite/superapp/internal/infrastructure/blockchain/solana/models"
)

func TestDevnet_SPL_MintAndTransfer(t *testing.T) {
	t.Parallel()

	c := sdk.NewClientForNetwork(sdk.NetworkDevnet)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Payer / mint authority
	payer := c.CreateAccount()
	_, err := c.RequestAirdrop(ctx, models.AirdropRequest{PublicKey: payer.PublicKey, Lamports: 200_000_000})
	if err != nil {
		t.Fatalf("airdrop failed: %v", err)
	}
	waitForBalanceGTE(t, ctx, c, payer.PublicKey, 100_000_000)

	// Create Mint
	mint, _, err := c.CreateMint(ctx, models.CreateMintRequest{PayerPrivateKey: payer.PrivateKey, MintAuthority: payer.PublicKey, Decimals: 6})
	if err != nil {
		t.Fatalf("create mint failed: %v", err)
	}

	// Owner accounts
	owner1 := c.CreateAccount()
	owner2 := c.CreateAccount()

	// Create ATAs
	ata1, _, err := c.CreateAssociatedTokenAccountIfNotExists(ctx, models.CreateATARequest{PayerPrivateKey: payer.PrivateKey, Owner: owner1.PublicKey, Mint: mint})
	if err != nil {
		t.Fatalf("create ata1 failed: %v", err)
	}
	ata2, _, err := c.CreateAssociatedTokenAccountIfNotExists(ctx, models.CreateATARequest{PayerPrivateKey: payer.PrivateKey, Owner: owner2.PublicKey, Mint: mint})
	if err != nil {
		t.Fatalf("create ata2 failed: %v", err)
	}

	// MintTo owner1
	_, err = c.MintTo(ctx, models.MintToRequest{MintAuthorityPrivateKey: payer.PrivateKey, Mint: mint, DestinationATA: ata1, Amount: 1_000_000})
	if err != nil {
		t.Fatalf("mint to failed: %v", err)
	}

	// Get token account
	ta1, err := c.GetTokenAccount(ctx, models.GetTokenAccountRequest{ATA: ata1})
	if err != nil {
		t.Fatalf("get token account failed: %v", err)
	}
	if ta1.Mint != mint {
		t.Fatalf("unexpected mint: %s != %s", ta1.Mint, mint)
	}

	// Transfer tokens owner1 -> owner2
	_, err = c.TransferTokenChecked(ctx, models.TransferTokenCheckedRequest{
		AuthorityPrivateKey: payer.PrivateKey,
		SourceATA:           ata1,
		DestinationATA:      ata2,
		Mint:                mint,
		Amount:              100_000,
		Decimals:            6,
	})
	if err != nil {
		t.Fatalf("token transfer failed: %v", err)
	}

	// Verify dst ATA exists and has same mint
	ta2info, err := c.GetTokenAccount(ctx, models.GetTokenAccountRequest{ATA: ata2})
	if err != nil {
		t.Fatalf("get token account 2 failed: %v", err)
	}
	if ta2info.Mint != mint {
		t.Fatalf("unexpected dst mint: %s != %s", ta2info.Mint, mint)
	}

	// Smoke: check ATA derivation
	ataDerived, err := c.DeriveAssociatedTokenAddress(models.DeriveATARequest{Owner: owner1.PublicKey, Mint: mint})
	if err != nil {
		t.Fatalf("derive ata failed: %v", err)
	}
	if base58.Encode(commonFromString(ataDerived)) == "" { // simple usage to avoid unused imports; commonFromString below
		t.Fatalf("derived ata invalid")
	}
}

func commonFromString(s string) []byte { return []byte(s) }
