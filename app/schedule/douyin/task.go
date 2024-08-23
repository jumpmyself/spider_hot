package douyin

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

const (
	redisKeyDouyinHot     = "douyin_hot"
	redisKeyDouyinHotData = "douyin_hot_data"
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
	body, err := fetchData(viper.GetString("hot_api.douyin"))
	if err != nil {
		handleError(err, "douyin: 获取热搜信息失败")
		return
	}

	data, hotinfoStr, err := parseJSON(body)
	if err != nil {
		handleError(err, "douyin: 解析JSON失败")
		return
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	ctx := context.Background()
	redisClient := model.RedisClient

	needsUpdate, err := checkAndUpdateRedis(ctx, redisClient, hashStr)
	if err != nil {
		handleError(err, "douyin: 检查和更新Redis失败")
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
			Hot:   strconv.Itoa(item.Hot),
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
		return nil, fmt.Errorf("douyin: 创建请求失败: %v", err)
	}
	//添加User-Agent
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.76")
	//添加Cookie
	request.Header.Add("Cookie", "ttwid=1%7CZfTDRjzLikJLdXf5vUt-PHdoxm50XhdPorB3mEo4lpw%7C1704785277%7Cce482bdd0bd65873b36caf586eb7c9045d7833732ee7f7dd3b058b1ef28e1422; douyin.com; device_web_cpu_core=16; device_web_memory_size=8; architecture=amd64; dy_swidth=1536; dy_sheight=864; csrf_session_id=a1ad739a0df8eb5fe9da982694aff9be; strategyABtestKey=%221704785278.681%22; s_v_web_id=verify_lr614zum_BIQ8Wfu6_jgYj_4uHu_8mXd_XqCjU8M0yyU6; volume_info=%7B%22isUserMute%22%3Afalse%2C%22isMute%22%3Atrue%2C%22volume%22%3A0.5%7D; passport_csrf_token=f12476d310a7abece48ccf2000fb1191; passport_csrf_token_default=f12476d310a7abece48ccf2000fb1191; bd_ticket_guard_client_web_domain=2; ttcid=7b03da3762684406893a3f32208385dc41; bd_ticket_guard_client_data=eyJiZC10aWNrZXQtZ3VhcmQtdmVyc2lvbiI6MiwiYmQtdGlja2V0LWd1YXJkLWl0ZXJhdGlvbi12ZXJzaW9uIjoxLCJiZC10aWNrZXQtZ3VhcmQtcmVlLXB1YmxpYy1rZXkiOiJCS0VVT3ZCMkNSMnp5NTFXMUIvd1BnbkpSQWFKTGpzbCtkdC9hN2I4eTNZOVhaT0M3M3R0MW9nWXBKU0hNSDhkMkJBV3BvUU4xYWgzblJLS2gyQkhZazA9IiwiYmQtdGlja2V0LWd1YXJkLXdlYi12ZXJzaW9uIjoxfQ%3D%3D; FORCE_LOGIN=%7B%22videoConsumedRemainSeconds%22%3A180%2C%22isForcePopClose%22%3A1%7D; SEARCH_RESULT_LIST_TYPE=%22single%22; stream_player_status_params=%22%7B%5C%22is_auto_play%5C%22%3A0%2C%5C%22is_full_screen%5C%22%3A0%2C%5C%22is_full_webscreen%5C%22%3A0%2C%5C%22is_mute%5C%22%3A1%2C%5C%22is_speed%5C%22%3A1%2C%5C%22is_visible%5C%22%3A1%7D%22; download_guide=%223%2F20240109%2F0%22; pwa2=%220%7C0%7C2%7C0%22; __ac_nonce=0659d291c005f5a1850c3; __ac_signature=_02B4Z6wo00f01JHQKBgAAIDDxFKjh9DXr7yR8CyAAEHtp4exxEb809p7RSpC.w.F3L0FHh30yMxQho9R68PekYFwakelqedtamPdgojEStl5o24Ja5j2mnVJBXHzpGqbydNfmJi8xYHEUfwL1c; IsDouyinActive=true; stream_recommend_feed_params=%22%7B%5C%22cookie_enabled%5C%22%3Atrue%2C%5C%22screen_width%5C%22%3A1536%2C%5C%22screen_height%5C%22%3A864%2C%5C%22browser_online%5C%22%3Atrue%2C%5C%22cpu_core_num%5C%22%3A16%2C%5C%22device_memory%5C%22%3A8%2C%5C%22downlink%5C%22%3A10%2C%5C%22effective_type%5C%22%3A%5C%224g%5C%22%2C%5C%22round_trip_time%5C%22%3A50%7D%22; home_can_add_dy_2_desktop=%221%22; msToken=FfdOKnbchGkVoZiQZD7kkuyyOaIPoQi-REGWK_iPkQ6M0FReZ_U8jpC0znarZdetvuwm4ci7scr6wJlhMTcKduXusF6UULypEWL02etM8wDJJ_GIRQ==; msToken=qtTgVJDncrAxogJwklf8JK0ZteGMiSNmq-pNfhtJWdU9Mvz4JvELg5pCqGdlptpQEsYsAnUW2mj12EweyhP62eXkX6OhX9Gls3J0T-XBVfORVSo0Yw==; tt_scid=SVzYkwrJ9v54XP85PprplUQdaTjaTGzsSloYk5-SNwbZvahTdKAS5ThgWFqBNcJ903f4")
	// 发送请求并获取响应
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("douyin: 发送请求错误: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("douyin: 读取响应失败: %v", err)
	}

	return body, nil
}
func parseJSON(body []byte) ([]*DouYin, string, error) {
	var ret Hot
	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, "", fmt.Errorf("douyin: JSON 解析失败: %v", err)
	}

	var data []*DouYin
	var hotinfoStr string
	now := time.Now().Unix()

	for _, result := range ret.Data.List {
		tmp := &DouYin{
			UpdateVer:   now,
			Title:       result.Title,
			Url:         "https://www.douyin.com/search/" + result.Title,
			Hot:         result.Hot,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, tmp)
		hotinfoStr = result.Title + "https://www.douyin.com/search/" + result.Title + hotinfoStr
	}

	return data, hotinfoStr, nil
}
func checkAndUpdateRedis(ctx context.Context, redisClient *redis.Client, hashStr string) (bool, error) {
	value, err := redisClient.Get(ctx, redisKeyDouyinHot).Result()
	if err == redis.Nil {
		return setRedisKey(ctx, redisClient, redisKeyDouyinHot, hashStr)
	}

	if err != nil {
		return false, fmt.Errorf("douyin: 从 Redis 获取数据失败: %v", err)
	}

	if hashStr != value {
		return setRedisKey(ctx, redisClient, redisKeyDouyinHot, hashStr)
	}

	return false, nil
}
func setRedisKey(ctx context.Context, redisClient *redis.Client, key, value string) (bool, error) {
	if err := redisClient.Set(ctx, key, value, 0).Err(); err != nil {
		return false, fmt.Errorf("douyin: 设置 Redis 键失败: %v", err)
	}
	return true, nil
}
func storeData(ctx context.Context, data []*DouYin) {
	if err := model.Conn.Create(data).Error; err != nil {
		handleError(err, "douyin: 将新数据写入数据库失败")
		return
	}

	HotDataJson, err := json.Marshal(data)
	if err != nil {
		handleError(err, "douyin: 将热搜数据转换为JSON格式失败")
		return
	}

	if err := model.RedisClient.Set(ctx, redisKeyDouyinHotData, HotDataJson, 24*time.Hour).Err(); err != nil {
		handleError(err, "douyin: 将最新热搜数据写入 Redis 失败")
	}
}
func handleError(err error, msg string) {
	logrus.Error(msg, err)
}
func Refresh() ([]DouYin, error) {
	var douyinList []DouYin
	ctx := context.Background()

	hotDataJson, err := model.RedisClient.Get(ctx, redisKeyDouyinHotData).Result()
	if err == redis.Nil {
		logrus.Error("douyin: 刷新 - Redis 中没有找到数据", err)
		return douyinList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("douyin: 刷新 - 从 Redis 获取数据失败", err)
		return douyinList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &douyinList); err != nil {
		logrus.Error("douyin: 刷新 - 反序列化 JSON 数据失败", err)
		return douyinList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	printRefreshedData(douyinList)
	return douyinList, nil
}
func printRefreshedData(data []DouYin) {
	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range data {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.Hot)
	}
}
