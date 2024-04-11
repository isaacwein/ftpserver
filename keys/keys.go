package keys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
)

// GeneratesRSAKeys generates a new RSA key pair and returns the private and public keys in PEM format.
func GeneratesRSAKeys(bitSize int) (privateKeyFile, publicKeyFile []byte) {

	// Safeguard: Only allow certain key sizes.
	validBitSizes := map[int]bool{2048: true, 3072: true, 4096: true}
	if !validBitSizes[bitSize] {
		return
	}

	// Generate RSA Key with the specified bit size.
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		log.Fatal("GeneratesRSAKeys", bitSize, "GenerateKey error", err.Error())
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
		log.Fatal("GeneratesRSAKeys", bitSize, "MarshalPKIXPublicKey error", err.Error())
		return
	}

	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyDER,
	}

	publicKeyFile = pem.EncodeToMemory(publicKeyPEM)

	return privateKeyFile, publicKeyFile
}

// GeneratesECDSAKeys generates a new ECDSA key pair and returns the private and public keys in PEM format.
func GeneratesECDSAKeys(bitSize int) (privateKeyFile, publicKeyFile []byte) {
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
		log.Fatal("GeneratesECDSAKeys", bitSize, "Invalid bit size")
		return
	}

	// Generate an ECDSA key.
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Fatal("GeneratesECDSAKeys", bitSize, "GenerateKey error", err.Error())
		return
	}

	// Convert the private key to PEM format.
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		log.Fatal("GeneratesECDSAKeys", bitSize, "MarshalECPrivateKey error", err.Error())
		return
	}

	privateKeyPEM := &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}

	// Write the key to a buffer.
	privateKeyFile = pem.EncodeToMemory(privateKeyPEM)

	// Now generate and write the public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatal("GeneratesECDSAKeys", bitSize, "MarshalPKIXPublicKey error", err.Error())
		return
	}

	publicKeyPEM := &pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}

	publicKeyFile = pem.EncodeToMemory(publicKeyPEM)

	return
}

// GeneratesED25519Keys generates a new EdDSA key pair and returns the private and public keys in PEM format.
func GeneratesED25519Keys() (privateKeyFile, publicKeyFile []byte) {
	// Generate an Ed25519 key.
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal("GeneratesED25519Keys", "GenerateKey error", err.Error())
		return
	}

	// Convert the private key to PEM format.
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatal("GeneratesED25519Keys", "MarshalPKCS8PrivateKey error", err.Error())
		return
	}

	privateKeyPEM := &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}

	// Write the key to a buffer.
	privateKeyFile = pem.EncodeToMemory(privateKeyPEM)

	// Now generate and write the public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatal("GeneratesED25519Keys", "MarshalPKIXPublicKey error", err.Error())
		return
	}

	publicKeyPEM := &pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}
	publicKeyFile = pem.EncodeToMemory(publicKeyPEM)
	return
}
