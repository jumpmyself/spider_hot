package lol

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
	"time"
)

func init() {
	tools.LoadConfig()
}

// Run 是定时任务的入口函数
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

// Do 是立即执行任务的函数
func Do() {
	GetInfo(nil)
}

// extractJSONP 用于提取JSONP格式中的JSON数据部分
func extractJSONP(jsonp []byte) ([]byte, error) {
	re := regexp.MustCompile(`^[^(]*\((.*)\);$`)
	matches := re.FindSubmatch(jsonp)
	if len(matches) < 2 {
		return nil, fmt.Errorf("解析json失败")
	}
	return matches[1], nil
}

// GetInfo 获取 LOL 热搜信息
func GetInfo(c *gin.Context) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", viper.GetString("hot_api.lol"), nil)
	if err != nil {
		logrus.Error("lol:创建http请求失败err:", err)
		return
	}
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	request.Header.Add("Cookie", "RK=mpPIt1oMR7; ptcz=817d17a09c36755d9b6b9e4df893f7bda228e409b6b250ee1e3a536123aacdba; pgv_pvid=6835227561; pac_uid=0_a640ac7df3ca5; iip=0; qq_domain_video_guid_verify=54c4d2c196964f47; o_cookie=1790587200; Qs_lvt_323937=1698482180; Qs_pv_323937=2491088095741199000; fqm_pvqid=a426d807-bec2-4830-a33c-e981c2b6d9c1; _clck=1ikym9v|1|fh5|0; eas_sid=91R7W0j2L475Z9g1F7G7j0k4b2; _qimei_uuid42=181090e231d1009be1873878a0aa6f3d6122d55369; _qimei_fingerprint=5c19576b49cd88cdf62f5665aa8d4476; _qimei_q36=; _qimei_h38=e147844c05e0d48c447986ab0200000ae17916; LW_uid=7167o06448B0J4K494I8I0F8p6; isHostDate=19731; PTTuserFirstTime=1704758400000; isOsSysDate=19731; PTTosSysFirstTime=1704758400000; isOsDate=19731; PTTosFirstTime=1704758400000; ts_refer=lol.qq.com/; ts_uid=459324438; weekloop=0-0-0-2; LW_sid=l1S730R43840G4M8Q9P2k3D7S0; pgv_info=ssid=s8077240133; lolqqcomrouteLine=news_index-tool_main_news_index-tool_main_index-tool; tokenParams=%3Fdocid%3D1719755674302751220")

	response, err := client.Do(request)
	if err != nil {
		logrus.Error("lol:http请求失败err:", err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.Error("lol:读取响应信息失败err:", err)
		return
	}

	jsonData, err := extractJSONP(body)
	if err != nil {
		logrus.Error("lol:数据解析失败err:", err)
		return
	}

	var ret Ret
	err = json.Unmarshal(jsonData, &ret)
	if err != nil {
		logrus.Error("lol:解码ummarshal失败err:", err)
		return
	}

	data := make([]*Lol, 0)
	now := time.Now().Unix()
	var hotinfoStr string

	for _, result := range ret.Data.Result {
		url := "https://lol.qq.com/news/detail.shtml?docid=" + result.IDocId
		sTitle := result.STitle
		sIdxTime := result.SIdxTime
		tmp := Lol{
			UpdateVer:   now,
			Title:       sTitle,
			Url:         url,
			Time:        sIdxTime,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, &tmp)
		hotinfoStr = sTitle + url + hotinfoStr
	}

	hashStr := tools.Sha256Hash(hotinfoStr)

	value, err := model.RedisClient.Get(context.Background(), "lol_hot").Result()
	if err == redis.Nil {
		err = model.RedisClient.Set(context.Background(), "lol_hot", hashStr, 0).Err()
		if err != nil {
			logrus.Error("lol:将hashstr数据设置进redis err:", err)
			return
		}

		err = model.Conn.Create(data).Error
		if err != nil {
			logrus.Error("lol:将数据保存到数据库失败err:", err)
			return
		}

		HotDataJson, err := json.Marshal(data)
		if err != nil {
			logrus.Error("lol：将热搜数据转换为JSON格式失败", err)
			return
		}

		err = model.RedisClient.Set(context.Background(), "lol_hot_data", HotDataJson, 24*time.Hour).Err()
		if err != nil {
			logrus.Error("lol：将最新热搜数据写入 Redis 失败", err)
		}

	} else if err != nil {
		logrus.Error("lol:从redis获取数据失败err:", err)
	} else {
		if hashStr != value {
			err = model.RedisClient.Set(context.Background(), "lol_hot", hashStr, 0).Err()
			if err != nil {
				logrus.Error("lol:更新redis数据失败err:", err)
			}

			err = model.Conn.Create(data).Error
			if err != nil {
				logrus.Error("lol:将数据保存到数据库失败err:", err)
				return
			}

			HotDataJson, err := json.Marshal(data)
			if err != nil {
				logrus.Error("lol：将热搜数据转换为JSON格式失败", err)
				return
			}

			err = model.RedisClient.Set(context.Background(), "lol_hot_data", HotDataJson, 24*time.Hour).Err()
			if err != nil {
				logrus.Error("lol：将最新热搜数据写入 Redis 失败", err)
			}

		} else {
			var maxUpdateVer int64
			var updateSlice []Lol
			model.Conn.Model(&Lol{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer)
			model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice)
			for _, record := range updateSlice {
				record.UpdateVer = now
				record.UpdatedTime = time.Now()
				err := model.Conn.Save(&record).Error
				if err != nil {
					logrus.Error("lol:更新lol数据的版本号和时间err:", err)
					return
				}
			}
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

// Refresh 从 Redis 中刷新最新的 LOL 热搜数据
func Refresh() ([]Lol, error) {
	var lolList []Lol

	hotDataJson, err := model.RedisClient.Get(context.Background(), "lol_hot_data").Result()
	if err == redis.Nil {
		logrus.Error("lol：刷新 - Redis 中没有找到数据", err)
		return lolList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("lol：刷新 - 从 Redis 获取数据失败", err)
		return lolList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	err = json.Unmarshal([]byte(hotDataJson), &lolList)
	if err != nil {
		logrus.Error("lol：刷新 - 反序列化 JSON 数据失败", err)
		return lolList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range lolList {
		fmt.Printf("Title: %s, Url: %s, Time: %s\n", item.Title, item.Url, item.Time)
	}

	return lolList, nil
}
