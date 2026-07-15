package storage

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreativeManifestRejectsInvalidItemsBeforeAssetWrites(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-manifest-invalid?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.Option{}, &model.FileUploadChannel{}, &model.File{}, &model.UserFile{}, &model.CreativeTask{}, &model.CreativeTaskAsset{}))
	require.NoError(t, database.Create(&model.FileUploadChannel{Id: 1, Type: service.FileUploadChannelTypeS3, Status: service.FileUploadChannelStatusEnabled, ConfigProflle: `{}`}).Error)
	require.NoError(t, database.Create(&model.Option{Key: model.CreativeStudioSettingsOption, Value: `{"version":2,"default_file_channel_id":1}`}).Error)
	task := model.CreativeTask{TaskKey: "ct_invalid_manifest", UserID: 7, Category: "image", Status: model.CreativeTaskStatusImporting, RequestedParams: "{}", ResolvedParams: "{}", ResultManifest: `{"items":[{"b64_json":"not-base64","mime_type":"image/png","file_name":"output.png"}]}`}
	require.NoError(t, database.Create(&task).Error)

	err = ImportCreativeTaskManifest(context.Background(), &task)
	require.Error(t, err)
	var count int64
	require.NoError(t, database.Model(&model.CreativeTaskAsset{}).Count(&count).Error)
	assert.Zero(t, count)
}
