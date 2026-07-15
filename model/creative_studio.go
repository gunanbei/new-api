package model

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const CreativeStudioSettingsOption = "creative_studio.settings"

type CreativeModel struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	ModelName   string    `json:"model_name" gorm:"type:varchar(255);not null"`
	ModelKey    string    `json:"model_key" gorm:"type:varchar(255);uniqueIndex;not null"`
	DisplayName string    `json:"display_name" gorm:"type:varchar(255);not null"`
	Vendor      string    `json:"vendor" gorm:"type:varchar(255);not null"`
	Description string    `json:"description" gorm:"type:text;not null"`
	Status      string    `json:"status" gorm:"type:varchar(16);index;not null"`
	SortOrder   int       `json:"sort_order" gorm:"not null"`
	CreatedBy   int64     `json:"created_by" gorm:"not null"`
	UpdatedBy   int64     `json:"updated_by" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CreativeModel) TableName() string { return "creative_model" }

type CreativeModelCapability struct {
	ID            uint64    `json:"id" gorm:"primaryKey"`
	ModelID       uint64    `json:"model_id" gorm:"index;uniqueIndex:idx_creative_capability"`
	Category      string    `json:"category" gorm:"type:varchar(16);index;uniqueIndex:idx_creative_capability"`
	Operation     string    `json:"operation" gorm:"type:varchar(32);uniqueIndex:idx_creative_capability"`
	AssetKind     string    `json:"asset_kind" gorm:"type:varchar(16);not null"`
	Protocol      string    `json:"protocol" gorm:"type:varchar(64);uniqueIndex:idx_creative_capability"`
	ExecutionMode string    `json:"execution_mode" gorm:"type:varchar(32);not null"`
	InputSchema   string    `json:"input_schema" gorm:"type:text;not null"`
	DefaultParams string    `json:"default_params" gorm:"type:text;not null"`
	Enabled       bool      `json:"enabled" gorm:"index;not null"`
	SortOrder     int       `json:"sort_order" gorm:"not null"`
	CreatedBy     int64     `json:"created_by" gorm:"not null"`
	UpdatedBy     int64     `json:"updated_by" gorm:"not null"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CreativeModelCapability) TableName() string { return "creative_model_capability" }

type CreativeModelPublication struct {
	ID                 uint64    `json:"id" gorm:"primaryKey"`
	CapabilityID       uint64    `json:"capability_id" gorm:"index;uniqueIndex:idx_creative_publication"`
	GroupName          string    `json:"group_name" gorm:"type:varchar(64);index;uniqueIndex:idx_creative_publication"`
	Enabled            bool      `json:"enabled" gorm:"index;not null"`
	SortOrder          int       `json:"sort_order" gorm:"not null"`
	GroupDefaultParams string    `json:"group_default_params" gorm:"type:text;not null"`
	CreatedBy          int64     `json:"created_by" gorm:"not null"`
	UpdatedBy          int64     `json:"updated_by" gorm:"not null"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (CreativeModelPublication) TableName() string { return "creative_model_publication" }

type CreativeChannelBinding struct {
	ID                  uint64     `json:"id" gorm:"primaryKey"`
	PublicationID       uint64     `json:"publication_id" gorm:"index;uniqueIndex:idx_creative_binding"`
	ChannelID           int        `json:"channel_id" gorm:"index;uniqueIndex:idx_creative_binding"`
	RequestModel        string     `json:"request_model" gorm:"type:varchar(255);not null"`
	Priority            int        `json:"priority" gorm:"not null"`
	Enabled             bool       `json:"enabled" gorm:"index;not null"`
	ValidationStatus    string     `json:"validation_status" gorm:"type:varchar(16);index;not null"`
	ValidationMessage   string     `json:"validation_message" gorm:"type:text;not null"`
	ValidationCheckedAt *time.Time `json:"validation_checked_at" gorm:"index"`
	CreatedBy           int64      `json:"created_by" gorm:"not null"`
	UpdatedBy           int64      `json:"updated_by" gorm:"not null"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (CreativeChannelBinding) TableName() string { return "creative_channel_binding" }

func CreativeModelKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

func GetOptionValue(key string) (string, error) {
	var option Option
	err := DB.Where(&Option{Key: key}).First(&option).Error
	if err != nil {
		return "", err
	}
	return option.Value, nil
}

func IsCreativeStudioFileChannel(channelID uint64) (bool, error) {
	raw, err := GetOptionValue(CreativeStudioSettingsOption)
	if err != nil {
		return false, nil
	}
	var settings struct {
		DefaultFileChannelID uint64 `json:"default_file_channel_id"`
	}
	if err := common.UnmarshalJsonStr(raw, &settings); err != nil {
		return false, err
	}
	return settings.DefaultFileChannelID == channelID, nil
}
