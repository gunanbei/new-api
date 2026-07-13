package storage

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestListFiltersStatsAndLogicalDelete(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousDB, previousRedis := model.DB, common.RedisEnabled
	model.DB, common.RedisEnabled = db, false
	t.Cleanup(func() { model.DB, common.RedisEnabled = previousDB, previousRedis })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.FileUploadChannel{}, &model.File{}, &model.UserFile{}))

	require.NoError(t, db.Create(&model.User{Id: 7, Username: "alice", Password: "password", AffCode: "storage-aff-a", Status: 1}).Error)
	require.NoError(t, db.Create(&model.User{Id: 8, Username: "bob", Password: "password", AffCode: "storage-aff-b", Status: 1}).Error)
	channel := model.FileUploadChannel{Id: 1, Name: "S3", Type: "3", Status: "1", IsDefault: "1", ConfigProflle: `{}`}
	require.NoError(t, db.Create(&channel).Error)
	file := model.File{Id: 11, FileChannelId: 1, ChannelType: "3", FileSize: 1024, ObjectKey: "files/a.png", Identifier: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RefCount: 2, Status: "1"}
	second := model.File{Id: 12, FileChannelId: 1, ChannelType: "3", FileSize: 2048, ObjectKey: "files/b.jpg", Identifier: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RefCount: 1, Status: "0"}
	require.NoError(t, db.Create(&file).Error)
	require.NoError(t, db.Create(&second).Error)
	rows := []model.UserFile{
		{Id: 21, FileId: 11, FileChannelId: 1, UserId: 7, FileName: "alpha", FileSuffix: "png", Status: "1"},
		{Id: 22, FileId: 12, FileChannelId: 1, UserId: 7, FileName: "beta", FileSuffix: "jpg", Status: "0"},
		{Id: 23, FileId: 11, FileChannelId: 1, UserId: 8, FileName: "shared", FileSuffix: "png", Status: "1"},
	}
	require.NoError(t, db.Create(&rows).Error)

	userID := int64(7)
	items, total, err := List(ListFilter{UserID: &userID}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2, "the default list includes uploading and available logical files")
	items, total, err = List(ListFilter{UserID: &userID, FileSuffixes: []string{"png"}, Status: "1"}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, uint64(21), items[0].ID)

	stats, err := StatsByChannel("3", "")
	require.NoError(t, err)
	require.Len(t, stats.Items, 1)
	assert.Equal(t, int64(3), stats.Items[0].UserFileCount)
	assert.Equal(t, int64(2), stats.Items[0].FileCount)
	assert.Equal(t, int64(3072), stats.Items[0].TotalSize)

	_, err = Delete(context.Background(), 21, &userID)
	require.NoError(t, err)
	var after model.File
	require.NoError(t, db.First(&after, 11).Error)
	assert.Equal(t, 1, after.RefCount, "deleting one logical reference must not delete a shared physical object")
	var remaining int64
	require.NoError(t, db.Model(&model.UserFile{}).Where("file_id = ?", 11).Count(&remaining).Error)
	assert.Equal(t, int64(1), remaining)
}
