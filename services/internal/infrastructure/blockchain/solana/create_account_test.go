package sdk_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/mr-tron/base58"
	sdk "github.com/whiteelite/superapp/internal/infrastructure/blockchain/solana"
)

func TestCreateAccount_ReturnsValidBase58Keys(t *testing.T) {
	client := &sdk.Client{}

	acc := client.CreateAccount()

	if acc.PrivateKey == "" {
		t.Fatalf("expected non-empty private key")
	}
	if acc.PublicKey == "" {
		t.Fatalf("expected non-empty public key")
	}

	priv, err := base58.Decode(acc.PrivateKey)
	if err != nil {
		t.Fatalf("private key is not valid base58: %v", err)
	}
	if len(priv) != 64 {
		t.Fatalf("unexpected private key length: got %d, want 64", len(priv))
	}

	pub, err := base58.Decode(acc.PublicKey)
	if err != nil {
		t.Fatalf("public key is not valid base58: %v", err)
	}
	if len(pub) != 32 {
		t.Fatalf("unexpected public key length: got %d, want 32", len(pub))
	}

	if got, want := priv[32:], pub; string(got) != string(want) {
		t.Fatalf("public key mismatch: does not match last 32 bytes of private key")
	}
}

func TestCreateAccount_KeysAreUsableForSignature(t *testing.T) {
	client := &sdk.Client{}
	acc := client.CreateAccount()

	privBytes, err := base58.Decode(acc.PrivateKey)
	if err != nil {
		t.Fatalf("private key is not valid base58: %v", err)
	}
	if len(privBytes) != 64 {
		t.Fatalf("unexpected private key length: got %d, want 64", len(privBytes))
	}
	pubBytes, err := base58.Decode(acc.PublicKey)
	if err != nil {
		t.Fatalf("public key is not valid base58: %v", err)
	}

	message := []byte("superapp solana keypair test")
	signature := ed25519.Sign(ed25519.PrivateKey(privBytes), message)
	if ok := ed25519.Verify(ed25519.PublicKey(pubBytes), message, signature); !ok {
		t.Fatalf("signature verification failed with generated keypair")
	}
}

