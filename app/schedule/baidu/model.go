package baidu

import "time"

// Hot 接受百度的热搜数据
type Hot struct {
	Data Date `json:"data"`
}
type Date struct {
	List []ResDate `json:"list"`
}
type ResDate struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Pic       string `json:"pic"`
	Hot       int    `json:"hot"`
	Url       string `json:"url"`
	MobileUrl string `json:"mobileUrl"`
}

// 返回给前端的结构体
//
//	type ResDate struct {
//		Num  int    `json:"num" `
//		Name string `json:"name"`
//		Hot  string `json:"hot"`
//		Url  string `json:"url"`
//	}

// BaiDu 建表语句
// CREATE TABLE `baidu` (
// `id` bigint NOT NULL AUTO_INCREMENT,
// `update_ver` bigint DEFAULT NULL,
// `title` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
// `url` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
// `hot` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
// `created_time` datetime DEFAULT NULL,
// `updated_time` datetime DEFAULT NULL,
// PRIMARY KEY (`id`),
// KEY `index` (`update_ver`) USING BTREE
// ) ENGINE=InnoDB AUTO_INCREMENT=2498 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
type BaiDu struct {
	ID int64 `json:"id" gorm:"id"`
	//更新的版本
	UpdateVer int64 `json:"update_ver" gorm:"update_ver"`
	//热搜标题
	Title string `json:"title" gorm:"title"`
	//关键字 or url
	Url string `json:"url" gorm:"url"`
	//热度
	Hot string `json:"hot" gorm:"hot"`
	//创建时间
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	//更新时间
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// TableName 表名称
func (*BaiDu) TableName() string {
	return "baidu"
}
