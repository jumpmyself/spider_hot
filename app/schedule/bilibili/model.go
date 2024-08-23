package bilibili

import "time"

type BiliBili struct {
	ID          int64     `json:"id" gorm:"id"`
	UpdateVer   int64     `json:"update_ver" gorm:"update_ver"`
	Title       string    `json:"title" gorm:"title"`
	Icon        string    `json:"icon" gorm:"icon"`
	Url         string    `json:"url" gorm:"url"`
	KeyWord     string    `json:"key_word" gorm:"key_word"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// TableName 表名称
func (*BiliBili) TableName() string {
	return "bilibili"
}

type Ret struct {
	Code    int    `json:"code" gorm:"code"`
	Message string `json:"message" gorm:"message"`
	Title   string `json:"title" gorm:"title"`
	Data    Data   `json:"data" gorm:"data"`
}
type Data struct {
	Trending *Trend `json:"trending" gorm:"trending"`
}
type Trend struct {
	Title   string  `json:"title" gorm:"title"`
	TrackId string  `json:"track_id" gorm:"track_id"`
	List    []*List `json:"list" gorm:"list"`
}
type List struct {
	KeyWord  string `json:"keyword" gorm:"keyword"`
	ShowName string `json:"show_name" gorm:"show_name"`
	Icon     string `json:"icon" gorm:"icon"`
	Url      string `json:"url" gorm:"url"`
	Goto     string `json:"goto" gorm:"goto"`
}
