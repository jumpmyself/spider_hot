package qingnian

import "time"

// Qingnian undefined
type QingNian struct {
	ID          int64     `json:"id" gorm:"id"`
	UpdateVer   int64     `json:"update_ver" gorm:"update_ver"`
	Title       string    `json:"title" gorm:"title"`
	URL         string    `json:"url" gorm:"url"`
	Version     string    `json:"version" gorm:"version"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// TableName 表名称
func (*QingNian) TableName() string {
	return "qingnian"
}

type ResQingNian struct {
	Title string `json:"title" gorm:"title"`
	Url   string `json:"url" gorm:"url"`
}
