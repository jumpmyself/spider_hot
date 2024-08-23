package jiefangjunbao

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
	currentDate := time.Now().Format("2006/01/02")
	api := fmt.Sprintf("http://www.81.cn/_szb/jfjb/%s/index.json?t=1718182570256", currentDate)

	client := &http.Client{}
	request, err := http.NewRequest("GET", api, nil)
	if err != nil {
		logrus.Errorf("解放军报: 创建请求失败: %v", err)
		return
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0")
	request.Header.Add("Cookie", "81_chinese_status=%22jian%22")

	response, err := client.Do(request)
	if err != nil {
		logrus.Errorf("解放军报: 请求失败: %v", err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.Errorf("解放军报: 读取响应内容失败: %v", err)
		return
	}

	var ret Ret
	if err := json.Unmarshal(body, &ret); err != nil {
		logrus.Errorf("解放军报: 解码响应内容失败: %v", err)
		return
	}

	data := processResponse(ret)
	hotinfoStr := buildHotInfoString(data)
	hashStr := tools.Sha256Hash(hotinfoStr)

	if err := updateRedisAndDB(hashStr, data); err != nil {
		logrus.Errorf("解放军报: 更新 Redis 和数据库失败: %v", err)
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
func processResponse(ret Ret) []*Jiefangjunbao {
	data := make([]*Jiefangjunbao, 0)
	for _, list := range ret.PaperInfo {
		for _, xylist := range list.XYList {
			if xylist.Title != "图片" {
				url := fmt.Sprintf("http://www.81.cn/szb_223187/szbxq/index.html?paperName=jfjb&paperDate=%s&paperNumber=%s&articleid=%d", list.Data, list.PName, xylist.ID)
				data = append(data, &Jiefangjunbao{
					UpdateVer:   time.Now().Unix(),
					Title:       xylist.Title,
					URL:         url,
					Version:     list.PName + list.Data,
					CreatedTime: time.Now(),
					UpdatedTime: time.Now(),
				})
			}
		}
	}
	return data
}
func buildHotInfoString(data []*Jiefangjunbao) string {
	var hotinfoStr string
	for _, item := range data {
		hotinfoStr += item.Title + item.URL
	}
	return hotinfoStr
}
func updateRedisAndDB(hashStr string, data []*Jiefangjunbao) error {
	ctx := context.Background()
	value, err := model.RedisClient.Get(ctx, "jiefangjunbao_hot").Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("从redis获取数据失败: %v", err)
	}

	if value == "" || value != hashStr {
		if err := setRedisAndDB(ctx, hashStr, data); err != nil {
			return err
		}
	} else {
		if err := updateDBTimestamps(); err != nil {
			return err
		}
	}
	return nil
}
func setRedisAndDB(ctx context.Context, hashStr string, data []*Jiefangjunbao) error {
	if err := model.RedisClient.Set(ctx, "jiefangjunbao_hot", hashStr, 0).Err(); err != nil {
		return fmt.Errorf("将 hashStr 数据设置进 redis 失败: %v", err)
	}
	if err := model.Conn.Create(data).Error; err != nil {
		return fmt.Errorf("将数据保存到数据库失败: %v", err)
	}
	HotDataJson, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("将热搜数据转换为 JSON 格式失败: %v", err)
	}
	if err := model.RedisClient.Set(ctx, "jiefangjunbao_hot_data", HotDataJson, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("将最新热搜数据写入 Redis 失败: %v", err)
	}
	return nil
}
func updateDBTimestamps() error {
	var maxUpdateVer int64
	var updateSlice []Jiefangjunbao
	model.Conn.Model(&Jiefangjunbao{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer)
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
func Refresh() ([]Jiefangjunbao, error) {
	var JiefangjunbaoList []Jiefangjunbao

	hotDataJson, err := model.RedisClient.Get(context.Background(), "jiefangjunbao_hot_data").Result()
	if err == redis.Nil {
		logrus.Errorf("解放军报：刷新 - Redis 中没有找到数据: %v", err)
		return JiefangjunbaoList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Errorf("解放军报：刷新 - 从 Redis 获取数据失败: %v", err)
		return JiefangjunbaoList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &JiefangjunbaoList); err != nil {
		logrus.Errorf("解放军报：刷新 - 反序列化 JSON 数据失败: %v", err)
		return JiefangjunbaoList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range JiefangjunbaoList {
		fmt.Printf("Title: %s, Url: %s\n", item.Title, item.URL)
	}

	return JiefangjunbaoList, nil
}
