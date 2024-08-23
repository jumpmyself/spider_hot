package jingji

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/goccy/go-json"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"io"
	"net/http"
	"spider_hot/app/model"
	"spider_hot/app/tools"
	"strconv"
	"strings"
	"time"
)

func init() {
	tools.LoadConfig()
}
func Run() {
	interval := viper.GetDuration("interval")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			GetInfo(nil)
		}
	}
}
func Do() {
	GetInfo(nil)
}

type PageInfo struct {
	Text string `json:"text"`
	Link string `json:"link"`
}

func GetLink() []PageInfo {
	currentDate := time.Now().Format("200601/02")
	api := fmt.Sprintf("http://paper.ce.cn/pc/layout/%s/node_01.html", currentDate)

	response, err := http.Get(api)
	if err != nil {
		logrus.Errorf("经济日报:http请求失败: %v", err)
		return nil
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.Errorf("经济日报:读取响应内容失败: %v", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		logrus.Errorf("经济日报:加载文件到goquery失败: %v", err)
		return nil
	}

	var pagesInfo []PageInfo
	fmt.Println("doc", doc)
	doc.Find("div.pull-left.page ul#layoutlist li").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")
		text := s.Find("a").Text()
		fmt.Println("link", link)
		page := PageInfo{
			Text: text,
			Link: link,
		}
		pagesInfo = append(pagesInfo, page)
	})
	return pagesInfo
}
func GetNews(link string) ([]string, []string) {
	currentDate := time.Now().Format("200601/02")
	api := fmt.Sprintf("http://paper.ce.cn/pc/layout/%s/%s", currentDate, link)

	response, err := http.Get(api)
	if err != nil {
		logrus.Errorf("经济日报:HTTP request failed: %v", err)
		return nil, nil
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.Errorf("经济日报:Failed to read response body: %v", err)
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		logrus.Errorf("经济日报:Failed to load HTML: %v", err)
		return nil, nil
	}

	var titles []string
	var links []string

	doc.Find("div.newsNav ul.newsList li").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")
		title := strings.TrimSpace(s.Find("a p").Text())
		if strings.Contains(title, "图片新闻") || strings.Contains(title, "来稿邮箱") {
			return
		}
		titles = append(titles, title)
		links = append(links, link)
	})
	return titles, links
}
func GetInfo(c *gin.Context) {
	links := GetLink()
	if links == nil {
		return
	}

	data, hotinfoStr := processLinks(links)
	hashStr := tools.Sha256Hash(hotinfoStr)

	ctx := context.Background()
	value, err := model.RedisClient.Get(ctx, "jingji_hot").Result()
	if err != nil && err != redis.Nil {
		logrus.Errorf("经济日报:从redis获取数据失败: %v", err)
		return
	}

	if value == "" || value != hashStr {
		if err := setRedisAndDB(ctx, hashStr, data); err != nil {
			logrus.Errorf("经济日报: 更新 Redis 和数据库失败: %v", err)
		}
	} else {
		if err := updateDBTimestamps(); err != nil {
			logrus.Errorf("经济日报: 更新数据版本号和时间失败: %v", err)
		}
	}
	// 构建返回给前端的部分数据
	var partialData []model.HtmlData
	for _, item := range data {
		partialData = append(partialData, model.HtmlData{
			Title: item.Title,
			Url:   item.URL,
			Hot:   strconv.Itoa(0),
		})
	}

	// 返回部分数据给前端
	c.JSON(http.StatusOK, tools.ECode{
		Message: "",
		Data:    partialData,
	})
}
func processLinks(links []PageInfo) ([]*Jingji, string) {
	data := make([]*Jingji, 0)
	var hotinfoStr string

	for _, link := range links {
		titles, urls := GetNews(link.Link)
		for i, title := range titles {
			fullUrl := fmt.Sprintf("http://paper.ce.cn/pc/%s", urls[i])
			newLink := strings.Replace(fullUrl, "../../../", "", 1)

			jingji := &Jingji{
				UpdateVer:   time.Now().Unix(),
				Title:       title,
				URL:         newLink,
				Version:     link.Text,
				CreatedTime: time.Now(),
				UpdatedTime: time.Now(),
			}
			data = append(data, jingji)
			hotinfoStr = title + newLink + hotinfoStr
		}
	}
	return data, hotinfoStr
}
func setRedisAndDB(ctx context.Context, hashStr string, data []*Jingji) error {
	if err := model.RedisClient.Set(ctx, "jingji_hot", hashStr, 0).Err(); err != nil {
		return fmt.Errorf("将 hashStr 数据设置进 redis 失败: %v", err)
	}
	if err := model.Conn.Create(data).Error; err != nil {
		return fmt.Errorf("将数据保存到数据库失败: %v", err)
	}
	HotDataJson, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("将热搜数据转换为 JSON 格式失败: %v", err)
	}
	if err := model.RedisClient.Set(ctx, "jingji_hot_data", HotDataJson, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("将最新热搜数据写入 Redis 失败: %v", err)
	}
	return nil
}
func updateDBTimestamps() error {
	var maxUpdateVer int64
	var updateSlice []Jingji
	model.Conn.Model(&Jingji{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer)
	model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice)
	for _, record := range updateSlice {
		record.UpdateVer = time.Now().Unix()
		record.UpdatedTime = time.Now()
		if err := model.Conn.Save(&record).Error; err != nil {
			return fmt.Errorf("更新数据版本号和时间失败: %v", err)
		}
	}
	return nil
}
func Refresh() ([]Jingji, error) {
	var jingjiList []Jingji

	ctx := context.Background()
	hotDataJson, err := model.RedisClient.Get(ctx, "jingji_hot_data").Result()
	if err == redis.Nil {
		logrus.Error("经济日报：刷新 - Redis 中没有找到数据")
		return jingjiList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Errorf("经济日报：刷新 - 从 Redis 获取数据失败: %v", err)
		return jingjiList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &jingjiList); err != nil {
		logrus.Errorf("经济日报：刷新 - 反序列化 JSON 数据失败: %v", err)
		return jingjiList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	fmt.Println("Refreshed data from Redis:")
	for _, item := range jingjiList {
		fmt.Printf("Title: %s, Url: %s\n", item.Title, item.URL)
	}

	return jingjiList, nil
}
