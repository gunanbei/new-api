package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreativeStudioDefaultFileChannelReference(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-studio-settings?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := DB
	DB = database
	t.Cleanup(func() { DB = previousDB })
	require.NoError(t, DB.AutoMigrate(&Option{}))
	raw, err := common.Marshal(map[string]any{"version": 1, "default_file_channel_id": 9})
	require.NoError(t, err)
	require.NoError(t, DB.Create(&Option{Key: CreativeStudioSettingsOption, Value: string(raw)}).Error)

	referenced, err := IsCreativeStudioFileChannel(9)
	require.NoError(t, err)
	assert.True(t, referenced)
	referenced, err = IsCreativeStudioFileChannel(8)
	require.NoError(t, err)
	assert.False(t, referenced)
}
