package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreativeTaskTransitionUsesCompareAndSwap(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-task-cas?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.CreativeTask{}, &model.CreativeTaskAsset{}))
	task := model.CreativeTask{TaskKey: "ct_test", UserID: 7, Category: "image", Status: model.CreativeTaskStatusImporting, RequestedParams: "{}", ResolvedParams: "{}", ResultManifest: ""}
	require.NoError(t, database.Create(&task).Error)

	transitioned, err := TransitionCreativeTask(task.ID, []string{model.CreativeTaskStatusImporting}, model.CreativeTaskStatusSucceeded, map[string]any{})
	require.NoError(t, err)
	assert.True(t, transitioned)
	transitioned, err = TransitionCreativeTask(task.ID, []string{model.CreativeTaskStatusImporting}, model.CreativeTaskStatusSucceeded, map[string]any{})
	require.NoError(t, err)
	assert.True(t, transitioned)
	transitioned, err = TransitionCreativeTask(task.ID, []string{model.CreativeTaskStatusDispatching}, model.CreativeTaskStatusFailed, map[string]any{})
	require.NoError(t, err)
	assert.False(t, transitioned)
}

func TestCreativeImportRetryIsOwnerScopedAndDoesNotDispatch(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-import-retry?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.CreativeTask{}, &model.CreativeTaskAsset{}))
	expiresAt := time.Now().Add(time.Minute)
	task := model.CreativeTask{TaskKey: "ct_import", UserID: 7, Category: "image", Status: model.CreativeTaskStatusFailed, RequestedParams: "{}", ResolvedParams: "{}", ResultManifest: `[{"kind":"b64"}]`, ErrorCode: "asset_import_failed", ImportExpiresAt: &expiresAt}
	require.NoError(t, database.Create(&task).Error)
	_, err = RetryCreativeImport(8, task.TaskKey)
	assert.Error(t, err)
	retried, err := RetryCreativeImport(7, task.TaskKey)
	require.NoError(t, err)
	assert.Equal(t, model.CreativeTaskStatusImporting, retried.Status)
	assert.Empty(t, retried.ErrorCode)
}

