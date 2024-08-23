package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"spider_hot/app/model"
	"spider_hot/app/tools"
	"strconv"
	"time"
)

const (
	redisKeyBilibiliHot     = "bilibili_hot"
	redisKeyBilibiliHotData = "bilibili_hot_data"
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
	body, err := fetchData(viper.GetString("hot_api.bilibili"))
	if err != nil {
		handleError(err, "bilibili: 获取热搜信息失败")
		return
	}

	data, hotinfoStr, err := parseJSON(body)
	if err != nil {
		handleError(err, "bilibili: 解析JSON失败")
		return
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	ctx := context.Background()
	redisClient := model.RedisClient

	needsUpdate, err := checkAndUpdateRedis(ctx, redisClient, hashStr)
	if err != nil {
		handleError(err, "bilibili: 检查和更新Redis失败")
		return
	}

	if needsUpdate {
		storeData(ctx, data)
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
func fetchData(url string) ([]byte, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("bilibili: 创建请求失败: %v", err)
	}
	request.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("bilibili: 发送请求失败: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("bilibili: 读取响应信息失败: %v", err)
	}

	return body, nil
}
func parseJSON(body []byte) ([]BiliBili, string, error) {
	var ret Ret
	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, "", fmt.Errorf("bilibili: JSON 解析失败: %v", err)
	}

	var data []BiliBili
	var hotinfoStr string
	now := time.Now().Unix()

	for _, list := range ret.Data.Trending.List {
		tmp := BiliBili{
			UpdateVer:   now,
			Title:       list.ShowName,
			Icon:        list.Icon,
			Url:         " https://search.bilibili.com/all?keyword=" + list.KeyWord,
			KeyWord:     list.KeyWord,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, tmp)
		hotinfoStr = list.ShowName + list.Icon + hotinfoStr
	}

	return data, hotinfoStr, nil
}
func checkAndUpdateRedis(ctx context.Context, redisClient *redis.Client, hashStr string) (bool, error) {
	value, err := redisClient.Get(ctx, redisKeyBilibiliHot).Result()
	if err == redis.Nil {
		return setRedisKey(ctx, redisClient, redisKeyBilibiliHot, hashStr)
	}

	if err != nil {
		return false, fmt.Errorf("bilibili: 从 Redis 获取数据失败: %v", err)
	}

	if hashStr != value {
		return setRedisKey(ctx, redisClient, redisKeyBilibiliHot, hashStr)
	}

	return false, nil
}
func setRedisKey(ctx context.Context, redisClient *redis.Client, key, value string) (bool, error) {
	if err := redisClient.Set(ctx, key, value, 0).Err(); err != nil {
		return false, fmt.Errorf("bilibili: 设置 Redis 键失败: %v", err)
	}
	return true, nil
}
func storeData(ctx context.Context, data []BiliBili) {
	if err := model.Conn.Create(data).Error; err != nil {
		handleError(err, "bilibili: 将新数据写入数据库失败")
		return
	}

	HotDataJson, err := json.Marshal(data)
	if err != nil {
		handleError(err, "bilibili: 将热搜数据转换为JSON格式失败")
		return
	}

	if err := model.RedisClient.Set(ctx, redisKeyBilibiliHotData, HotDataJson, 24*time.Hour).Err(); err != nil {
		handleError(err, "bilibili: 将最新热搜数据写入 Redis 失败")
	}
}
func Refresh() ([]BiliBili, error) {
	var biliList []BiliBili

	hotDataJson, err := model.RedisClient.Get(context.Background(), redisKeyBilibiliHotData).Result()
	if err == redis.Nil {
		logrus.Error("bilibili：刷新 - Redis 中没有找到数据")
		return biliList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("bilibili：刷新 - 从 Redis 获取数据失败", err)
		return biliList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &biliList); err != nil {
		logrus.Error("bilibili：刷新 - 反序列化 JSON 数据失败", err)
		return biliList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	logrus.Infof("Refreshed data from Redis:")
	for _, item := range biliList {
		logrus.Infof("Title: %s, Icon: %s, Keyword: %s", item.Title, item.Icon, item.KeyWord)
	}

	return biliList, nil
}
func handleError(err error, msg string) {
	if err != nil {
		logrus.Errorf("%s: %v", msg, err)
	}
}
