package storage

import (
	"context"
	"strings"
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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.FileUploadChannel{}, &model.File{}, &model.UserFile{}, &model.CreativeTaskAsset{}))

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

func TestBuildObjectKeyKeepsExistingSuffix(t *testing.T) {
	channel := &model.FileUploadChannel{ConfigProflle: `{}`}
	key := buildObjectKey(channel, 7, "creative-image.png", "png")
	assert.Regexp(t, `^\d{4}/\d{2}/\d{2}/7/`, key)
	assert.True(t, strings.HasSuffix(key, "creative-image.png"))
	assert.NotContains(t, key, ".png.png")
}

func TestDeleteRejectsCreativeOutput(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, db.AutoMigrate(&model.FileUploadChannel{}, &model.File{}, &model.UserFile{}, &model.CreativeTaskAsset{}))
	require.NoError(t, db.Create(&model.FileUploadChannel{Id: 1, Name: "S3", Type: "3", Status: "1"}).Error)
	require.NoError(t, db.Create(&model.File{Id: 1, FileChannelId: 1, ChannelType: "3", Identifier: "creative-output", RefCount: 1, Status: "1"}).Error)
	userFile := model.UserFile{Id: 1, FileId: 1, FileChannelId: 1, UserId: 7, FileName: "work.png", Status: "1"}
	require.NoError(t, db.Create(&userFile).Error)
	require.NoError(t, db.Create(&model.CreativeTaskAsset{TaskID: 1, UserFileID: 1, FileID: 1, Role: "output", Position: 0, FileName: "work.png"}).Error)

	userID := int64(7)
	_, err = Delete(context.Background(), 1, &userID)
	require.EqualError(t, err, "creative output files cannot be deleted")
}
