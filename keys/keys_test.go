package keys

import (
	"fmt"
	"testing"
)

func Test_GeneratesRSAKeys(t *testing.T) {
	tests := []struct {
		keySize int
	}{
		{2048},
		{3072},
		{4096},
	}

	for _, tt := range tests {
		t.Run("RSAKeySize"+fmt.Sprintf("%d", tt.keySize), func(t *testing.T) {
			privateKey, publicKey := GeneratesRSAKeys(tt.keySize)
			t.Logf("privateKey: %s\n", string(privateKey))
			t.Logf("publicKey: %s\n", string(publicKey))
		})
	}
}

func Test_GeneratesECDSAKeys(t *testing.T) {
	tests := []struct {
		keySize int
	}{
		{224},
		{256},
		{384},
		{521},
	}

	for _, tt := range tests {
		t.Run("ECDSAKeySize"+fmt.Sprintf("%d", tt.keySize), func(t *testing.T) {
			privateKey, publicKey := GeneratesECDSAKeys(tt.keySize)
			t.Logf("privateKey: %s\n", string(privateKey))
			t.Logf("publicKey: %s\n", string(publicKey))
		})
	}
}

func Test_GeneratesED25519Keys(t *testing.T) {
	privateKey, publicKey := GeneratesED25519Keys()

	t.Logf("privateKey: %s\n", string(privateKey))
	t.Logf("publicKey: %s\n", string(publicKey))
}
