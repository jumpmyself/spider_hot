package gongren

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/goccy/go-json"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"spider_hot/app/model"
	"spider_hot/app/tools"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	redisKeyGuangmingHot     = "guangming_hot"
	redisKeyGuangmingHotData = "guangming_hot_data"
)

// 初始化配置
var once sync.Once

func init() {
	once.Do(func() {
		tools.LoadConfig()
	})
}

// Run 启动定时任务
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

// Do 手动触发获取信息
func Do() {
	GetInfo(nil)
}

type PageInfo struct {
	Text string `json:"text"`
	Link string `json:"link"`
}

// GetLink 获取当日的版块名和链接
func GetLink() []PageInfo {
	currentDate := time.Now().Format("2006/01/02")
	api := fmt.Sprintf("https://www.workercn.cn/papers/grrb/%s/1/page.html", currentDate)
	body, err := fetchURL(api)
	if err != nil {
		logrus.Error("工人日报数据请求失败:", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		logrus.Error("工人日报：将html字符串加载到goquery失败：", err)
		return nil
	}

	var pagesInfo []PageInfo
	doc.Find("div.holder ul#pageUrl li").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Eq(1).Attr("href")
		text := s.Text()
		page := PageInfo{Text: text, Link: link}
		pagesInfo = append(pagesInfo, page)
	})
	fmt.Println("pageinfo", pagesInfo)
	return pagesInfo
}

// GetNews 获取新闻标题和链接
func GetNews(link string) ([]string, []string) {
	api := fmt.Sprintf("https://www.workercn.cn/%s", link)
	body, err := fetchURL(api)
	if err != nil {
		logrus.Error("工人日报：http请求失败", err)
		return nil, nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		logrus.Error("工人日报：将html字符串加载到goquery文档中失败：", err)
		return nil, nil
	}
	var titles, links []string
	doc.Find("div.holder ul#pageTitle li a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		title := strings.TrimSpace(s.Text())
		titles = append(titles, title)
		links = append(links, link)
	})
	return titles, links
}

// GetInfo 获取并处理信息
func GetInfo(c *gin.Context) {
	links := GetLink()
	if links == nil {
		logrus.Error("工人日报：未获取到任何版块链接")
		return
	}

	data := make([]*Gongren, 0)
	var hotinfoStr strings.Builder // 使用字符串构建器
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, link := range links {
		wg.Add(1)
		go func(link PageInfo) {
			defer wg.Done()
			titles, urls := GetNews(link.Link)
			if titles == nil || urls == nil {
				logrus.Error("工人日报：未获取到任何新闻")
				return
			}

			for i, title := range titles {
				fullUrl := fmt.Sprintf("https://www.workercn.cn%s", urls[i])
				gongren := Gongren{
					UpdateVer:   time.Now().Unix(),
					Title:       title,
					URL:         fullUrl,
					Version:     link.Text + "版",
					CreatedTime: time.Now(),
					UpdatedTime: time.Now(),
				}
				mu.Lock()
				data = append(data, &gongren)
				hotinfoStr.WriteString(title + fullUrl) // 使用字符串构建器添加数据
				mu.Unlock()
			}
		}(link)
	}

	wg.Wait()

	hashStr := tools.Sha256Hash(hotinfoStr.String()) // 获取构建完成的字符串的哈希值

	// 使用封装的 Redis 操作函数
	if shouldUpdate, err := shouldUpdateRedis(redisKeyGuangmingHot, hashStr); err == nil && shouldUpdate {
		model.Conn.Create(data)
		storeHotDataInRedis(data)
	} else if err != nil {
		logrus.Error("工人日报：处理 Redis 操作失败", err)
	} else {
		updateExistingRecords()
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

// fetchURL 获取 URL 的响应内容
func fetchURL(api string) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return "", fmt.Errorf("工人: 创建请求失败: %v", err)
	}
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.76")
	response, err := client.Do(request)

	if err != nil {
		return "", fmt.Errorf("工人: 发送请求错误: %v", err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("douyin: 读取响应失败: %v", err)
	}

	return string(body), nil

}

// shouldUpdateRedis 检查是否需要更新 Redis
func shouldUpdateRedis(key, hashStr string) (bool, error) {
	ctx := context.Background()
	value, err := model.RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		if err := model.RedisClient.Set(ctx, key, hashStr, 0).Err(); err != nil {
			return false, fmt.Errorf("将值写入 Redis 失败: %w", err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("从 Redis 获取数据失败: %w", err)
	}

	if hashStr == value {
		return false, nil
	}

	if err := model.RedisClient.Set(ctx, key, hashStr, 0).Err(); err != nil {
		return false, fmt.Errorf("更新 Redis 中 hashStr 的值失败: %w", err)
	}
	return true, nil
}

// storeHotDataInRedis 存储最新的热搜数据到 Redis
func storeHotDataInRedis(data []*Gongren) {
	hotDataJson, err := json.Marshal(data)
	if err != nil {
		logrus.Error("工人日报：将热搜数据转换为 JSON 格式失败", err)
		return
	}
	err = model.RedisClient.Set(context.Background(), redisKeyGuangmingHotData, hotDataJson, 24*time.Hour).Err()
	if err != nil {
		logrus.Error("工人日报：将最新热搜数据写入 Redis 失败", err)
	}
}

// updateExistingRecords 更新已有记录的版本号和时间
func updateExistingRecords() {
	var maxUpdateVer int64
	var updateSlice []Gongren
	err := model.Conn.Model(&Gongren{}).Select("MAX(update_ver) as max_update_ver").Scan(&maxUpdateVer).Error
	if err != nil {
		logrus.Error("获取最大更新版本失败", err)
		return
	}

	err = model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice).Error
	if err != nil {
		logrus.Error("获取需要更新的记录失败", err)
		return
	}

	for _, record := range updateSlice {
		record.UpdateVer = time.Now().Unix()
		record.UpdatedTime = time.Now()
		err := model.Conn.Save(&record).Error
		if err != nil {
			logrus.Error("更新工人日报信息失败", err)
			return
		}
	}
}
