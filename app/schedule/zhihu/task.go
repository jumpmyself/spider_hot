package zhihu

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
	"regexp"
	"spider_hot/app/model"
	"spider_hot/app/tools"
	"strconv"
	"strings"
	"time"
)

const (
	redisKeyZhihuHot     = "zhihu_hot"
	redisKeyZhihuHotData = "zhihu_hot_data"
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

// fetchURL 获取 URL 的响应内容
func fetchURL(api string) ([]byte, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return nil, fmt.Errorf("创建http请求失败: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http请求失败: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应信息失败: %w", err)
	}

	return body, nil
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
func storeHotDataInRedis(data []ZhiHu) error {
	hotDataJson, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("将热搜数据转换为 JSON 格式失败: %w", err)
	}
	err = model.RedisClient.Set(context.Background(), redisKeyZhihuHotData, hotDataJson, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("将最新热搜数据写入 Redis 失败: %w", err)
	}
	return nil
}

// updateExistingRecords 更新已有记录的版本号和时间
func updateExistingRecords(now int64) error {
	var maxUpdateVer int64
	var updateSlice []ZhiHu
	err := model.Conn.Model(&ZhiHu{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer).Error
	if err != nil {
		return fmt.Errorf("获取最大更新版本失败: %w", err)
	}

	err = model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice).Error
	if err != nil {
		return fmt.Errorf("获取需要更新的记录失败: %w", err)
	}

	for _, record := range updateSlice {
		record.UpdateVer = now
		record.UpdatedTime = time.Now()
		err := model.Conn.Save(&record).Error
		if err != nil {
			return fmt.Errorf("更新知乎数据的版本号和时间失败: %w", err)
		}
	}
	return nil
}

// extractDigits 从字符串中提取所有数字并返回一个放大 10,000 倍的整数
func extractDigits(input string) (int64, error) {
	re := regexp.MustCompile(`\d+`)
	digitStr := re.FindString(input)
	digits, err := strconv.ParseInt(digitStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return digits * 10000, nil
}

func GetInfo(c *gin.Context) {
	body, err := fetchURL(viper.GetString("hot_api.zhihu"))
	if err != nil {
		logrus.Error("zhihu: ", err)
		return
	}

	var zhihuHot Hot
	err = json.Unmarshal(body, &zhihuHot)
	if err != nil {
		logrus.Error("zhihu: 解码ummarshal失败err: ", err)
		return
	}

	var data []ZhiHu
	now := time.Now().Unix()
	var hotinfoStr string
	for _, datum := range zhihuHot.Data {
		hot, err := extractDigits(datum.DeTail_Text)
		if err != nil {
			fmt.Println("Error extracting digits:", err)
			continue
		}
		//拼接网址
		url := datum.List.Url
		url1 := strings.Replace(url, "api.zhihu.com", "www.zhihu.com", 1)
		url2 := strings.Replace(url1, "/questions/", "/question/", 1)

		a := ZhiHu{
			UpdateVer:   now,
			Title:       datum.List.Title,
			Url:         url2,
			Hot:         strconv.FormatInt(hot, 10),
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, a)
		hotinfoStr += datum.List.Title + datum.List.Url
	}
	hashStr := tools.Sha256Hash(hotinfoStr)

	shouldUpdate, err := shouldUpdateRedis(redisKeyZhihuHot, hashStr)
	if err != nil {
		logrus.Error("zhihu: ", err)
		return
	}

	if shouldUpdate {
		err = model.Conn.Create(data).Error
		if err != nil {
			logrus.Error("zhihu: 将数据保存到数据库失败err: ", err)
			return
		}
		err = storeHotDataInRedis(data)
		if err != nil {
			logrus.Error("zhihu: ", err)
		}
	} else {
		err = updateExistingRecords(now)
		if err != nil {
			logrus.Error("zhihu: ", err)
		}
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

func Refresh() ([]ZhiHu, error) {
	var zhihuList []ZhiHu

	hotDataJson, err := model.RedisClient.Get(context.Background(), redisKeyZhihuHotData).Result()
	if err == redis.Nil {
		logrus.Error("zhihu: 刷新 - Redis 中没有找到数据", err)
		return zhihuList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("zhihu: 刷新 - 从 Redis 获取数据失败", err)
		return zhihuList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	err = json.Unmarshal([]byte(hotDataJson), &zhihuList)
	if err != nil {
		logrus.Error("zhihu: 刷新 - 反序列化 JSON 数据失败", err)
		return zhihuList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range zhihuList {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.Hot)
	}

	return zhihuList, nil
}
