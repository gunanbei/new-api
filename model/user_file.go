package model

import "time"

// UserFile is a user's logical reference to a physical File.
type UserFile struct {
	Id            uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	FileId        uint64    `json:"file_id" gorm:"column:file_id;not null;uniqueIndex:uk_user_file;index"`
	FileChannelId uint64    `json:"file_channel_id" gorm:"column:file_channel_id;not null;index"`
	FileName      string    `json:"file_name" gorm:"column:file_name;type:varchar(1000);not null"`
	FileSuffix    string    `json:"file_suffix" gorm:"column:file_suffix;type:varchar(255);not null;default:'';index:idx_user_suffix,priority:2"`
	UserId        int64     `json:"user_id" gorm:"column:user_id;not null;uniqueIndex:uk_user_file;index:idx_user_suffix,priority:1;index:idx_user_status,priority:1;index:idx_user_id_create_time,priority:1"`
	Source        string    `json:"source" gorm:"column:source;type:varchar(64);not null;default:''"`
	Status        string    `json:"status" gorm:"column:status;type:varchar(1);not null;default:'1';index:idx_user_status,priority:2"`
	CreateTime    time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime;index:idx_user_id_create_time,priority:2"`
	UpdateTime    time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
}

func (UserFile) TableName() string { return "user_file" }
