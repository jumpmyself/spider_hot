package taptap_android

import (
	"time"
)

type Ret struct {
	Data Data `json:"data"`
}
type Data struct {
	List []List `json:"list"`
}
type List struct {
	App App `json:"app"`
}
type App struct {
	Id    int64  `json:"id"`
	Title string `json:"title"`
	Stat  struct {
		Rating    Rating `json:"rating"`
		PlayTotal int64  `json:"play_total"`
	} `json:"stat"`
	Tags    []Tags `json:"tags"`
	RecText string `json:"rec_text"`
}
type Rating struct {
	Score string `json:"score"`
}
type Tags struct {
	ID     int    `json:"id"`
	Value  string `json:"value"`
	URI    string `json:"uri"`
	WebURL string `json:"web_url"`
}

// CREATE TABLE `taptap_android` (
// `id` bigint NOT NULL AUTO_INCREMENT,
// `update_ver` bigint DEFAULT NULL,
// `title` varchar(255) DEFAULT NULL,
// `tag` varchar(255) DEFAULT NULL,
// `url` varchar(255) DEFAULT NULL,
// `score` varchar(255) DEFAULT NULL,
// `play_total` bigint NOT NULL,
// `rec_text` varchar(255) DEFAULT NULL,
// `created_time` datetime DEFAULT NULL,
// `updated_time` datetime DEFAULT NULL,
// PRIMARY KEY (`id`),
// KEY `index` (`update_ver`) USING BTREE
// ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
type TaptapAndroid struct {
	Id          int64     `json:"id"  gorm:"column:id;primary_key;AUTO_INCREMENT;NOT NULL"`
	UpdateVer   int64     `json:"update_ver" gorm:"column:update_ver;default:NULL"`
	Title       string    `json:"title" gorm:"column:title;default:NULL"`
	Tag         string    `json:"tag" gorm:"column:tag;default:NULL"`
	Url         string    `json:"url" gorm:"column:url;default:NULL"`
	Score       string    `json:"score" gorm:"column:score;default:NULL"`
	PlayTotal   int64     `json:"play_total" gorm:"column:playtotal;NOT NULL"`
	RecText     string    `json:"rec_text" gorm:"column:rectext;default:NULL"`
	CreatedTime time.Time `json:"created_time" gorm:"column:created_time;default:NULL"`
	UpdatedTime time.Time `json:"updated_time" gorm:"column:updated_time;default:NULL"`
}
type ResTaptapAndroid struct {
	Title string `json:"title" gorm:"column:title;default:NULL"`
	Url   string `json:"url" gorm:"column:url;default:NULL"`
}

func (t *TaptapAndroid) TableName() string {
	return "taptapandroid"
}
