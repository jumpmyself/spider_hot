package hyper

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/goccy/go-json"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"spider_hot/app/model"
	"spider_hot/app/tools"
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
func getFirstTag(tagList []*TagList) string {
	if len(tagList) > 0 {
		return tagList[0].Tag
	}
	return ""
}
func fetchHyperHotData() ([]*Hyper, string, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", viper.GetString("hot_api.hyper"), nil)
	if err != nil {
		return nil, "", fmt.Errorf("hyper: 创建http请求失败: %v", err)
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.76")
	request.Header.Add("Cookie", "ttwid=...")

	response, err := client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("hyper: http请求失败: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("hyper: 读取响应信息失败: %v", err)
	}

	var ret Ret
	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, "", fmt.Errorf("hyper: 解码json失败: %v", err)
	}

	data := make([]*Hyper, 0)
	now := time.Now().Unix()
	var hotinfoStr string
	for _, hyper := range ret.Data.HotNews {
		url := "https://www.thepaper.cn/newDetail_forward" + hyper.ContId
		tmp := Hyper{
			UpdateVer:   now,
			Title:       hyper.Name,
			KeyWord:     getFirstTag(hyper.TagList),
			Url:         url,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, &tmp)
		hotinfoStr = hyper.Name + url + hotinfoStr
	}

	return data, tools.Sha256Hash(hotinfoStr), nil
}
func storeHotData(data []*Hyper, hashStr string) error {
	HotDataJson, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("hyper: 将热搜数据转换为JSON格式失败: %v", err)
	}

	err = model.RedisClient.Set(context.Background(), "hyper_hot_data", HotDataJson, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("hyper: 将最新热搜数据写入 Redis 失败: %v", err)
	}

	return nil
}
func updateDatabase(data []*Hyper, now int64) error {
	var maxUpdateVer int64
	var updateSlice []Hyper

	if err := model.Conn.Model(&Hyper{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer).Error; err != nil {
		return fmt.Errorf("hyper: 查询数据库最大版本号失败: %v", err)
	}

	if err := model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice).Error; err != nil {
		return fmt.Errorf("hyper: 查询更新数据失败: %v", err)
	}

	for _, record := range updateSlice {
		record.UpdateVer = now
		record.UpdatedTime = time.Now()
		if err := model.Conn.Save(&record).Error; err != nil {
			return fmt.Errorf("hyper: 更新抖音数据的版本号和时间失败: %v", err)
		}
	}

	return nil
}
func GetInfo(c *gin.Context) {
	data, hashStr, err := fetchHyperHotData()
	if err != nil {
		logrus.Error(err)
		return
	}

	value, err := model.RedisClient.Get(context.Background(), "hyper_hot").Result()
	if err == redis.Nil {
		if err := model.RedisClient.Set(context.Background(), "hyper_hot", hashStr, 0).Err(); err != nil {
			logrus.Error("hyper: 将hashstr数据设置进redis失败: ", err)
			return
		}
		if err := model.Conn.Create(data).Error; err != nil {
			logrus.Error("hyper: 将数据保存到数据库失败: ", err)
			return
		}
		if err := storeHotData(data, hashStr); err != nil {
			logrus.Error(err)
		}
	} else if err != nil {
		logrus.Error("hyper: 从redis获取数据失败: ", err)
	} else if hashStr != value {
		if err := model.RedisClient.Set(context.Background(), "hyper_hot", hashStr, 0).Err(); err != nil {
			logrus.Error("hyper: 更新redis数据失败: ", err)
		}
		if err := model.Conn.Create(data).Error; err != nil {
			logrus.Error("hyper: 将数据保存到数据库失败: ", err)
		}
		if err := storeHotData(data, hashStr); err != nil {
			logrus.Error(err)
		}
	} else {
		now := time.Now().Unix()
		if err := updateDatabase(data, now); err != nil {
			logrus.Error(err)
		}
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
func Refresh() ([]Hyper, error) {
	var hyperList []Hyper

	hotDataJson, err := model.RedisClient.Get(context.Background(), "hyper_hot_data").Result()
	if err == redis.Nil {
		logrus.Error("hyper: 刷新 - Redis 中没有找到数据", err)
		return hyperList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("hyper: 刷新 - 从 Redis 获取数据失败", err)
		return hyperList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &hyperList); err != nil {
		logrus.Error("hyper: 刷新 - 反序列化 JSON 数据失败", err)
		return hyperList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range hyperList {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.KeyWord)
	}

	return hyperList, nil
}
