package baidu

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
	"strings"
	"time"
)

const (
	redisKeyBaiduHot     = "baidu_hot"
	redisKeyBaiduHotData = "baidu_hot_data"
)

func init() {
	tools.LoadConfig()
}

func Run() {
	interval := viper.GetDuration("interval")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		GetInfo(nil)
	}
}

func Do() {
	GetInfo(nil)
}

func GetInfo(c *gin.Context) {
	body, err := fetchData(viper.GetString("hot_api.baidu"))
	if err != nil {
		handleError(err, "获取百度数据失败")
		return
	}

	data, hotinfoStr, err := parseHTML(body)
	if err != nil {
		handleError(err, "解析HTML失败")
		return
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	ctx := context.Background()
	redisClient := model.RedisClient

	needsUpdate, err := checkAndUpdateRedis(ctx, redisClient, hashStr)
	if err != nil {
		handleError(err, "检查和更新Redis失败")
		return
	}

	if needsUpdate {
		storeData(data)
	}

	// 构建返回给前端的部分数据
	var partialData []model.HtmlData
	for _, item := range data {
		partialData = append(partialData, model.HtmlData{
			Title: item.Title,
			Url:   item.Url,
			Hot:   item.Hot,
		})
	}

	// 返回部分数据给前端
	c.JSON(http.StatusOK, tools.ECode{
		Message: "",
		Data:    partialData,
	})
}

func fetchData(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求百度数据失败: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	return body, nil
}

func parseHTML(body []byte) ([]BaiDu, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	fmt.Println("doc", doc)
	if err != nil {
		return nil, "", fmt.Errorf("将html字符串加载到goquery失败: %v", err)
	}

	var data []BaiDu
	var hotinfoStr string

	doc.Find("#sanRoot").Each(func(i int, s *goquery.Selection) {
		divContent, _ := s.Html()
		res := strings.Split(divContent, "热搜榜")
		re := strings.Split(res[0], "appUrl")

		for i1, s2 := range re {
			if i1 != 0 {
				// 获取url
				ss := strings.Split(s2, "&amp")[0]
				u1 := strings.Split(ss, "https://www.baidu.com/")[1]
				url1 := fmt.Sprintf("https://www.baidu.com/%s", u1)
				var titl1 string
				if i1 == 1 {
					// 获取标题
					ss1 := strings.Split(s2, "word\":\"")[1]
					a1 := strings.Split(ss1, "\"}")[0]
					titl1 = strings.Split(a1, "\",\"isTop")[0]
				} else {
					// 获取标题
					ss1 := strings.Split(s2, "word\":\"")[1]
					titl1 = strings.Split(ss1, "\"}")[0]
				}
				// 获取热度
				ss2 := strings.Split(s2, "hotScore\":\"")[1]
				hot1 := strings.Split(ss2, "\",\"hotTag")[0]

				a := BaiDu{
					UpdateVer:   time.Now().Unix(),
					Title:       titl1,
					Url:         url1,
					Hot:         hot1,
					CreatedTime: time.Now(),
					UpdatedTime: time.Now(),
				}
				data = append(data, a)
				hotinfoStr = titl1 + url1 + hotinfoStr
			}
		}
	})

	return data, hotinfoStr, nil
}

func checkAndUpdateRedis(ctx context.Context, redisClient *redis.Client, hashStr string) (bool, error) {
	value, err := redisClient.Get(ctx, redisKeyBaiduHot).Result()
	if err == redis.Nil {
		return setRedisKey(ctx, redisClient, redisKeyBaiduHot, hashStr)
	}

	if err != nil {
		return false, fmt.Errorf("从redis获取百度数据失败: %v", err)
	}

	if hashStr == value {
		return false, nil
	}

	return setRedisKey(ctx, redisClient, redisKeyBaiduHot, hashStr)
}

func setRedisKey(ctx context.Context, redisClient *redis.Client, key, value string) (bool, error) {
	if err := redisClient.Set(ctx, key, value, 0).Err(); err != nil {
		return false, fmt.Errorf("设置redis键失败: %v", err)
	}
	return true, nil
}

func storeData(data []BaiDu) {
	ctx := context.Background()
	redisClient := model.RedisClient

	if err := model.Conn.Create(data).Error; err != nil {
		handleError(err, "将新数据写入数据库失败")
		return
	}

	HotDataJson, err := json.Marshal(data)
	if err != nil {
		handleError(err, "将热搜数据转换为JSON格式失败")
		return
	}

	if err := redisClient.Set(ctx, redisKeyBaiduHotData, HotDataJson, 24*time.Hour).Err(); err != nil {
		handleError(err, "将最新热搜数据写入 Redis 失败")
	}
}

func Refresh() ([]BaiDu, error) {
	var baiduList []BaiDu

	hotDataJson, err := model.RedisClient.Get(context.Background(), redisKeyBaiduHotData).Result()
	if err == redis.Nil {
		return baiduList, fmt.Errorf("刷新 - Redis 中没有找到数据")
	}
	if err != nil {
		return baiduList, fmt.Errorf("刷新 - 从 Redis 获取数据失败: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &baiduList); err != nil {
		return baiduList, fmt.Errorf("刷新 - 反序列化 JSON 数据失败: %v", err)
	}

	logrus.Infof("Refreshed data from Redis:\n")
	for _, item := range baiduList {
		logrus.Infof("Title: %s, Url: %s, Hot: %s", item.Title, item.Url, item.Hot)
	}

	return baiduList, nil
}

func handleError(err error, msg string) {
	if err != nil {
		logrus.Errorf("%s: %v", msg, err)
	}
}
