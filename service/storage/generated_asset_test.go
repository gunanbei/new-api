package storage

import (
	"bytes"
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedAssetMagicValidation(t *testing.T) {
	assert.True(t, generatedMagicMatches("image/png", []byte("\x89PNG\r\n\x1a\nbody")))
	assert.True(t, generatedMagicMatches("image/svg+xml", []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>")))
	assert.False(t, generatedMagicMatches("image/png", []byte("not an image")))
	assert.False(t, validGeneratedMime("text/html"))
	assert.Equal(t, "png", generatedAssetSuffix("image/png"))
	assert.Equal(t, "jpg", generatedAssetSuffix("image/jpeg"))
}

func TestGeneratedImageBedRelativeURLUsesStorageOrigin(t *testing.T) {
	base, err := url.Parse("https://storage.example/api")
	relative, err := url.Parse("/file/example.png")
	require.NoError(t, err)
	assert.Equal(t, "https://storage.example/file/example.png", base.ResolveReference(relative).String())
}

func TestGeneratedAssetReaderRejectsBytesPastLimit(t *testing.T) {
	reader := &generatedLimitReader{reader: bytes.NewReader([]byte("abc")), remaining: 2}
	data, err := io.ReadAll(reader)
	require.Error(t, err)
	assert.True(t, reader.exceeded)
	assert.Equal(t, []byte("ab"), data)
}
