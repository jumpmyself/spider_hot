package zhihu

import "time"

type ZhiHu struct {
	ID          int64     `json:"id" gorm:"id"`
	UpdateVer   int64     `json:"update_ver" gorm:"update_ver"`
	Title       string    `json:"title" gorm:"title"`
	Url         string    `json:"url" gorm:"url"`
	Hot         string    `json:"hot" gorm:"hot"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// TableName 表名称
func (*ZhiHu) TableName() string {
	return "zhihu"
}

type Hot struct {
	Data []Date `json:"data" gorm:"data"`
}
type Date struct {
	List        ResData `json:"target" gorm:"list"`
	DeTail_Text string  `json:"detail_text" gorm:"detail_text"`
}
type ResData struct {
	Title string `json:"title" gorm:"title"`
	Url   string `json:"url" gorm:"url"`
	Type  string `json:"type" gorm:"type"`
}

type AutoGenerated struct {
	Data           []Data   `json:"data"`
	Paging         Paging   `json:"paging"`
	FreshText      string   `json:"fresh_text"`
	DisplayNum     int      `json:"display_num"`
	DisplayFirst   bool     `json:"display_first"`
	FbBillMainRise int      `json:"fb_bill_main_rise"`
	HeadZone       HeadZone `json:"head_zone"`
}
type CardLabel struct {
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	NightIcon string `json:"night_icon"`
}
type Author struct {
	Type      string `json:"type"`
	UserType  string `json:"user_type"`
	ID        string `json:"id"`
	URLToken  string `json:"url_token"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	Headline  string `json:"headline"`
	AvatarURL string `json:"avatar_url"`
}
type Target struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	Type          string `json:"type"`
	Created       int    `json:"created"`
	AnswerCount   int    `json:"answer_count"`
	FollowerCount int    `json:"follower_count"`
	Author        Author `json:"author"`
	BoundTopicIds []int  `json:"bound_topic_ids"`
	CommentCount  int    `json:"comment_count"`
	IsFollowing   bool   `json:"is_following"`
	Excerpt       string `json:"excerpt"`
}
type Children struct {
	Type      string `json:"type"`
	Thumbnail string `json:"thumbnail"`
}

type Paging struct {
	IsEnd    bool   `json:"is_end"`
	IsStart  bool   `json:"is_start"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Totals   int    `json:"totals"`
}
type Data struct {
	ID           string `json:"id"`
	LinkURL      string `json:"link_url"`
	Title        string `json:"title"`
	SourceType   int    `json:"source_type"`
	AttachedInfo string `json:"attached_info"`
	Tag          string `json:"tag"`
	TagBgColor   string `json:"tag_bg_color"`
}
type HeadZone struct {
	Type string `json:"type"`
	Data []Data `json:"data"`
}
