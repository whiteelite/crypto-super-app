package sdk_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/whiteelite/superapp/services/internal/infrastructure/blockchain/solana"
	"github.com/whiteelite/superapp/services/internal/infrastructure/blockchain/solana/models"
)

func TestDevnet_SOL_AirdropAndTransfer(t *testing.T) {
	t.Parallel()

	c := sdk.NewClientForNetwork(sdk.NetworkDevnet)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create two accounts
	acc1 := c.CreateAccount()
	acc2 := c.CreateAccount()

	// Airdrop to acc1
	_, err := c.RequestAirdrop(ctx, models.AirdropRequest{PublicKey: acc1.PublicKey, Lamports: 100_000_000}) // 0.1 SOL
	if err != nil {
		t.Fatalf("airdrop failed: %v", err)
	}

	// Wait until balance >= requested amount
	waitForBalanceGTE(t, ctx, c, acc1.PublicKey, 50_000_000) // at least 0.05 SOL to cover fees

	// Transfer 0.02 SOL to acc2
	_, err = c.TransferSOL(ctx, models.TransferSOLRequest{
		FromPrivateKey: acc1.PrivateKey,
		ToPublicKey:    acc2.PublicKey,
		Lamports:       20_000_000,
	})
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
	}

	// Verify destination balance increased
	waitForBalanceGTE(t, ctx, c, acc2.PublicKey, 10_000_000)
}

func waitForBalanceGTE(t *testing.T, ctx context.Context, c *sdk.Client, pub string, want uint64) {
	t.Helper()
	for {
		bal, err := c.GetBalance(ctx, models.BalanceRequest{PublicKey: pub})
		if err == nil && bal >= want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting balance >= %d: last bal=%v err=%v", want, bal, err)
		case <-time.After(2 * time.Second):
		}
	}
}