func TestDeleteCreativeTaskDeletesOnlyTerminalOwnerTask(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-task-delete?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.CreativeTask{}, &model.CreativeTaskAsset{}))
	completed := model.CreativeTask{TaskKey: "ct_delete", UserID: 7, Category: "image", Status: model.CreativeTaskStatusSucceeded, RequestedParams: "{}", ResolvedParams: "{}"}
	processing := model.CreativeTask{TaskKey: "ct_keep", UserID: 7, Category: "image", Status: model.CreativeTaskStatusProcessing, RequestedParams: "{}", ResolvedParams: "{}"}
	require.NoError(t, database.Create(&completed).Error)
	require.NoError(t, database.Create(&processing).Error)
	require.Error(t, DeleteCreativeTask(8, completed.TaskKey))
	require.Error(t, DeleteCreativeTask(7, processing.TaskKey))
	require.NoError(t, DeleteCreativeTask(7, completed.TaskKey))
	var count int64
	require.NoError(t, database.Model(&model.CreativeTask{}).Where("id = ?", completed.ID).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCreativeTaskRejectsForbiddenBillingInputs(t *testing.T) {
	stream := dto.CreativeTaskCreateRequest{CapabilityID: 1, GroupName: "default", Params: map[string]json.RawMessage{"stream": json.RawMessage("true")}}
	require.EqualError(t, ValidateCreativeTaskRequest(stream), "stream is not supported for creative tasks")
	tooMany := dto.CreativeTaskCreateRequest{CapabilityID: 1, GroupName: "default", Params: map[string]json.RawMessage{"n": json.RawMessage("129")}}
	assert.EqualError(t, ValidateCreativeTaskRequest(tooMany), "n must be an integer between 1 and 128")
	assert.True(t, forbiddenCreativeParameter("channel_id"))
	assert.True(t, forbiddenCreativeParameter("system_prompt"))
	assert.False(t, forbiddenCreativeParameter("prompt"))
}

func TestResolveCreativeInputFilesLimitsReferenceImages(t *testing.T) {
	settings := defaultCreativeStudioSettings()
	_, err := resolveCreativeInputFiles(
		7,
		"image",
		json.RawMessage(`[1,2,3,4,5,6]`),
		map[string]any{"multiple": true, "max_items": float64(5)},
		settings,
	)
	require.EqualError(t, err, "image: accepts between 1 and 5 input images")
}

func TestResolveCreativeTaskParamsUsesDirectReferenceImages(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-direct-references?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.Option{}))

	capability := model.CreativeModelCapability{
		Operation:     "edit",
		InputSchema:   `{"properties":{"prompt":{"type":"textarea","required":true},"image":{"type":"file","required":true,"multiple":true,"max_items":5}}}`,
		DefaultParams: `{}`,
	}
	publication := model.CreativeModelPublication{GroupDefaultParams: `{}`}
	requested, resolved, assets, err := resolveCreativeTaskParams(
		1,
		capability,
		publication,
		map[string]json.RawMessage{"prompt": json.RawMessage(`"a cat"`)},
		2,
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"prompt":"a cat"}`, requested)
	assert.JSONEq(t, `{"prompt":"a cat"}`, resolved)
	assert.Empty(t, assets)

	_, _, _, err = resolveCreativeTaskParams(
		1,
		capability,
		publication,
		map[string]json.RawMessage{"prompt": json.RawMessage(`"a cat"`)},
		6,
	)
	require.EqualError(t, err, "reference images: accepts between 1 and 5 input images")
}

func TestCreativeTaskAddsRequiredPromptWhenSchemaIsEmpty(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-task-default-prompt?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.CreativeModelCapability{}, &model.CreativeModelPublication{}))
	capability := model.CreativeModelCapability{InputSchema: "{}", DefaultParams: "{}"}
	publication := model.CreativeModelPublication{GroupDefaultParams: "{}"}

	_, _, _, err = resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{})
	require.EqualError(t, err, "prompt: is required")
	_, _, _, err = resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{"prompt": json.RawMessage(`" "`)})
	require.EqualError(t, err, "prompt: is required")
	requested, resolved, _, err := resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{"prompt": json.RawMessage(`"a cat"`)})
	require.NoError(t, err)
	assert.JSONEq(t, `{"prompt":"a cat"}`, requested)
	assert.JSONEq(t, `{"prompt":"a cat"}`, resolved)
}

func TestCreativeTaskDefaultImageSpecificationsAreValidated(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-task-default-specs?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	capability := model.CreativeModelCapability{Protocol: "openai_image", InputSchema: "{}", DefaultParams: "{}"}
	publication := model.CreativeModelPublication{GroupDefaultParams: "{}"}

	requested, resolved, _, err := resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{
		"prompt":              json.RawMessage(`"a cat"`),
		"n":                   json.RawMessage(`2`),
		"size":                json.RawMessage(`"1536x1024"`),
		"download_resolution": json.RawMessage(`"4K"`),
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"prompt":"a cat","n":2,"size":"1536x1024","download_resolution":"4K"}`, requested)
	assert.Contains(t, resolved, `"quality":"auto"`)

	_, _, _, err = resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{
		"prompt": json.RawMessage(`"a cat"`),
		"n":      json.RawMessage(`3`),
	})
	require.EqualError(t, err, "n: is not an allowed option")
	_, _, _, err = resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{
		"prompt": json.RawMessage(`"a cat"`),
		"size":   json.RawMessage(`"2048x2048"`),
	})
	require.EqualError(t, err, "size: is not an allowed option")
}

func TestCreativeTaskClaudeSVGAllowsBoundedCustomRatio(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-task-svg-ratio?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	capability := model.CreativeModelCapability{Protocol: "claude_svg", InputSchema: "{}", DefaultParams: "{}"}
	publication := model.CreativeModelPublication{GroupDefaultParams: "{}"}

	_, _, _, err = resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{
		"prompt":       json.RawMessage(`"an icon"`),
		"aspect_ratio": json.RawMessage(`"4:3"`),
	})
	require.NoError(t, err)
	_, _, _, err = resolveCreativeTaskParams(1, capability, publication, map[string]json.RawMessage{
		"prompt":       json.RawMessage(`"an icon"`),
		"aspect_ratio": json.RawMessage(`"1000:1"`),
	})
	require.EqualError(t, err, "aspect_ratio: is not an allowed option")
}

func TestRecoverCreativeTasksFailsOnlyStaleDispatches(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-task-recovery?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.CreativeTask{}))
	stale := model.CreativeTask{TaskKey: "ct_stale", UserID: 7, Category: "image", Status: model.CreativeTaskStatusDispatching, RequestedParams: "{}", ResolvedParams: "{}"}
	current := model.CreativeTask{TaskKey: "ct_current", UserID: 7, Category: "image", Status: model.CreativeTaskStatusDispatching, RequestedParams: "{}", ResolvedParams: "{}"}
	require.NoError(t, database.Create(&stale).Error)
	require.NoError(t, database.Create(&current).Error)
	require.NoError(t, database.Model(&stale).Update("updated_at", time.Now().Add(-11*time.Minute)).Error)
	require.NoError(t, RecoverCreativeTasks())
	require.NoError(t, database.First(&stale, stale.ID).Error)
	require.NoError(t, database.First(&current, current.ID).Error)
	assert.Equal(t, model.CreativeTaskStatusFailed, stale.Status)
	assert.Equal(t, "interrupted_before_result", stale.ErrorCode)
	assert.Equal(t, model.CreativeTaskStatusDispatching, current.Status)
}
