package service

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreativeBindingUsesDirectoryModelWhenRequestModelIsEmpty(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-binding-model?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.CreativeModel{}, &model.CreativeModelCapability{}, &model.CreativeModelPublication{}))
	creativeModel := model.CreativeModel{ModelName: "gpt-image-2", ModelKey: "gpt-image-2", DisplayName: "GPT Image 2", Status: "enabled"}
	require.NoError(t, database.Create(&creativeModel).Error)
	capability := model.CreativeModelCapability{ModelID: creativeModel.ID, Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_images", ExecutionMode: "sync", InputSchema: "{}", DefaultParams: "{}", Enabled: true}
	require.NoError(t, database.Create(&capability).Error)
	publication := model.CreativeModelPublication{CapabilityID: capability.ID, GroupName: "default", GroupDefaultParams: "{}", Enabled: true}
	require.NoError(t, database.Create(&publication).Error)
	require.NoError(t, database.Create(&model.Channel{Id: 1, Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, database.Create(&model.Ability{Group: "default", Model: "gpt-image-2", ChannelId: 1, Enabled: true}).Error)

	binding := &model.CreativeChannelBinding{PublicationID: publication.ID}
	require.NoError(t, applyCreativeBinding(binding, dto.CreativeBindingRequest{ChannelID: 1}))
	assert.Equal(t, "gpt-image-2", binding.RequestModel)
}

func TestCreativeStudioRejectsDuplicateRecords(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-duplicate-records?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.CreativeModel{}, &model.CreativeModelCapability{}, &model.CreativeModelPublication{}, &model.CreativeChannelBinding{}))

	modelInput := dto.CreativeModelRequest{ModelName: "gpt-image-2", DisplayName: "GPT Image 2", Vendor: "OpenAI", Status: "enabled"}
	creativeModel, err := CreateCreativeModel(modelInput, 1)
	require.NoError(t, err)
	_, err = CreateCreativeModel(modelInput, 1)
	require.EqualError(t, err, "模型目录：gpt-image-2 已存在")
	otherModel, err := CreateCreativeModel(dto.CreativeModelRequest{ModelName: "gpt-image-3", DisplayName: "GPT Image 3", Vendor: "OpenAI", Status: "enabled"}, 1)
	require.NoError(t, err)
	_, err = UpdateCreativeModel(otherModel.ID, modelInput, 1)
	require.EqualError(t, err, "模型目录：gpt-image-2 已存在")

	invalidCapability := dto.CreativeCapabilityRequest{Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_images", ExecutionMode: "sync", InputSchema: json.RawMessage("{}"), DefaultParams: json.RawMessage("{}"), Enabled: true}
	_, err = CreateCreativeCapability(creativeModel.ID, invalidCapability, 1)
	require.EqualError(t, err, "protocol does not match the image capability contract")

	capabilityInput := dto.CreativeCapabilityRequest{Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_image", ExecutionMode: "sync", InputSchema: json.RawMessage("{}"), DefaultParams: json.RawMessage("{}"), Enabled: true}
	capability, err := CreateCreativeCapability(creativeModel.ID, capabilityInput, 1)
	require.NoError(t, err)
	_, err = CreateCreativeCapability(creativeModel.ID, capabilityInput, 1)
	require.EqualError(t, err, "能力：image/generate｜openai_image 已存在")
	otherCapability, err := CreateCreativeCapability(creativeModel.ID, dto.CreativeCapabilityRequest{Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_responses_image", ExecutionMode: "sync", InputSchema: json.RawMessage("{}"), DefaultParams: json.RawMessage("{}"), Enabled: true}, 1)
	require.NoError(t, err)
	_, err = UpdateCreativeCapability(otherCapability.ID, capabilityInput, 1)
	require.EqualError(t, err, "能力：image/generate｜openai_image 已存在")

	publicationInput := dto.CreativePublicationRequest{GroupName: "default", GroupDefaultParams: json.RawMessage("{}"), Enabled: true}
	publication, err := CreateCreativePublication(capability.ID, publicationInput, 1)
	require.NoError(t, err)
	_, err = CreateCreativePublication(capability.ID, publicationInput, 1)
	require.EqualError(t, err, "分组：default 已存在")
	otherPublication, err := CreateCreativePublication(capability.ID, dto.CreativePublicationRequest{GroupName: "vip", GroupDefaultParams: json.RawMessage("{}"), Enabled: true}, 1)
	require.NoError(t, err)
	_, err = UpdateCreativePublication(otherPublication.ID, publicationInput, 1)
	require.EqualError(t, err, "分组：default 已存在")

	require.NoError(t, database.Create(&model.Channel{Id: 1, Name: "渠道一", Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, database.Create(&model.Channel{Id: 2, Name: "渠道二", Status: common.ChannelStatusEnabled}).Error)
	binding, err := CreateCreativeBinding(publication.ID, dto.CreativeBindingRequest{ChannelID: 1}, 1)
	require.NoError(t, err)
	_, err = CreateCreativeBinding(publication.ID, dto.CreativeBindingRequest{ChannelID: 1}, 1)
	require.EqualError(t, err, "渠道：渠道一 已存在")
	otherBinding, err := CreateCreativeBinding(publication.ID, dto.CreativeBindingRequest{ChannelID: 2}, 1)
	require.NoError(t, err)
	_, err = UpdateCreativeBinding(otherBinding.ID, dto.CreativeBindingRequest{ChannelID: 1}, 1)
	require.EqualError(t, err, "渠道：渠道一 已存在")
	require.NotZero(t, binding.ID)
}

func TestCreativeModelRenameInvalidatesBindings(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-model-rename?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.CreativeModel{}, &model.CreativeModelCapability{}, &model.CreativeModelPublication{}, &model.CreativeChannelBinding{}))
	creativeModel := model.CreativeModel{ModelName: "old-image", ModelKey: "old-image", DisplayName: "Old", Vendor: "OpenAI", Status: "enabled"}
	require.NoError(t, database.Create(&creativeModel).Error)
	capability := model.CreativeModelCapability{ModelID: creativeModel.ID, Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_image", ExecutionMode: "sync", InputSchema: "{}", DefaultParams: "{}", Enabled: true}
	require.NoError(t, database.Create(&capability).Error)
	publication := model.CreativeModelPublication{CapabilityID: capability.ID, GroupName: "default", GroupDefaultParams: "{}", Enabled: true}
	require.NoError(t, database.Create(&publication).Error)
	binding := model.CreativeChannelBinding{PublicationID: publication.ID, ChannelID: 1, RequestModel: "old-image", Enabled: true, ValidationStatus: "valid", ValidationMessage: "ready"}
	require.NoError(t, database.Create(&binding).Error)

	_, err = UpdateCreativeModel(creativeModel.ID, dto.CreativeModelRequest{ModelName: "new-image", DisplayName: "New", Vendor: "OpenAI", Status: "enabled"}, 9)
	require.NoError(t, err)
	require.NoError(t, database.First(&binding, binding.ID).Error)
	assert.Equal(t, "new-image", binding.RequestModel)
	assert.False(t, binding.Enabled)
	assert.Equal(t, "unverified", binding.ValidationStatus)
}

func TestCreativeStudioSettingsUpgradeUsesSafeDefaults(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-settings-v2?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.Option{}))
	require.NoError(t, database.Create(&model.Option{Key: model.CreativeStudioSettingsOption, Value: `{"version":1,"default_file_channel_id":7}`}).Error)

	settings, err := GetCreativeStudioSettings()
	require.NoError(t, err)
	assert.Equal(t, 3, settings.Version)
	assert.Equal(t, uint64(7), settings.DefaultFileChannelID)
	assert.Equal(t, int64(32<<20), settings.MaxRasterBytes)
	assert.Equal(t, int64(1<<20), settings.MaxSVGBytes)
	assert.Contains(t, settings.AllowedImageMIMETypes, "image/svg+xml")
}

func TestCreativeBootstrapOnlyReturnsVerifiedBindings(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-user-bootstrap?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.User{}, &model.Option{}, &model.Channel{}, &model.Ability{}, &model.FileUploadChannel{}, &model.CreativeModel{}, &model.CreativeModelCapability{}, &model.CreativeModelPublication{}, &model.CreativeChannelBinding{}))
	require.NoError(t, database.Create(&model.User{Id: 7, Username: "creative-user", Password: "password", AffCode: "creative-bootstrap", Status: 1, Group: "default"}).Error)
	require.NoError(t, database.Create(&model.FileUploadChannel{Id: 9, Name: "S3", Type: "3", Status: "1"}).Error)
	require.NoError(t, database.Create(&model.Option{Key: model.CreativeStudioSettingsOption, Value: `{"version":2,"default_file_channel_id":9}`}).Error)
	creativeModel := model.CreativeModel{ModelName: "gpt-image", ModelKey: "gpt-image", DisplayName: "GPT Image", Vendor: "OpenAI", Status: "enabled"}
	require.NoError(t, database.Create(&creativeModel).Error)
	capability := model.CreativeModelCapability{ModelID: creativeModel.ID, Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_image", ExecutionMode: "sync", InputSchema: "{}", DefaultParams: "{}", Enabled: true}
	require.NoError(t, database.Create(&capability).Error)
	publication := model.CreativeModelPublication{CapabilityID: capability.ID, GroupName: "default", GroupDefaultParams: "{}", Enabled: true}
	require.NoError(t, database.Create(&publication).Error)
	require.NoError(t, database.Create(&model.Channel{Id: 1, Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, database.Create(&model.Ability{Group: "default", Model: "gpt-image", ChannelId: 1, Enabled: true}).Error)
	binding := model.CreativeChannelBinding{PublicationID: publication.ID, ChannelID: 1, RequestModel: "gpt-image", Enabled: false, ValidationStatus: "unverified"}
	require.NoError(t, database.Create(&binding).Error)

	bootstrap, err := CreativeUserBootstrap(7)
	require.NoError(t, err)
	assert.Empty(t, bootstrap["models"])
	validated, err := RevalidateCreativeBinding(binding.ID, 1)
	require.NoError(t, err)
	assert.True(t, validated.Enabled)
	assert.Equal(t, "valid", validated.ValidationStatus)
	assert.NotNil(t, validated.ValidationCheckedAt)
	bootstrap, err = CreativeUserBootstrap(7)
	require.NoError(t, err)
	assert.Len(t, bootstrap["models"], 1)
	require.NoError(t, database.Model(&model.Channel{}).Where("id = ?", 1).Update("status", common.ChannelStatusManuallyDisabled).Error)
	invalid, err := RevalidateCreativeBinding(binding.ID, 1)
	require.NoError(t, err)
	assert.False(t, invalid.Enabled)
	assert.Equal(t, "invalid", invalid.ValidationStatus)
	assert.Equal(t, "channel is unavailable", invalid.ValidationMessage)
	assert.NotNil(t, invalid.ValidationCheckedAt)
}
