package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// Wallet representa uma carteira com par de chaves ECDSA
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// NewWallet cria uma nova carteira com par de chaves ECDSA
func NewWallet() (*Wallet, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// NewWalletFromPrivateKey cria uma carteira a partir de uma chave privada existente
func NewWalletFromPrivateKey(privateKeyHex string) (*Wallet, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key hex: %w", err)
	}

	privateKey := new(ecdsa.PrivateKey)
	curve := elliptic.P256()
	privateKey.Curve = curve
	privateKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privateKey.X, privateKey.Y = curve.ScalarBaseMult(privateKeyBytes)

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// GetPrivateKeyHex retorna a chave privada em formato hexadecimal
func (w *Wallet) GetPrivateKeyHex() string {
	return hex.EncodeToString(w.PrivateKey.D.Bytes())
}

// GetPublicKeyHex retorna a chave pública em formato hexadecimal (concatenação de X e Y)
func (w *Wallet) GetPublicKeyHex() string {
	// Garante que X e Y tenham exatamente 32 bytes cada (padding com zeros à esquerda)
	xBytes := w.PublicKey.X.Bytes()
	yBytes := w.PublicKey.Y.Bytes()

	pubKeyBytes := make([]byte, 64)
	copy(pubKeyBytes[32-len(xBytes):32], xBytes)
	copy(pubKeyBytes[64-len(yBytes):64], yBytes)

	return hex.EncodeToString(pubKeyBytes)
}

// GetAddress retorna o endereço da carteira (hash da chave pública)
func (w *Wallet) GetAddress() string {
	// Usa o GetPublicKeyHex para garantir consistência no formato
	publicKeyHex := w.GetPublicKeyHex()
	publicKeyBytes, _ := hex.DecodeString(publicKeyHex)
	hash := sha256.Sum256(publicKeyBytes)
	return hex.EncodeToString(hash[:])
}

// Sign assina dados usando a chave privada
func (w *Wallet) Sign(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign data: %w", err)
	}

	// Garante que r e s tenham exatamente 32 bytes (padding com zeros à esquerda)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	signature := make([]byte, 64)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	return hex.EncodeToString(signature), nil
}

// Verify verifica uma assinatura usando uma chave pública
func Verify(publicKeyHex string, data []byte, signatureHex string) (bool, error) {
	// Decodifica a chave pública
	publicKey, err := PublicKeyFromHex(publicKeyHex)
	if err != nil {
		return false, err
	}

	// Decodifica a assinatura
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("invalid signature hex: %w", err)
	}

	// Divide a assinatura em r e s
	if len(signatureBytes) != 64 {
		return false, fmt.Errorf("invalid signature length: expected 64 bytes, got %d", len(signatureBytes))
	}

	r := new(big.Int).SetBytes(signatureBytes[:32])
	s := new(big.Int).SetBytes(signatureBytes[32:])

	// Calcula o hash dos dados
	hash := sha256.Sum256(data)

	// Verifica a assinatura
	valid := ecdsa.Verify(publicKey, hash[:], r, s)
	return valid, nil
}

// PublicKeyFromHex converte uma chave pública hexadecimal para *ecdsa.PublicKey
func PublicKeyFromHex(publicKeyHex string) (*ecdsa.PublicKey, error) {
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid public key hex: %w", err)
	}

	if len(publicKeyBytes) != 64 {
		return nil, fmt.Errorf("invalid public key length: expected 64 bytes, got %d", len(publicKeyBytes))
	}

	x := new(big.Int).SetBytes(publicKeyBytes[:32])
	y := new(big.Int).SetBytes(publicKeyBytes[32:])

	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	return publicKey, nil
}

// AddressFromPublicKey calcula o endereço a partir de uma chave pública
func AddressFromPublicKey(publicKeyHex string) (string, error) {
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid public key hex: %w", err)
	}

	hash := sha256.Sum256(publicKeyBytes)
	return hex.EncodeToString(hash[:]), nil
}
