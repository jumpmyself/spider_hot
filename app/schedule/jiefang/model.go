package jiefang

import "time"

// Jiefang undefined
type Jiefang struct {
	ID          int64     `json:"id" gorm:"id"`
	UpdateVer   int64     `json:"update_ver" gorm:"update_ver"`
	Title       string    `json:"title" gorm:"title"`
	URL         string    `json:"url" gorm:"url"`
	Version     string    `json:"version" gorm:"version"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// TableName 表名称
func (*Jiefang) TableName() string {
	return "jiefang"
}

type Ret struct {
	Pages []*pages `json:"pages"`
}
type pages struct {
	ImgUrl      string         `json:"imgurl"`
	ArticleList []*ArticleList `json:"articleList"`
	PName       string         `json:"pname"`
	PNumber     string         `json:"pnumber"`
	JDate       string         `json:"jdate"`
}
type ArticleList struct {
	Id    int64  `json:"id"`
	Title string `json:"title"`
}

type ResJieFang struct {
	Title string `json:"title" gorm:"title"`
	Url   string `json:"url" gorm:"url"`
}
