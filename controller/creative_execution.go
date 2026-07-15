package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/storage"
	"github.com/QuantumNous/new-api/types"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

type creativePreview struct {
	Base64   string
	MimeType string
}

var creativePreviews sync.Map

func executeCreativeImageTask(parent *gin.Context, task *model.CreativeTask, references []*multipart.FileHeader) error {
	if task.Protocol != "openai_image" && task.Protocol != "advanced_custom_image" {
		return errors.New("creative protocol execution is not available")
	}
	candidateChannelIDs, err := service.CreativeTaskChannelCandidates(task)
	if err != nil {
		return err
	}
	var params map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(task.ResolvedParams, &params); err != nil {
		return err
	}
	modelRaw, err := common.Marshal(task.ModelName)
	if err != nil {
		return err
	}
	params["model"] = modelRaw
	delete(params, "stream")
	delete(params, "download_resolution")
	body, err := common.Marshal(params)
	if err != nil {
		return err
	}
	user, err := model.GetUserById(int(task.UserID), true)
	if err != nil {
		return err
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	requestPath := "/v1/images/generations"
	contentType := "application/json"
	var requestBody io.Reader = bytes.NewReader(body)
	if task.Operation == "edit" {
		pipeReader, pipeWriter := io.Pipe()
		writer := multipart.NewWriter(pipeWriter)
		go func() {
			defer pipeWriter.Close()
			defer writer.Close()
			for key, value := range params {
				if key == "image" || key == "images" || key == "stream" {
					continue
				}
				_ = writer.WriteField(key, strings.Trim(string(value), "\""))
			}
			if len(references) > 0 {
				for _, reference := range references {
					input, err := reference.Open()
					if err != nil {
						_ = pipeWriter.CloseWithError(err)
						return
					}
					part, err := writer.CreateFormFile("image", reference.Filename)
					if err == nil {
						_, err = io.Copy(part, input)
					}
					_ = input.Close()
					if err != nil {
						_ = pipeWriter.CloseWithError(err)
						return
					}
				}
				return
			}
			var inputAssets []model.CreativeTaskAsset
			if err := model.DB.Where("task_id = ? AND role = ?", task.ID, "input").Order("position asc").Find(&inputAssets).Error; err != nil || len(inputAssets) == 0 {
				_ = pipeWriter.CloseWithError(errors.New("image edit requires a creative input file"))
				return
			}
			for _, inputAsset := range inputAssets {
				input, inputView, err := storage.OpenCreativeUserFile(parent.Request.Context(), task.UserID, inputAsset.UserFileID)
				if err != nil {
					_ = pipeWriter.CloseWithError(err)
					return
				}
				part, err := writer.CreateFormFile("image", inputView.FileName)
				if err == nil {
					_, err = io.Copy(part, input)
				}
				input.Close()
				if err != nil {
					_ = pipeWriter.CloseWithError(err)
					return
				}
			}
		}()
		requestPath = "/v1/images/edits"
		contentType = writer.FormDataContentType()
		requestBody = pipeReader
		defer pipeReader.Close()
	}
	request, err := http.NewRequestWithContext(parent.Request.Context(), http.MethodPost, requestPath, requestBody)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", contentType)
	context.Request = request
	if err := setupCreativeRelayContext(context, user, task); err != nil {
		return err
	}
	context.Set(common.RequestIdKey, task.RequestID)
	context.Set("creative_candidate_channel_ids", candidateChannelIDs)
	Relay(context, types.RelayFormatOpenAIImage)
	if recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices {
		return fmt.Errorf("creative relay failed")
	}
	var response dto.ImageResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return errors.New("image relay returned no assets")
	}
	items := make([]map[string]string, 0, len(response.Data))
	for _, item := range response.Data {
		if item.B64Json != "" {
			decoded, err := base64.StdEncoding.DecodeString(item.B64Json)
			if err != nil {
				return err
			}
			mimeType := http.DetectContentType(decoded)
			if !isCreativeRasterMIME(mimeType) {
				return errors.New("image relay returned an unsupported asset type")
			}
			items = append(items, map[string]string{"b64_json": item.B64Json, "mime_type": mimeType, "file_name": "creative-image"})
			continue
		}
		if item.Url != "" {
			items = append(items, map[string]string{"url": item.Url, "file_name": "creative-image"})
		}
	}
	if len(items) == 0 {
		return errors.New("image relay returned no importable assets")
	}
	return importCreativeResult(task, context, items)
}

func isCreativeRasterMIME(mimeType string) bool {
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	}
	return false
}

