package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
