package wallet

import (
	"crypto/sha256"
	"testing"
)

func TestNewWallet(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	if wallet.PrivateKey == nil {
		t.Error("Private key is nil")
	}

	if wallet.PublicKey == nil {
		t.Error("Public key is nil")
	}
}

func TestWalletKeys(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	privateKeyHex := wallet.GetPrivateKeyHex()
	if privateKeyHex == "" {
		t.Error("Private key hex is empty")
	}

	publicKeyHex := wallet.GetPublicKeyHex()
	if publicKeyHex == "" {
		t.Error("Public key hex is empty")
	}

	address := wallet.GetAddress()
	if address == "" {
		t.Error("Address is empty")
	}

	// Verifica que o endereço é um hash SHA-256 (64 caracteres hex)
	if len(address) != 64 {
		t.Errorf("Address should be 64 characters, got %d", len(address))
	}
}

func TestNewWalletFromPrivateKey(t *testing.T) {
	// Cria uma carteira original
	original, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create original wallet: %v", err)
	}

	privateKeyHex := original.GetPrivateKeyHex()

	// Recria a carteira a partir da chave privada
	restored, err := NewWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		t.Fatalf("Failed to restore wallet from private key: %v", err)
	}

	// Verifica que as chaves são iguais
	if original.GetPrivateKeyHex() != restored.GetPrivateKeyHex() {
		t.Error("Private keys do not match")
	}

	if original.GetPublicKeyHex() != restored.GetPublicKeyHex() {
		t.Error("Public keys do not match")
	}

	if original.GetAddress() != restored.GetAddress() {
		t.Error("Addresses do not match")
	}
}

func TestSignAndVerify(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	data := []byte("Hello, Krakovia Blockchain!")

	// Assina os dados
	signature, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	if signature == "" {
		t.Error("Signature is empty")
	}

	// Verifica a assinatura
	valid, err := Verify(wallet.GetPublicKeyHex(), data, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	if !valid {
		t.Error("Signature verification failed")
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	data := []byte("Original data")
	signature, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Tenta verificar com dados diferentes
	tamperedData := []byte("Tampered data")
	valid, err := Verify(wallet.GetPublicKeyHex(), tamperedData, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	if valid {
		t.Error("Signature verification should have failed for tampered data")
	}
}

func TestVerifyWithWrongPublicKey(t *testing.T) {
	wallet1, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet 1: %v", err)
	}

	wallet2, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet 2: %v", err)
	}

	data := []byte("Test data")
	signature, err := wallet1.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Tenta verificar com chave pública diferente
	valid, err := Verify(wallet2.GetPublicKeyHex(), data, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	if valid {
		t.Error("Signature verification should have failed with wrong public key")
	}
}

func TestAddressFromPublicKey(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	address1 := wallet.GetAddress()
	address2, err := AddressFromPublicKey(wallet.GetPublicKeyHex())
	if err != nil {
		t.Fatalf("Failed to get address from public key: %v", err)
	}

	if address1 != address2 {
		t.Error("Addresses do not match")
	}
}

func TestPublicKeyFromHex(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	publicKeyHex := wallet.GetPublicKeyHex()

	publicKey, err := PublicKeyFromHex(publicKeyHex)
	if err != nil {
		t.Fatalf("Failed to parse public key from hex: %v", err)
	}

	if publicKey == nil {
		t.Fatal("Public key is nil")
	}

	// Verifica que os valores X e Y são iguais
	if wallet.PublicKey.X.Cmp(publicKey.X) != 0 {
		t.Error("Public key X coordinates do not match")
	}

	if wallet.PublicKey.Y.Cmp(publicKey.Y) != 0 {
		t.Error("Public key Y coordinates do not match")
	}
}

func TestSignatureLength(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	data := []byte("Test signature length")
	signature, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// A assinatura ECDSA deve ter 64 bytes (32 bytes para r + 32 bytes para s)
	// Em hexadecimal, isso é 128 caracteres
	if len(signature) != 128 {
		t.Errorf("Expected signature length of 128 characters, got %d", len(signature))
	}
}

func TestDeterministicAddress(t *testing.T) {
	// 64 caracteres hex = 32 bytes
	privateKeyHex := "c9f2a4a4c7d2b8e1a6f5e4d3c2b1a0908070605040302010f1e2d3c4b5a6978e"

	wallet1, err := NewWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		t.Fatalf("Failed to create wallet 1: %v", err)
	}

	wallet2, err := NewWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		t.Fatalf("Failed to create wallet 2: %v", err)
	}

	// Mesma chave privada deve gerar mesmo endereço
	if wallet1.GetAddress() != wallet2.GetAddress() {
		t.Error("Same private key should generate same address")
	}

	// Mesma chave privada deve gerar mesma chave pública
	if wallet1.GetPublicKeyHex() != wallet2.GetPublicKeyHex() {
		t.Error("Same private key should generate same public key")
	}
}

func TestMultipleSignatures(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	data := []byte("Test data for multiple signatures")

	// Assina múltiplas vezes
	signature1, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data (1): %v", err)
	}

	signature2, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data (2): %v", err)
	}

	// Devido à aleatoriedade no ECDSA, as assinaturas devem ser diferentes
	if signature1 == signature2 {
		t.Log("Warning: Signatures are the same (this is very unlikely but possible)")
	}

	// Mas ambas devem ser válidas
	valid1, err := Verify(wallet.GetPublicKeyHex(), data, signature1)
	if err != nil || !valid1 {
		t.Error("First signature is invalid")
	}

	valid2, err := Verify(wallet.GetPublicKeyHex(), data, signature2)
	if err != nil || !valid2 {
		t.Error("Second signature is invalid")
	}
}

func TestHashConsistency(t *testing.T) {
	publicKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	address1, err := AddressFromPublicKey(publicKey)
	if err != nil {
		t.Fatalf("Failed to get address (1): %v", err)
	}

	address2, err := AddressFromPublicKey(publicKey)
	if err != nil {
		t.Fatalf("Failed to get address (2): %v", err)
	}

	// Mesma chave pública deve sempre gerar mesmo endereço
	if address1 != address2 {
		t.Error("Address should be deterministic")
	}

	// Verifica que o endereço é um hash SHA-256 válido
	if len(address1) != 64 {
		t.Errorf("Address should be 64 characters (SHA-256), got %d", len(address1))
	}
}

func BenchmarkNewWallet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewWallet()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSign(b *testing.B) {
	wallet, _ := NewWallet()
	data := []byte("Benchmark data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := wallet.Sign(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	wallet, _ := NewWallet()
	data := []byte("Benchmark data")
	signature, _ := wallet.Sign(data)
	publicKeyHex := wallet.GetPublicKeyHex()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Verify(publicKeyHex, data, signature)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAddress(b *testing.B) {
	wallet, _ := NewWallet()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wallet.GetAddress()
	}
}

func BenchmarkSHA256(b *testing.B) {
	data := []byte("Benchmark SHA-256 hashing performance")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sha256.Sum256(data)
	}
}