func executeCreativeResponsesImageTask(parent *gin.Context, task *model.CreativeTask) error {
	var params map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(task.ResolvedParams, &params); err != nil {
		return err
	}
	modelRaw, err := common.Marshal(task.ModelName)
	if err != nil {
		return err
	}
	params["model"] = modelRaw
	delete(params, "stream")
	delete(params, "download_resolution")
	body, err := common.Marshal(params)
	if err != nil {
		return err
	}
	responseBody, context, err := executeCreativeRelay(parent, task, http.MethodPost, "/v1/responses", body, types.RelayFormatOpenAIResponses)
	if err != nil {
		return err
	}
	var response struct {
		Output []struct {
			Type   string `json:"type"`
			Result string `json:"result"`
		} `json:"output"`
	}
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return err
	}
	items := make([]map[string]string, 0, len(response.Output))
	for _, output := range response.Output {
		if output.Type != dto.ResponsesOutputTypeImageGenerationCall || strings.TrimSpace(output.Result) == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(output.Result)
		if err != nil {
			return errors.New("Responses image result is not valid base64")
		}
		items = append(items, map[string]string{"b64_json": output.Result, "mime_type": http.DetectContentType(decoded), "file_name": "creative-image"})
	}
	return importCreativeResult(task, context, items)
}

func executeCreativeSVGTask(parent *gin.Context, task *model.CreativeTask) error {
	var params map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(task.ResolvedParams, &params); err != nil {
		return err
	}
	promptRaw, ok := params["prompt"]
	if !ok {
		return errors.New("SVG prompt is required")
	}
	var prompt string
	if err := common.Unmarshal(promptRaw, &prompt); err != nil || strings.TrimSpace(prompt) == "" {
		return errors.New("SVG prompt is required")
	}
	aspectRatio := "1:1"
	if raw, ok := params["aspect_ratio"]; ok {
		if err := common.Unmarshal(raw, &aspectRatio); err != nil {
			return errors.New("SVG aspect ratio is invalid")
		}
	}
	count := uint(1)
	if raw, ok := params["n"]; ok {
		if err := common.Unmarshal(raw, &count); err != nil || count == 0 || count > 4 {
			return errors.New("SVG quantity is invalid")
		}
	}
	settings, err := service.GetCreativeStudioSettings()
	if err != nil {
		return err
	}
	items := make([]map[string]string, 0, count)
	var relayContext *gin.Context
	for index := uint(0); index < count; index++ {
		body, err := common.Marshal(map[string]any{
			"model":      task.ModelName,
			"max_tokens": 4096,
			"stream":     false,
			"system":     fmt.Sprintf("Return exactly one complete SVG document and no other text or Markdown. Use a %s canvas aspect ratio and a matching viewBox.", aspectRatio),
			"messages":   []map[string]string{{"role": "user", "content": prompt}},
		})
		if err != nil {
			return err
		}
		responseBody, context, err := executeCreativeRelay(parent, task, http.MethodPost, "/v1/messages", body, types.RelayFormatClaude)
		if err != nil {
			return err
		}
		relayContext = context
		var response dto.ClaudeResponse
		if err := common.Unmarshal(responseBody, &response); err != nil {
			return err
		}
		var output strings.Builder
		for _, content := range response.Content {
			if content.Type == "text" && content.Text != nil {
				output.WriteString(*content.Text)
			}
		}
		svg, err := service.SanitizeCreativeSVG([]byte(output.String()), settings.MaxSVGBytes)
		if err != nil {
			return err
		}
		items = append(items, map[string]string{"b64_json": base64.StdEncoding.EncodeToString(svg), "mime_type": "image/svg+xml", "file_name": fmt.Sprintf("creative-%d.svg", index+1)})
	}
	return importCreativeResult(task, relayContext, items)
}

