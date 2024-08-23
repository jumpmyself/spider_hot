package jiefangjunbao

import "time"

// Jiefangjunbao undefined
type Jiefangjunbao struct {
	ID          int64     `json:"id" gorm:"id"`
	UpdateVer   int64     `json:"update_ver" gorm:"update_ver"`
	Title       string    `json:"title" gorm:"title"`
	URL         string    `json:"url" gorm:"url"`
	Version     string    `json:"version" gorm:"version"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// TableName 表名称
func (*Jiefangjunbao) TableName() string {
	return "jiefangjunbao"
}

type ResJiefangjunbao struct {
	Title string `json:"title" gorm:"title"`
	Url   string `json:"url" gorm:"url"`
}

type Ret struct {
	PaperInfo []*PaperInfo `json:"paperInfo"`
}
type PaperInfo struct {
	ID      int64     `json:"id" gorm:"id"`
	Data    string    `json:"paperData" gorm:"data"`
	Img     string    `json:"paperImg" gorm:"img"`
	PName   string    `json:"paperName" gorm:"p_name"`
	PNumber string    `json:"paperNumber" gorm:"p_number"`
	Url     string    `json:"webUrl" gorm:"url"`
	XYList  []*XYList `json:"xyList" gorm:"xy_list"`
}
type XYList struct {
	ID         int64    `json:"id" gorm:"id"`
	Title      string   `json:"Title" gorm:"g_title"`
	Content    string   `json:"content" gorm:"content"`
	GuideTitle string   `json:"guideTitle" gorm:"guideTitle"`
	ParentId   int64    `json:"parentId" gorm:"parent_id"`
	Point      []string `json:"point" gorm:"point"`
	Type       string   `json:"type" gorm:"type"`
	Author     string   `json:"author" gorm:"author"`
	Title2     string   `json:"title2" gorm:"title2"`
}
