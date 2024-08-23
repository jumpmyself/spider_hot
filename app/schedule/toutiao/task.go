package toutiao

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

func GetInfo(c *gin.Context) {
	client := &http.Client{}
	request, err := createRequest()
	if err != nil {
		logrus.Error("TOUTIAO: 创建HTTP请求失败:", err)
		return
	}

	response, err := client.Do(request)
	if err != nil {
		logrus.Error("TOUTIAO: HTTP请求失败:", err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.Error("TOUTIAO: 读取响应信息失败:", err)
		return
	}

	var toutiaoHot Hot
	if err = json.Unmarshal(body, &toutiaoHot); err != nil {
		logrus.Error("TOUTIAO: 解码JSON失败:", err)
		return
	}

	data, hashStr := processResponseData(toutiaoHot)
	updateDataInRedisAndDB(data, hashStr)
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

func createRequest() (*http.Request, error) {
	request, err := http.NewRequest("GET", viper.GetString("hot_api.toutiao"), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	return request, nil
}

func processResponseData(response Hot) ([]TouTiao, string) {
	now := time.Now().Unix()
	var data []TouTiao
	var hotinfoStr string

	for _, datum := range response.List {
		a := TouTiao{
			UpdateVer:   now,
			Title:       datum.Title,
			Icon:        datum.Label,
			Url:         datum.Url,
			Hot:         datum.Hot,
			LabelDesc:   datum.LabelDesc,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}

		data = append(data, a)
		hotinfoStr += datum.Title + datum.Url
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	return data, hashStr
}

func updateDataInRedisAndDB(data []TouTiao, hashStr string) {
	ctx := context.Background()

	value, err := model.RedisClient.Get(ctx, "toutiao_hot").Result()
	if err == redis.Nil {
		if err = model.RedisClient.Set(ctx, "toutiao_hot", hashStr, 0).Err(); err != nil {
			logrus.Error("TOUTIAO: 设置Redis数据失败:", err)
			return
		}

		if err = model.Conn.Create(data).Error; err != nil {
			logrus.Error("TOUTIAO: 保存数据到数据库失败:", err)
			return
		}

		storeHotDataInRedis(data)

	} else if err != nil {
		logrus.Error("TOUTIAO: 从Redis获取数据失败:", err)
	} else {
		if hashStr != value {
			if err = model.RedisClient.Set(ctx, "toutiao_hot", hashStr, 0).Err(); err != nil {
				logrus.Error("TOUTIAO: 更新Redis数据失败:", err)
			}

			if err = model.Conn.Create(data).Error; err != nil {
				logrus.Error("TOUTIAO: 保存数据到数据库失败:", err)
			}

			storeHotDataInRedis(data)

		} else {
			updateExistingData()
		}
	}
}

func storeHotDataInRedis(data []TouTiao) {
	ctx := context.Background()
	hotDataJson, err := json.Marshal(data)
	if err != nil {
		logrus.Error("TOUTIAO: 将热搜数据转换为JSON格式失败:", err)
		return
	}

	if err = model.RedisClient.Set(ctx, "toutiao_hot_data", hotDataJson, 24*time.Hour).Err(); err != nil {
		logrus.Error("TOUTIAO: 将最新热搜数据写入Redis失败:", err)
	}
}

func updateExistingData() {
	var maxUpdateVer int64
	now := time.Now()

	if err := model.Conn.Model(&TouTiao{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer).Error; err != nil {
		logrus.Error("TOUTIAO: 获取数据库中最大update_ver失败:", err)
		return
	}

	var updateSlice []TouTiao
	if err := model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice).Error; err != nil {
		logrus.Error("TOUTIAO: 获取具有最大update_ver的记录失败:", err)
		return
	}

	for _, record := range updateSlice {
		record.UpdateVer = now.Unix()
		record.UpdatedTime = now

		if err := model.Conn.Save(&record).Error; err != nil {
			logrus.Error("TOUTIAO: 更新记录失败:", err)
		}
	}
}

func Refresh() ([]TouTiao, error) {
	var toutiaoList []TouTiao
	ctx := context.Background()

	hotDataJson, err := model.RedisClient.Get(ctx, "toutiao_hot_data").Result()
	if err == redis.Nil {
		logrus.Error("TOUTIAO: 刷新 - Redis中没有找到数据")
		return toutiaoList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("TOUTIAO: 刷新 - 从Redis获取数据失败", err)
		return toutiaoList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err = json.Unmarshal([]byte(hotDataJson), &toutiaoList); err != nil {
		logrus.Error("TOUTIAO: 刷新 - 反序列化JSON数据失败", err)
		return toutiaoList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range toutiaoList {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.Hot)
	}

	return toutiaoList, nil
}
