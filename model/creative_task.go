package model

import "time"

const (
	CreativeTaskStatusValidating  = "validating"
	CreativeTaskStatusDispatching = "dispatching"
	CreativeTaskStatusProcessing  = "processing"
	CreativeTaskStatusImporting   = "importing"
	CreativeTaskStatusSucceeded   = "succeeded"
	CreativeTaskStatusFailed      = "failed"
)

type CreativeTask struct {
	ID              uint64     `json:"-" gorm:"primaryKey"`
	TaskKey         string     `json:"task_key" gorm:"type:varchar(64);uniqueIndex;not null"`
	UserID          int64      `json:"-" gorm:"index:idx_creative_task_user_created,priority:1;index:idx_creative_task_user_category_status_created,priority:1;not null"`
	Category        string     `json:"category" gorm:"type:varchar(16);index:idx_creative_task_user_category_status_created,priority:2;not null"`
	Operation       string     `json:"operation" gorm:"type:varchar(16);not null"`
	CapabilityID    uint64     `json:"-" gorm:"not null"`
	BindingID       uint64     `json:"-" gorm:"not null"`
	ModelName       string     `json:"model_name" gorm:"type:varchar(255);not null"`
	DisplayName     string     `json:"display_name" gorm:"type:varchar(255);not null"`
	GroupName       string     `json:"group_name" gorm:"type:varchar(64);not null"`
	Protocol        string     `json:"protocol" gorm:"type:varchar(64);not null"`
	ExecutionMode   string     `json:"execution_mode" gorm:"type:varchar(32);not null"`
	ChannelID       int        `json:"-" gorm:"not null"`
	RequestID       string     `json:"request_id,omitempty" gorm:"type:varchar(64);index"`
	ExternalTaskID  string     `json:"-" gorm:"type:varchar(191);index"`
	Status          string     `json:"status" gorm:"type:varchar(16);index:idx_creative_task_user_category_status_created,priority:3;index:idx_creative_task_status_updated,priority:1;not null"`
	RequestedParams string     `json:"requested_params" gorm:"type:text;not null"`
	ResolvedParams  string     `json:"resolved_params" gorm:"type:text;not null"`
	ResultManifest  string     `json:"-" gorm:"type:text;not null"`
	ImportExpiresAt *time.Time `json:"-"`
	ErrorCode       string     `json:"error_code,omitempty" gorm:"type:varchar(64);not null"`
	ErrorMessage    string     `json:"error_message,omitempty" gorm:"type:text;not null"`
	RetryOfID       uint64     `json:"-" gorm:"index"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at" gorm:"index:idx_creative_task_user_created,priority:2"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"index:idx_creative_task_status_updated,priority:2"`
}

func (CreativeTask) TableName() string { return "creative_task" }

type CreativeTaskAsset struct {
	ID         uint64    `json:"id" gorm:"primaryKey"`
	TaskID     uint64    `json:"-" gorm:"index;uniqueIndex:idx_creative_task_asset"`
	UserFileID uint64    `json:"user_file_id" gorm:"index;uniqueIndex:idx_creative_task_asset"`
	FileID     uint64    `json:"file_id" gorm:"index;not null"`
	Role       string    `json:"role" gorm:"type:varchar(16);uniqueIndex:idx_creative_task_asset"`
	Position   int       `json:"position" gorm:"uniqueIndex:idx_creative_task_asset"`
	MimeType   string    `json:"mime_type" gorm:"type:varchar(128);not null"`
	FileSize   int64     `json:"file_size" gorm:"not null"`
	FileName   string    `json:"file_name" gorm:"type:varchar(1000);not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (CreativeTaskAsset) TableName() string { return "creative_task_asset" }
