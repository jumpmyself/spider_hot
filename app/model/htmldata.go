package model

type HtmlData struct {
	Title string `json:"title" gorm:"title" `
	Url   string `json:"url" gorm:"url" `
	Hot   string `json:"hot" gorm:"hot" `
}
