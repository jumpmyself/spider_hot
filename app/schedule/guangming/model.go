package guangming

import "time"

// CREATE TABLE `guangming` (
// `id` bigint NOT NULL,
// `update_ver` bigint DEFAULT NULL,
// `title` varchar(255) DEFAULT NULL,
// `url` varchar(255) DEFAULT NULL,
// `version` varchar(255) DEFAULT NULL,
// `created_time` datetime DEFAULT NULL,
// `updated_time` datetime DEFAULT NULL,
// PRIMARY KEY (`id`),
// KEY `index` (`update_ver`) USING BTREE
// ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

type GuangMing struct {
	Id          int64     `json:"id" gorm:"primary_key;column:id"`
	UpdateVer   int64     `json:"update_ver" gorm:"column:update_ver"`
	Title       string    `json:"title" gorm:"column:title"`
	Url         string    `json:"url" gorm:"column:url"`
	Version     string    `json:"version" gorm:"column:version"`
	CreatedTime time.Time `json:"created_time" gorm:"column:created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"column:updated_time"`
}

type ResGuangMing struct {
	Title string `json:"title" gorm:"column:title"`
	Url   string `json:"url" gorm:"column:url"`
}

func (g *GuangMing) TableName() string {
	return "guangming"
}
