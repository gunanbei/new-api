package model

import "time"

// File is a physical object. identifier and channel_type are the cross-user
// instant-upload key, so the same content on different providers stays distinct.
type File struct {
	Id            uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	FileChannelId uint64    `json:"file_channel_id" gorm:"column:file_channel_id;not null;default:0;index"`
	ChannelType   string    `json:"channel_type" gorm:"column:channel_type;type:varchar(1);not null;uniqueIndex:uk_identifier_channel_type;index"`
	FileSize      int64     `json:"file_size" gorm:"column:file_size;not null;default:0"`
	ObjectKey     string    `json:"object_key" gorm:"column:object_key;type:varchar(512);not null;default:''"`
	FileUrl       string    `json:"file_url" gorm:"column:file_url;type:varchar(1000);not null;default:''"`
	Identifier    string    `json:"identifier" gorm:"column:identifier;type:varchar(255);not null;uniqueIndex:uk_identifier_channel_type"`
	MimeType      string    `json:"mime_type" gorm:"column:mime_type;type:varchar(128);not null;default:''"`
	ETag          string    `json:"etag" gorm:"column:etag;type:varchar(128);not null;default:''"`
	RefCount      int       `json:"ref_count" gorm:"column:ref_count;not null;default:0"`
	Status        string    `json:"status" gorm:"column:status;type:varchar(1);not null;default:'1';index"`
	CreaterUserId int64     `json:"creater_user_id" gorm:"column:creater_user_id;not null;default:0"`
	CreateTime    time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime    time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
}

func (File) TableName() string { return "file" }
