package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreativeStudioReordersCompleteSiblingLists(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-reorder?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(
		&model.CreativeModel{},
		&model.CreativeModelCapability{},
		&model.CreativeModelPublication{},
		&model.CreativeChannelBinding{},
	))

	models := []model.CreativeModel{
		{ModelName: "image-one", ModelKey: "image-one", DisplayName: "Image One", Status: "enabled", SortOrder: 30},
		{ModelName: "image-two", ModelKey: "image-two", DisplayName: "Image Two", Status: "enabled", SortOrder: 10},
		{ModelName: "image-three", ModelKey: "image-three", DisplayName: "Image Three", Status: "enabled", SortOrder: 20},
	}
	require.NoError(t, database.Create(&models).Error)
	capabilities := []model.CreativeModelCapability{
		{ModelID: models[0].ID, Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_image", ExecutionMode: "sync", InputSchema: "{}", DefaultParams: "{}", SortOrder: 20},
		{ModelID: models[0].ID, Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_responses_image", ExecutionMode: "sync", InputSchema: "{}", DefaultParams: "{}", SortOrder: 30},
		{ModelID: models[0].ID, Category: "image", Operation: "edit", AssetKind: "raster", Protocol: "advanced_custom_image", ExecutionMode: "sync", InputSchema: "{}", DefaultParams: "{}", SortOrder: 10},
	}
	require.NoError(t, database.Create(&capabilities).Error)
	otherCapability := model.CreativeModelCapability{ModelID: models[1].ID, Category: "image", Operation: "generate", AssetKind: "raster", Protocol: "openai_image", ExecutionMode: "sync", InputSchema: "{}", DefaultParams: "{}"}
	require.NoError(t, database.Create(&otherCapability).Error)
	publications := []model.CreativeModelPublication{
		{CapabilityID: capabilities[0].ID, GroupName: "default", GroupDefaultParams: "{}", SortOrder: 30},
		{CapabilityID: capabilities[0].ID, GroupName: "vip", GroupDefaultParams: "{}", SortOrder: 10},
		{CapabilityID: capabilities[0].ID, GroupName: "team", GroupDefaultParams: "{}", SortOrder: 20},
	}
	require.NoError(t, database.Create(&publications).Error)
	bindings := []model.CreativeChannelBinding{
		{PublicationID: publications[0].ID, ChannelID: 1, RequestModel: models[0].ModelName, Priority: 10, ValidationStatus: "unverified", ValidationMessage: "not validated"},
		{PublicationID: publications[0].ID, ChannelID: 2, RequestModel: models[0].ModelName, Priority: 30, ValidationStatus: "unverified", ValidationMessage: "not validated"},
		{PublicationID: publications[0].ID, ChannelID: 3, RequestModel: models[0].ModelName, Priority: 20, ValidationStatus: "unverified", ValidationMessage: "not validated"},
	}
	require.NoError(t, database.Create(&bindings).Error)

	const userID = int64(9)
	require.NoError(t, ReorderCreativeStudio(dto.CreativeReorderRequest{
		Kind: "model",
		IDs:  []uint64{models[2].ID, models[0].ID, models[1].ID},
	}, userID))
	require.NoError(t, ReorderCreativeStudio(dto.CreativeReorderRequest{
		Kind:     "capability",
		ParentID: models[0].ID,
		IDs:      []uint64{capabilities[2].ID, capabilities[0].ID, capabilities[1].ID},
	}, userID))
	require.NoError(t, ReorderCreativeStudio(dto.CreativeReorderRequest{
		Kind:     "publication",
		ParentID: capabilities[0].ID,
		IDs:      []uint64{publications[1].ID, publications[2].ID, publications[0].ID},
	}, userID))
	require.NoError(t, ReorderCreativeStudio(dto.CreativeReorderRequest{
		Kind:     "binding",
		ParentID: publications[0].ID,
		IDs:      []uint64{bindings[2].ID, bindings[0].ID, bindings[1].ID},
	}, userID))

	orderedModels, err := ListCreativeModels()
	require.NoError(t, err)
	require.Len(t, orderedModels, 3)
	assert.Equal(t, []uint64{models[2].ID, models[0].ID, models[1].ID}, []uint64{orderedModels[0].ID, orderedModels[1].ID, orderedModels[2].ID})
	assert.Equal(t, []int{0, 1, 2}, []int{orderedModels[0].SortOrder, orderedModels[1].SortOrder, orderedModels[2].SortOrder})
	assert.Equal(t, []int64{userID, userID, userID}, []int64{orderedModels[0].UpdatedBy, orderedModels[1].UpdatedBy, orderedModels[2].UpdatedBy})

	orderedCapabilities, err := ListCreativeCapabilities(models[0].ID)
	require.NoError(t, err)
	require.Len(t, orderedCapabilities, 3)
	assert.Equal(t, []uint64{capabilities[2].ID, capabilities[0].ID, capabilities[1].ID}, []uint64{orderedCapabilities[0].ID, orderedCapabilities[1].ID, orderedCapabilities[2].ID})
	assert.Equal(t, []int{0, 1, 2}, []int{orderedCapabilities[0].SortOrder, orderedCapabilities[1].SortOrder, orderedCapabilities[2].SortOrder})
	assert.Equal(t, []int64{userID, userID, userID}, []int64{orderedCapabilities[0].UpdatedBy, orderedCapabilities[1].UpdatedBy, orderedCapabilities[2].UpdatedBy})

	orderedPublications, err := ListCreativePublications(capabilities[0].ID)
	require.NoError(t, err)
	require.Len(t, orderedPublications, 3)
	assert.Equal(t, []uint64{publications[1].ID, publications[2].ID, publications[0].ID}, []uint64{orderedPublications[0].ID, orderedPublications[1].ID, orderedPublications[2].ID})
	assert.Equal(t, []int{0, 1, 2}, []int{orderedPublications[0].SortOrder, orderedPublications[1].SortOrder, orderedPublications[2].SortOrder})
	assert.Equal(t, []int64{userID, userID, userID}, []int64{orderedPublications[0].UpdatedBy, orderedPublications[1].UpdatedBy, orderedPublications[2].UpdatedBy})

	orderedBindings, err := ListCreativeBindings(publications[0].ID)
	require.NoError(t, err)
	require.Len(t, orderedBindings, 3)
	assert.Equal(t, []uint64{bindings[2].ID, bindings[0].ID, bindings[1].ID}, []uint64{orderedBindings[0].ID, orderedBindings[1].ID, orderedBindings[2].ID})
	assert.Equal(t, []int{3, 2, 1}, []int{orderedBindings[0].Priority, orderedBindings[1].Priority, orderedBindings[2].Priority})
	assert.Equal(t, []int64{userID, userID, userID}, []int64{orderedBindings[0].UpdatedBy, orderedBindings[1].UpdatedBy, orderedBindings[2].UpdatedBy})

	require.EqualError(t, ReorderCreativeStudio(dto.CreativeReorderRequest{
		Kind:     "capability",
		ParentID: models[0].ID,
		IDs:      []uint64{capabilities[1].ID, otherCapability.ID, capabilities[0].ID},
	}, userID), "reorder ids must belong to the requested parent")
	require.EqualError(t, ReorderCreativeStudio(dto.CreativeReorderRequest{
		Kind:     "capability",
		ParentID: models[0].ID,
		IDs:      []uint64{capabilities[1].ID, capabilities[0].ID},
	}, userID), "reorder ids must contain every sibling exactly once")

	orderedCapabilities, err = ListCreativeCapabilities(models[0].ID)
	require.NoError(t, err)
	assert.Equal(t, []uint64{capabilities[2].ID, capabilities[0].ID, capabilities[1].ID}, []uint64{orderedCapabilities[0].ID, orderedCapabilities[1].ID, orderedCapabilities[2].ID})
}