func executeCreativeMidjourneyTask(parent *gin.Context, task *model.CreativeTask) error {
	var params map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(task.ResolvedParams, &params); err != nil {
		return err
	}
	promptRaw, ok := params["prompt"]
	if !ok {
		return errors.New("Midjourney prompt is required")
	}
	var prompt string
	if err := common.Unmarshal(promptRaw, &prompt); err != nil || strings.TrimSpace(prompt) == "" {
		return errors.New("Midjourney prompt is required")
	}
	if raw, ok := params["aspect_ratio"]; ok {
		var aspectRatio string
		if err := common.Unmarshal(raw, &aspectRatio); err != nil {
			return errors.New("Midjourney aspect ratio is invalid")
		}
		prompt = withMidjourneyAspectRatio(prompt, aspectRatio)
	}
	body, err := common.Marshal(map[string]string{"prompt": prompt})
	if err != nil {
		return err
	}
	candidateChannelIDs, err := service.CreativeTaskChannelCandidates(task)
	if err != nil {
		return err
	}
	user, err := model.GetUserById(int(task.UserID), true)
	if err != nil {
		return err
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request, err := http.NewRequestWithContext(parent.Request.Context(), http.MethodPost, "/mj/submit/imagine", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	context.Request = request
	if err := setupCreativeRelayContext(context, user, task); err != nil {
		return err
	}
	context.Set(common.RequestIdKey, task.RequestID)
	context.Set("creative_candidate_channel_ids", candidateChannelIDs)
	relayInfo, err := relaycommon.GenRelayInfo(context, types.RelayFormatMjProxy, nil, nil)
	if err != nil {
		return err
	}
	retry := 0
	channel, channelErr := getChannel(context, relayInfo, &service.RetryParam{Ctx: context, TokenGroup: relayInfo.TokenGroup, ModelName: task.ModelName, RequestPath: request.URL.Path, Retry: &retry})
	if channelErr != nil {
		return channelErr
	}
	response := relay.RelayMidjourneySubmit(context, relayInfo)
	if response != nil {
		return errors.New("Midjourney submission failed")
	}
	var result dto.MidjourneyResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		return err
	}
	if result.Code != 1 || strings.TrimSpace(result.Result) == "" {
		return errors.New("Midjourney submission failed")
	}
	transitioned, err := service.TransitionCreativeTask(task.ID, []string{model.CreativeTaskStatusDispatching}, model.CreativeTaskStatusProcessing, map[string]any{"external_task_id": result.Result, "channel_id": channel.Id})
	if err != nil || !transitioned {
		return errors.New("creative task state changed unexpectedly")
	}
	task.Status, task.ExternalTaskID, task.ChannelID = model.CreativeTaskStatusProcessing, result.Result, channel.Id
	return nil
}

func withMidjourneyAspectRatio(prompt, aspectRatio string) string {
	fields := strings.Fields(prompt)
	for index, field := range fields {
		if (field == "--ar" && index+1 < len(fields)) || strings.HasPrefix(field, "--ar=") {
			return prompt
		}
	}
	return prompt + " --ar " + aspectRatio
}

func executeCreativeRelay(parent *gin.Context, task *model.CreativeTask, method, requestPath string, body []byte, format types.RelayFormat) ([]byte, *gin.Context, error) {
	candidateChannelIDs, err := service.CreativeTaskChannelCandidates(task)
	if err != nil {
		return nil, nil, err
	}
	user, err := model.GetUserById(int(task.UserID), true)
	if err != nil {
		return nil, nil, err
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request, err := http.NewRequestWithContext(parent.Request.Context(), method, requestPath, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	context.Request = request
	if err := setupCreativeRelayContext(context, user, task); err != nil {
		return nil, nil, err
	}
	context.Set(common.RequestIdKey, task.RequestID)
	context.Set("creative_candidate_channel_ids", candidateChannelIDs)
	Relay(context, format)
	if recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices {
		return nil, nil, creativeRelayError(recorder.Code, recorder.Body.Bytes())
	}
	return recorder.Body.Bytes(), context, nil
}

func creativeRelayError(statusCode int, body []byte) error {
	var response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if common.Unmarshal(body, &response) == nil && strings.TrimSpace(response.Error.Message) != "" {
		return errors.New(response.Error.Message)
	}
	return fmt.Errorf("creative relay failed (HTTP %d)", statusCode)
}

func setupCreativeRelayContext(c *gin.Context, user *model.User, task *model.CreativeTask) error {
	user.ToBaseUser().WriteContext(c)
	common.SetContextKey(c, constant.ContextKeyUserId, user.Id)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, task.GroupName)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, task.ModelName)
	common.SetContextKey(c, constant.ContextKeyTokenGroup, task.GroupName)
	common.SetContextKey(c, constant.ContextKeyCreativeTask, true)
	return nil
}

func importCreativeResult(task *model.CreativeTask, relayContext *gin.Context, items []map[string]string) error {
	if len(items) == 0 {
		return errors.New("creative relay returned no importable assets")
	}
	persisted := make([]map[string]string, 0, len(items))
	for _, item := range items {
		if item["url"] != "" {
			persisted = append(persisted, item)
		}
	}
	updates := map[string]any{"result_manifest": "", "import_expires_at": nil}
	if len(persisted) > 0 {
		manifest, err := common.Marshal(map[string]any{"items": persisted})
		if err != nil {
			return err
		}
		importExpiresAt := time.Now().Add(24 * time.Hour)
		// ponytail: URL retry manifest is capped below MySQL TEXT; add a separate result table if oversized URL batches need retry.
		if len(manifest) <= 60_000 {
			updates["result_manifest"] = string(manifest)
			updates["import_expires_at"] = importExpiresAt
			task.ResultManifest, task.ImportExpiresAt = string(manifest), &importExpiresAt
		}
	}
	if selectedChannelID, ok := relayContext.Get("creative_selected_channel_id"); ok {
		if channelID, valid := selectedChannelID.(int); valid {
			updates["channel_id"] = channelID
			task.ChannelID = channelID
		}
	}
	updates["status"] = model.CreativeTaskStatusImporting
	if err := model.DB.Model(&model.CreativeTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
		return err
	}
	task.Status = model.CreativeTaskStatusImporting
	taskCopy := *task
	previews := make([]creativePreview, 0, len(items))
	for _, item := range items {
		if item["b64_json"] != "" {
			previews = append(previews, creativePreview{Base64: item["b64_json"], MimeType: item["mime_type"]})
		}
	}
	if len(previews) > 0 {
		creativePreviews.Store(task.ID, previews)
	}
	gopool.Go(func() {
		defer creativePreviews.Delete(taskCopy.ID)
		if err := storage.ImportCreativeTaskResults(context.Background(), &taskCopy, items); err != nil {
			service.MarkCreativeTaskImportFailed(taskCopy.ID, err)
		}
	})
	return nil
}
