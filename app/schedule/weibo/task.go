package weibo

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
	defer func() {
		ticker.Stop()
	}()

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
	//创建http客户端
	client := &http.Client{}
	//创建Get请求
	request, err := http.NewRequest("GET", viper.GetString("hot_api.weibo"), nil)
	if err != nil {
		logrus.Error("weibo:创建http请求失败err:", err)
		return
	}
	//添加user-agent
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.76")
	//添加Cookie
	request.Header.Add("Cookie", "ttwid=1%7CZfTDRjzLikJLdXf5vUt-PHdoxm50XhdPorB3mEo4lpw%7C1704785277%7Cce482bdd0bd65873b36caf586eb7c9045d7833732ee7f7dd3b058b1ef28e1422; douyin.com; device_web_cpu_core=16; device_web_memory_size=8; architecture=amd64; dy_swidth=1536; dy_sheight=864; csrf_session_id=a1ad739a0df8eb5fe9da982694aff9be; strategyABtestKey=%221704785278.681%22; s_v_web_id=verify_lr614zum_BIQ8Wfu6_jgYj_4uHu_8mXd_XqCjU8M0yyU6; volume_info=%7B%22isUserMute%22%3Afalse%2C%22isMute%22%3Atrue%2C%22volume%22%3A0.5%7D; passport_csrf_token=f12476d310a7abece48ccf2000fb1191; passport_csrf_token_default=f12476d310a7abece48ccf2000fb1191; bd_ticket_guard_client_web_domain=2; ttcid=7b03da3762684406893a3f32208385dc41; bd_ticket_guard_client_data=eyJiZC10aWNrZXQtZ3VhcmQtdmVyc2lvbiI6MiwiYmQtdGlja2V0LWd1YXJkLWl0ZXJhdGlvbi12ZXJzaW9uIjoxLCJiZC10aWNrZXQtZ3VhcmQtcmVlLXB1YmxpYy1rZXkiOiJCS0VVT3ZCMkNSMnp5NTFXMUIvd1BnbkpSQWFKTGpzbCtkdC9hN2I4eTNZOVhaT0M3M3R0MW9nWXBKU0hNSDhkMkJBV3BvUU4xYWgzblJLS2gyQkhZazA9IiwiYmQtdGlja2V0LWd1YXJkLXdlYi12ZXJzaW9uIjoxfQ%3D%3D; FORCE_LOGIN=%7B%22videoConsumedRemainSeconds%22%3A180%2C%22isForcePopClose%22%3A1%7D; SEARCH_RESULT_LIST_TYPE=%22single%22; stream_player_status_params=%22%7B%5C%22is_auto_play%5C%22%3A0%2C%5C%22is_full_screen%5C%22%3A0%2C%5C%22is_full_webscreen%5C%22%3A0%2C%5C%22is_mute%5C%22%3A1%2C%5C%22is_speed%5C%22%3A1%2C%5C%22is_visible%5C%22%3A1%7D%22; download_guide=%223%2F20240109%2F0%22; pwa2=%220%7C0%7C2%7C0%22; __ac_nonce=0659d291c005f5a1850c3; __ac_signature=_02B4Z6wo00f01JHQKBgAAIDDxFKjh9DXr7yR8CyAAEHtp4exxEb809p7RSpC.w.F3L0FHh30yMxQho9R68PekYFwakelqedtamPdgojEStl5o24Ja5j2mnVJBXHzpGqbydNfmJi8xYHEUfwL1c; IsDouyinActive=true; stream_recommend_feed_params=%22%7B%5C%22cookie_enabled%5C%22%3Atrue%2C%5C%22screen_width%5C%22%3A1536%2C%5C%22screen_height%5C%22%3A864%2C%5C%22browser_online%5C%22%3Atrue%2C%5C%22cpu_core_num%5C%22%3A16%2C%5C%22device_memory%5C%22%3A8%2C%5C%22downlink%5C%22%3A10%2C%5C%22effective_type%5C%22%3A%5C%224g%5C%22%2C%5C%22round_trip_time%5C%22%3A50%7D%22; home_can_add_dy_2_desktop=%221%22; msToken=FfdOKnbchGkVoZiQZD7kkuyyOaIPoQi-REGWK_iPkQ6M0FReZ_U8jpC0znarZdetvuwm4ci7scr6wJlhMTcKduXusF6UULypEWL02etM8wDJJ_GIRQ==; msToken=qtTgVJDncrAxogJwklf8JK0ZteGMiSNmq-pNfhtJWdU9Mvz4JvELg5pCqGdlptpQEsYsAnUW2mj12EweyhP62eXkX6OhX9Gls3J0T-XBVfORVSo0Yw==; tt_scid=SVzYkwrJ9v54XP85PprplUQdaTjaTGzsSloYk5-SNwbZvahTdKAS5ThgWFqBNcJ903f4")
	//发送请求并获取响应
	resp, err := client.Do(request)
	if err != nil {
		logrus.Error("weibo:http请求失败err:", err)
		return
	}
	defer resp.Body.Close()

	//读取响应内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error("weibo:读取响应信息失败err:", err)
		return
	}
	//解析响应内容到结构体
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		logrus.Error("weibo:解码ummarshal失败err:", err)
		return
	}
	realtimeData := response.Data.Realtime
	data := make([]*WeiBo, 0)
	now := time.Now().Unix()
	head := "https://s.weibo.com/weibo?q=%23"
	tail := "%23&t=31&band_rank=2&Refer=top"
	var hotinfoStr string
	for _, list := range realtimeData {
		url := head + list.Note + tail
		tmp := WeiBo{
			UpdateVer:   now,
			Title:       list.Note,
			Hot:         list.RawHot,
			Url:         url,
			IconDesc:    list.IconDesc,
			Category:    list.Category,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, &tmp)
		hotinfoStr = hotinfoStr + list.Note + list.IconDesc + list.Category
	}
	model.Conn.Create(data)
	// 存储最新的热搜数据到 Redis
	HotDataJson, err := json.Marshal(data)
	if err != nil {
		logrus.Error("bilibili：将热搜数据转换为JSON格式失败", err)
		return
	}
	err = model.RedisClient.Set(context.Background(), "bilibili_hot_data", HotDataJson, 24*time.Hour).Err()
	if err != nil {
		logrus.Error("bilibili：将最新热搜数据写入 Redis 失败", err)
	}

	hashStr := tools.Sha256Hash(hotinfoStr)

	value, err := model.RedisClient.Get(context.Background(), "weibo_hot").Result()
	if err == redis.Nil {
		err = model.RedisClient.Set(context.Background(), "weibo_hot", hashStr, 0).Err()
		if err != nil {
			logrus.Error("weibo:将hashstr数据设置进redis err:", err)
			return
		}
		model.Conn.Create(data)
		// 存储最新的热搜数据到 Redis
		HotDataJson, err := json.Marshal(data)
		if err != nil {
			logrus.Error("weibo：将热搜数据转换为JSON格式失败", err)
			return
		}
		err = model.RedisClient.Set(context.Background(), "weibo_hot_data", HotDataJson, 24*time.Hour).Err()
		if err != nil {
			logrus.Error("weibo：将最新热搜数据写入 Redis 失败", err)
		}

	} else if err != nil {
		logrus.Error("weibo:从redis获取数据失败err:", err)
	} else {
		if hashStr != value {
			err = model.RedisClient.Set(context.Background(), "weibo_hot", hashStr, 0).Err()
			if err != nil {
				logrus.Error("weibo:更新redis数据失败err:", err)
			}
			err = model.Conn.Create(data).Error
			if err != nil {
				logrus.Error("weibo:将数据保存到数据库失败err:", err)
			}
			// 存储最新的热搜数据到 Redis
			HotDataJson, err := json.Marshal(data)
			if err != nil {
				logrus.Error("weibo：将热搜数据转换为JSON格式失败", err)
				return
			}
			err = model.RedisClient.Set(context.Background(), "weibo_hot_data", HotDataJson, 24*time.Hour).Err()
			if err != nil {
				logrus.Error("weibo：将最新热搜数据写入 Redis 失败", err)
			}

		} else {
			var maxUpdateVer int64
			var updateSlice []WeiBo
			model.Conn.Model(&WeiBo{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer)
			model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice)
			for _, record := range updateSlice {
				record.UpdateVer = now
				record.UpdatedTime = time.Now()
				err := model.Conn.Save(&record).Error
				if err != nil {
					logrus.Error("weibo:更新抖音数据的版本号和时间err:", err)
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
			Hot:   strconv.Itoa(item.Hot),
		})
	}

	// 返回部分数据给前端
	c.JSON(http.StatusOK, tools.ECode{
		Message: "",
		Data:    partialData,
	})
}
func Refresh() ([]WeiBo, error) {
	var baiduList []WeiBo

	// 从 Redis 中获取最新的热搜数据
	hotDataJson, err := model.RedisClient.Get(context.Background(), "weibo_hot_data").Result()
	if err == redis.Nil {
		logrus.Error("百度：刷新 - Redis 中没有找到数据", err)
		return baiduList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("百度：刷新 - 从 Redis 获取数据失败", err)
		return baiduList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	// 将 JSON 数据反序列化为 BaiDu 列表
	err = json.Unmarshal([]byte(hotDataJson), &baiduList)
	if err != nil {
		logrus.Error("百度：刷新 - 反序列化 JSON 数据失败", err)
		return baiduList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	// 打印查询结果
	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range baiduList {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.Hot)
	}

	return baiduList, nil
}
