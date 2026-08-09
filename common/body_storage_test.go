package common

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBodyStorageNewReaderHasIndependentCursor(t *testing.T) {
	storage, err := CreateBodyStorage([]byte("payload"))
	require.NoError(t, err)
	defer storage.Close()

	buf := make([]byte, 1)
	_, err = storage.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte("p"), buf)

	replay, err := storage.NewReader()
	require.NoError(t, err)
	defer replay.Close()

	data, err := io.ReadAll(replay)
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), data)
}
