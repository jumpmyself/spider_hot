package guangming

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/goccy/go-json"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"spider_hot/app/model"
	"spider_hot/app/tools"
	"strconv"
	"strings"
	"time"
)

const (
	redisKeyGuangmingHot     = "guangming_hot"
	redisKeyGuangmingHotData = "guangming_hot_data"
)

func init() {
	tools.LoadConfig()
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
	currentDate := time.Now().Format("2006-01/02")
	api := fmt.Sprintf("https://epaper.gmw.cn/gmrb/html/%s/nbs.D110000gmrb_01.htm", currentDate)

	body, err := fetchHTTPBody(api)
	if err != nil {
		logrus.Error("请求光明日报数据失败:", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		logrus.Error("光明日报：将html字符串加载到goquery失败：", err)
		return nil
	}

	var pagesInfo []PageInfo
	doc.Find("span.modbd div.list_r div.l_c.l_c1 div#pageList ul li a#pageLink").Each(func(i int, s *goquery.Selection) {
		link := s.AttrOr("href", "")
		text := s.Text()
		page := PageInfo{Text: text, Link: link}
		pagesInfo = append(pagesInfo, page)
	})

	return pagesInfo
}

// GetNews 获取新闻标题和链接
func GetNews(link string) ([]string, []string) {
	currentDate := time.Now().Format("2006-01/02")
	api := fmt.Sprintf("https://epaper.gmw.cn/gmrb/html/%s/%s", currentDate, link)

	body, err := fetchHTTPBody(api)
	if err != nil {
		logrus.Error("光明日报：http请求失败", err)
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		logrus.Error("光明日报：将html字符串加载到goquery文档中失败err:", err)
		return nil, nil
	}

	var titles, links []string
	doc.Find("div#ozoom div.list_t div.list_l div.l_c div#titleList ul li").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")
		title := strings.TrimSpace(s.Find("a").Text())
		if strings.Contains(title, "图片报道") {
			return
		}
		titles = append(titles, title)
		links = append(links, link)
	})

	return titles, links
}

// GetInfo 获取并处理信息
func GetInfo(c *gin.Context) {
	links := GetLink()
	fmt.Println("links", links)
	data := make([]*GuangMing, 0)
	var hotinfoStr string

	for _, link := range links {
		titles, urls := GetNews(link.Link)
		for i, title := range titles {
			currentDate := time.Now().Format("2006-01/02")
			fullUrl := fmt.Sprintf("https://epaper.gmw.cn/gmrb/html/%s/%s", currentDate, urls[i])
			guangming := GuangMing{
				UpdateVer:   time.Now().Unix(),
				Title:       title,
				Url:         fullUrl,
				Version:     link.Text,
				CreatedTime: time.Now(),
				UpdatedTime: time.Now(),
			}
			data = append(data, &guangming)
			hotinfoStr = title + fullUrl + hotinfoStr
		}
	}

	hashStr := tools.Sha256Hash(hotinfoStr)

	ctx := context.Background()
	if updated, err := checkAndUpdateRedis(ctx, model.RedisClient, redisKeyGuangmingHot, hashStr); err != nil {
		logrus.Error("光明日报：检查和更新Redis数据失败", err)
	} else if updated {
		storeData(ctx, model.Conn, model.RedisClient, data, redisKeyGuangmingHotData)
	} else {
		updateExistingRecords()
	}
	// 构建返回给前端的部分数据
	var partialData []model.HtmlData
	for _, item := range data {
		partialData = append(partialData, model.HtmlData{
			Title: item.Title,
			Url:   item.Url,
			Hot:   strconv.Itoa(0),
		})
	}

	// 返回部分数据给前端
	c.JSON(http.StatusOK, tools.ECode{
		Message: "",
		Data:    partialData,
	})
}
func fetchHTTPBody(api string) (string, error) {
	response, err := http.Get(api)
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应内容失败: %v", err)
	}

	return string(body), nil
}
func checkAndUpdateRedis(ctx context.Context, redisClient *redis.Client, redisKey, hashStr string) (bool, error) {
	value, err := redisClient.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		return setRedisKey(ctx, redisClient, redisKey, hashStr)
	}
	if err != nil {
		return false, fmt.Errorf("从redis获取数据失败: %v", err)
	}
	if hashStr == value {
		return false, nil
	}
	return setRedisKey(ctx, redisClient, redisKey, hashStr)
}
func setRedisKey(ctx context.Context, redisClient *redis.Client, key, value string) (bool, error) {
	if err := redisClient.Set(ctx, key, value, 0).Err(); err != nil {
		return false, fmt.Errorf("设置redis键失败: %v", err)
	}
	return true, nil
}
func storeData(ctx context.Context, db *gorm.DB, redisClient *redis.Client, data []*GuangMing, redisKeyData string) {
	if err := db.Create(data).Error; err != nil {
		handleError(err, "将新数据写入数据库失败")
		return
	}

	hotDataJson, err := json.Marshal(data)
	if err != nil {
		handleError(err, "将热搜数据转换为JSON格式失败")
		return
	}

	if err := redisClient.Set(ctx, redisKeyData, hotDataJson, 24*time.Hour).Err(); err != nil {
		handleError(err, "将最新热搜数据写入 Redis 失败")
	}
}
func handleError(err error, msg string) {
	fmt.Printf("%s: %v\n", msg, err)
	// You can handle the error further here, e.g., logging, notifying, etc.
}
func updateExistingRecords() {
	var maxUpdateVer int64
	var updateSlice []GuangMing
	err := model.Conn.Model(&GuangMing{}).Select("MAX(update_ver) as max_update_ver").Scan(&maxUpdateVer).Error
	if err != nil {
		logrus.Error("获取最大更新版本失败", err)
		return
	}
	model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice)
	for _, record := range updateSlice {
		record.UpdateVer = time.Now().Unix()
		record.UpdatedTime = time.Now()
		err := model.Conn.Save(&record).Error
		if err != nil {
			logrus.Error("更新光明日报信息失败", err)
			return
		}
	}
}
