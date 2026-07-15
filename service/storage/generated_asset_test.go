package storage

import (
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

func TestImageBedObjectKeyIncludesConfiguredRoot(t *testing.T) {
	config := map[string]any{"upload_folder": "creative-root/"}
	assert.Equal(t, "creative-root/2026/07/15/7/image.png", imageBedObjectKey(config, "2026/07/15/7/image.png"))
	assert.Equal(t, "2026/07/15/7/image.png", imageBedObjectKey(map[string]any{}, "2026/07/15/7/image.png"))
}

func TestImageBedChunkCount(t *testing.T) {
	count, err := imageBedChunkCount(17, 8)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	_, err = imageBedChunkCount(17, 0)
	require.Error(t, err)
}
