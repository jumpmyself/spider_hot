package wangyi

import (
	"context"
	"fmt"
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
	"time"
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
	client := &http.Client{}
	request, err := http.NewRequest("GET", viper.GetString("hot_api.wangyi"), nil)
	if err != nil {
		logrus.Error("wangyi: 创建http请求失败: ", err)
		return
	}

	// 添加请求头
	setHeaders(request)

	resp, err := client.Do(request)
	if err != nil {
		logrus.Error("wangyi: http请求失败: ", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error("wangyi: 读取响应信息失败: ", err)
		return
	}

	var response T
	if err = json.Unmarshal(body, &response); err != nil {
		logrus.Error("wangyi: 解码失败: ", err)
		return
	}

	data, hashStr := processData(response)
	if err := saveDataToRedis(data, hashStr); err != nil {
		logrus.Error("wangyi: 保存数据到Redis失败: ", err)
		return
	}

	if err := saveDataToDatabase(data, hashStr); err != nil {
		logrus.Error("wangyi: 保存数据到数据库失败: ", err)
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
func setHeaders(request *http.Request) {
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.76")
	request.Header.Add("Cookie", "your_cookie_here")
}
func processData(response T) ([]*WangYi, string) {
	realtimeData := response.Data.List
	data := make([]*WangYi, 0)
	now := time.Now().Unix()
	var hotinfoStr string

	for _, list := range realtimeData {
		newStr := list.Title + list.Url

		tmp := WangYi{
			UpdateVer:   now,
			Title:       list.Title,
			KeyWord:     list.Keyword,
			Url:         list.Url,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, &tmp)
		hotinfoStr += newStr
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	return data, hashStr
}
func saveDataToRedis(data []*WangYi, hashStr string) error {
	value, err := model.RedisClient.Get(context.Background(), "wangyi_hot").Result()
	if err == redis.Nil {
		if err := model.RedisClient.Set(context.Background(), "wangyi_hot", hashStr, 0).Err(); err != nil {
			return fmt.Errorf("将hashstr数据设置进redis失败: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("从redis获取数据失败: %w", err)
	} else if hashStr == value {
		return nil
	}

	HotDataJson, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("将热搜数据转换为JSON格式失败: %w", err)
	}

	if err := model.RedisClient.Set(context.Background(), "wangyi_hot_data", HotDataJson, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("将最新热搜数据写入 Redis 失败: %w", err)
	}

	return nil
}
func saveDataToDatabase(data []*WangYi, hashStr string) error {
	if err := model.Conn.Create(data).Error; err != nil {
		return fmt.Errorf("将数据保存到数据库失败: %w", err)
	}

	var maxUpdateVer int64
	model.Conn.Model(&WangYi{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer)

	if maxUpdateVer > 0 {
		var updateSlice []WangYi
		model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice)
		for _, record := range updateSlice {
			record.UpdateVer = time.Now().Unix()
			record.UpdatedTime = time.Now()
			if err := model.Conn.Save(&record).Error; err != nil {
				return fmt.Errorf("更新数据版本号和时间失败: %w", err)
			}
		}
	}

	return nil
}
func Refresh() ([]WangYi, error) {
	var baiduList []WangYi

	hotDataJson, err := model.RedisClient.Get(context.Background(), "wangyi_hot_data").Result()
	if err == redis.Nil {
		logrus.Error("百度：刷新 - Redis 中没有找到数据: ", err)
		return baiduList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("百度：刷新 - 从 Redis 获取数据失败: ", err)
		return baiduList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &baiduList); err != nil {
		logrus.Error("百度：刷新 - 反序列化 JSON 数据失败: ", err)
		return baiduList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	for _, item := range baiduList {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.ID)
	}

	return baiduList, nil
}
