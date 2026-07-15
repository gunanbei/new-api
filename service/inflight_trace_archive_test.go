package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInflightTraceArchiveStatsCountsUnuploadedArchives(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:inflight-trace-archive-stats?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.InflightTraceArchive{}))
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })

	previousConfig := common.GetDiskCacheConfig()
	directory := t.TempDir()
	common.SetDiskCacheConfig(common.DiskCacheConfig{InflightTracePath: directory})
	t.Cleanup(func() { common.SetDiskCacheConfig(previousConfig) })
	archiveDirectory := filepath.Join(directory, "inflight-traces")
	require.NoError(t, os.MkdirAll(archiveDirectory, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(archiveDirectory, "active.csv"), []byte("active"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(archiveDirectory, "uploaded.csv"), []byte("uploaded"), 0600))
	require.NoError(t, database.Create([]model.InflightTraceArchive{
		{UserID: 1, FileName: "active.csv", Status: inflightTraceArchiveStatusActive},
		{UserID: 1, FileName: "failed.csv", Status: inflightTraceArchiveStatusFailed},
		{UserID: 1, FileName: "uploading.csv", Status: inflightTraceArchiveStatusUploading},
		{UserID: 1, FileName: "uploaded.csv", Status: inflightTraceArchiveStatusUploaded},
	}).Error)

	stats, err := GetInflightTraceArchiveStats()
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.FileCount)
	assert.Equal(t, int64(len("active")+len("uploaded")), stats.TotalSize)
	assert.Equal(t, int64(2), stats.PendingUploadCount)
}

func TestRecoverInflightTraceArchiveUploadsMakesInterruptedArchivesRetryable(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:inflight-trace-archive-recovery?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.InflightTraceArchive{}))
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })

	interrupted := model.InflightTraceArchive{UserID: 1, FileName: "interrupted.csv", LocalPath: "/tmp/interrupted.csv", Status: inflightTraceArchiveStatusUploading}
	active := model.InflightTraceArchive{UserID: 1, FileName: "active.csv", Status: inflightTraceArchiveStatusActive}
	require.NoError(t, database.Create(&interrupted).Error)
	require.NoError(t, database.Create(&active).Error)

	require.NoError(t, RecoverInflightTraceArchiveUploads())
	require.NoError(t, database.First(&interrupted, interrupted.ID).Error)
	assert.Equal(t, inflightTraceArchiveStatusFailed, interrupted.Status)
	assert.Equal(t, "archive upload interrupted; retry required", interrupted.LastError)
	require.NoError(t, database.First(&active, active.ID).Error)
	assert.Equal(t, inflightTraceArchiveStatusActive, active.Status)
}

func TestDeleteInflightTraceArchivesBeforeKeepsUnuploadedCSV(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:inflight-trace-archive-retention?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.InflightTraceArchive{}))
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })

	directory := t.TempDir()
	failedPath := filepath.Join(directory, "failed.csv")
	uploadedPath := filepath.Join(directory, "uploaded.csv")
	require.NoError(t, os.WriteFile(failedPath, []byte("failed"), 0600))
	require.NoError(t, os.WriteFile(uploadedPath, []byte("uploaded"), 0600))
	failed := model.InflightTraceArchive{UserID: 1, FileName: "failed.csv", LocalPath: failedPath, Status: inflightTraceArchiveStatusFailed, LatestRecordedAt: 10}
	uploaded := model.InflightTraceArchive{UserID: 1, FileName: "uploaded.csv", LocalPath: uploadedPath, Status: inflightTraceArchiveStatusUploaded, LatestRecordedAt: 10}
	require.NoError(t, database.Create(&failed).Error)
	require.NoError(t, database.Create(&uploaded).Error)

	deleted, err := DeleteInflightTraceArchivesBefore(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	assert.FileExists(t, failedPath)
	assert.NoFileExists(t, uploadedPath)
	require.NoError(t, database.First(&failed, failed.ID).Error)
	assert.Error(t, database.First(&model.InflightTraceArchive{}, uploaded.ID).Error)
}

func TestDeleteInflightTraceArchiveFileCloudflareImageBed(t *testing.T) {
	type requestInfo struct {
		method        string
		path          string
		source        string
		authorization string
	}
	requests := make(chan requestInfo, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- requestInfo{
			method:        request.Method,
			path:          request.URL.Path,
			source:        request.URL.Query().Get("src"),
			authorization: request.Header.Get("Authorization"),
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	database, err := gorm.Open(sqlite.Open("file:inflight-trace-archive-delete?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.FileUploadChannel{}))
	previousDB := model.DB
	model.DB = database
	defer func() { model.DB = previousDB }()

	channel := model.FileUploadChannel{
		Type:          FileUploadChannelTypeCloudflareImageBed,
		ConfigProflle: `{"base_url":"` + server.URL + `","api_token":"test-token","upload_folder":"archive"}`,
	}
	require.NoError(t, database.Create(&channel).Error)
	require.NoError(t, deleteInflightTraceArchiveFile(context.Background(), channel.Id, "archive/Inflight_1.csv"))
	request := <-requests
	require.Equal(t, http.MethodGet, request.method)
	require.Equal(t, "/api/manage/delete/archive/Inflight_1.csv", request.path)
	require.Empty(t, request.source)
	require.Equal(t, "Bearer test-token", request.authorization)
}
