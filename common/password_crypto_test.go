package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPasswordEncryptionRoundTripAndKeyBinding(t *testing.T) {
	privatePEM, err := GeneratePasswordEncryptionPrivateKey()
	require.NoError(t, err)
	require.NoError(t, LoadPasswordEncryptionPrivateKey(privatePEM))

	keyID, publicPEM := PasswordEncryptionPublicKey()
	require.NotEmpty(t, keyID)
	require.NotEmpty(t, publicPEM)
	block, _ := pem.Decode([]byte(publicPEM))
	require.NotNil(t, block)
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	publicKey, ok := parsed.(*rsa.PublicKey)
	require.True(t, ok)
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, []byte("secret"), nil)
	require.NoError(t, err)
	plaintext, err := DecryptPassword(base64.StdEncoding.EncodeToString(ciphertext), keyID)
	require.NoError(t, err)
	require.Equal(t, "secret", plaintext)

	_, err = DecryptPassword(base64.StdEncoding.EncodeToString(ciphertext), "wrong-key")
	require.ErrorIs(t, err, ErrPasswordEncryptionInvalid)
}
