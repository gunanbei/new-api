package common

import (
	"io"
	"strings"
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

func TestCreateBodyStorageFromReader_ZeroLimitAllowsFullBody(t *testing.T) {
	storage, err := CreateBodyStorageFromReader(strings.NewReader("payload"), -1, 0)
	require.NoError(t, err)
	defer storage.Close()

	data, err := storage.Bytes()
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), data)
}

func TestCreateBodyStorageFromReader_PositiveLimitStillRejectsOversizedBody(t *testing.T) {
	_, err := CreateBodyStorageFromReader(strings.NewReader("payload"), -1, 3)
	require.ErrorIs(t, err, ErrRequestBodyTooLarge)
}
