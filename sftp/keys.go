package sftp

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// GeneratesRSAKeys generates a new RSA key pair and returns the private and public keys in PEM format.
func GeneratesRSAKeys(bitSize int) (privateKeyFile, publicKeyFile []byte, err error) {

	// Safeguard: Only allow certain key sizes.
	validBitSizes := map[int]bool{2048: true, 3072: true, 4096: true}
	if !validBitSizes[bitSize] {
		return nil, nil, fmt.Errorf("invalid bit size: %d", bitSize)
	}

	// Generate RSA Key with the specified bit size.
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		err = fmt.Errorf("error generating RSA private key: %w", err)
		return
	}

	// Convert the private key to PEM format.
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write the private key to a buffer.
	privateKeyFile = pem.EncodeToMemory(privateKeyPEM)

	// Generate and write the public key.
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		err = fmt.Errorf("error marshaling RSA public key: %w", err)
		return
	}

	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyDER,
	}

	publicKeyFile = pem.EncodeToMemory(publicKeyPEM)

	return privateKeyFile, publicKeyFile, nil
}

// GeneratesECDSAKeys generates a new ECDSA key pair and returns the private and public keys in PEM format.
func GeneratesECDSAKeys(bitSize int) (privateKeyFile, publicKeyFile []byte, err error) {
	var curve elliptic.Curve

	// Select curve based on bit size
	switch bitSize {
	case 224:
		curve = elliptic.P224()
	case 256:
		curve = elliptic.P256()
	case 384:
		curve = elliptic.P384()
	case 521:
		curve = elliptic.P521()
	default:
		err = fmt.Errorf("unsupported bitsize")
		return
	}

	// Generate an ECDSA key.
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		err = fmt.Errorf("error generating ECDSA private key: %w", err)
		return
	}

	// Convert the private key to PEM format.
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		err = fmt.Errorf("error marshaling ECDSA private key: %w", err)
		return
	}

	privateKeyPEM := &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}

	// Write the key to a buffer.
	privateKeyFile = pem.EncodeToMemory(privateKeyPEM)

	// Now generate and write the public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		err = fmt.Errorf("error marshaling ECDSA public key: %w", err)
		return
	}

	publicKeyPEM := &pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}

	publicKeyFile = pem.EncodeToMemory(publicKeyPEM)

	return
}

// GeneratesEdDSAKeys generates a new EdDSA key pair and returns the private and public keys in PEM format.
func GeneratesEdDSAKeys() (privateKeyFile, publicKeyFile []byte, err error) {
	// Generate an Ed25519 key.
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		err = fmt.Errorf("error generating EdDSA private key: %w", err)
		return
	}

	// Convert the private key to PEM format.
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		err = fmt.Errorf("error marshaling EdDSA private key: %w", err)
		return
	}

	privateKeyPEM := &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}

	// Write the key to a buffer.
	privateKeyFile = pem.EncodeToMemory(privateKeyPEM)

	// Now generate and write the public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		err = fmt.Errorf("error marshaling EdDSA public key: %w", err)
		return
	}

	publicKeyPEM := &pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}
	publicKeyFile = pem.EncodeToMemory(publicKeyPEM)
	return
}
