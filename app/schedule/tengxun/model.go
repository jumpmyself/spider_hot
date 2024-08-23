package tengxun

import "time"

type TengXun struct {
	ID          int64     `json:"id" gorm:"id"`
	UpdateVer   int64     `json:"update_ver" gorm:"update_ver"`
	Title       string    `json:"title" gorm:"title"`
	Time        string    `json:"time" gorm:"time"`
	Url         string    `json:"url" gorm:"url"`
	Hot         int       `json:"hot" gorm:"hot"`
	Source      string    `json:"source" gorm:"source"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// TableName 表名称
func (TengXun) TableName() string {
	return "tengxun"
}

type Response struct {
	Ret      int           `json:"ret"`
	TopWords TopWords      `json:"topWords"`
	TraceID  string        `json:"trace_id"`
	Type     int           `json:"type"`
	Hotlist  []HotlistItem `json:"hotlist"`
}

type TopWords struct {
	Fixed     []string        `json:"fixed" gorm:"fixed"`
	Alternate []AlternateWord `json:"alternate" gorm:"alternate"`
}
type AlternateWord struct {
	Word string `json:"word" gorm:"word"`
	Form string `json:"form" gorm:"form"`
}
type HotlistItem struct {
	ID          string `json:"id" gorm:"id"`
	Title       string `json:"title" gorm:"title"`
	Time        string `json:"time"`
	Abstract    string `json:"abstract"`
	Comments    int    `json:"comments"`
	ShareUrl    string `json:"shareUrl"`
	LikeInfo    int    `json:"likeInfo"`
	ReadCount   int    `json:"readCount"`
	Source      string `json:"source"`
	TlTitle     string `json:"tlTitle"`
	NewsSource  string `json:"NewsSource"`
	UserAddress string `json:"userAddress"`
}
