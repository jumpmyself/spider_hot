package bilibili_rank

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
	redisKeyBilibiliRankHot     = "bilibili_rank_hot"
	redisKeyBilibiliRankHotData = "bilibili_rank_hot_data"
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
	body, err := fetchData(viper.GetString("hot_api.bilibili_rank"))
	if err != nil {
		handleError(err, "bilibili_rank: 获取热搜信息失败")
		return
	}

	data, hotinfoStr, err := parseJSON(body)
	if err != nil {
		handleError(err, "bilibili_rank: 解析JSON失败")
		return
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	ctx := context.Background()
	redisClient := model.RedisClient

	needsUpdate, err := checkAndUpdateRedis(ctx, redisClient, hashStr)
	if err != nil {
		handleError(err, "bilibili_rank: 检查和更新Redis失败")
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
		return nil, fmt.Errorf("bilibili_rank: 创建请求失败: %v", err)
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	request.Header.Add("Cookie", "buvid3=43094C0F-3E29-EEEC-7AF9-71709B5FB1D330870infoc; b_nut=1689662630; i-wanna-go-back=-1; _uuid=258B7F9B-3FAB-5833-EF4D-E34410EFCFA6136563infoc; FEED_LIVE_VERSION=V8; rpdid=|(~ummuR|k~0J'uY)mYYkY~l; DedeUserID=39362173; DedeUserID__ckMd5=0a62ae35f67fb3ec; buvid4=DFD3FA46-D7A5-1705-5B4D-6FA923F5A8BA45558-023071814-hTRASHGPQlzJ6%2BUUl9tIRw%3D%3D; header_theme_version=CLOSE; buvid_fp_plain=undefined; hit-new-style-dyn=1; hit-dyn-v2=1; b_ut=5; is-2022-channel=1; enable_web_push=DISABLE; CURRENT_BLACKGAP=0; CURRENT_FNVAL=4048; dy_spec_agreed=1; CURRENT_QUALITY=80; fingerprint=4d23e39272c183e66fde0d7793bc93a2; buvid_fp=4f9bfd389e80ed5e3b4daa8231fb596f; LIVE_BUVID=AUTO3817037443317552; bili_ticket=eyJhbGciOiJIUzI1NiIsImtpZCI6InMwMyIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MDQ5NjEzOTMsImlhdCI6MTcwNDcwMjEzMywicGx0IjotMX0.5jYWcWYlWdJAt8ubGPESTrb-cxL5AnzNgK83cKqR3As; bili_ticket_expires=1704961333; home_feed_column=5; browser_resolution=1920-969; SESSDATA=f28f92ab%2C1720487121%2Cbb06a%2A12CjDbycLLcqhZGyxdoxgLWPAh5a3jz4v6EiAGEbyaO9IV3fnzj5YJQOE7t4yur1IAzLUSVjg3Zjh5bHVaeEo3SGtpNVdfdVhiTjBscFRKMDZjcVdEZUpuSHkzREg1MzRCcERmanJNMzB4eVhYQmhQTHAwSVh3N0lMLWFKOVRFVGR5TnRIZloyR3J3IIEC; bili_jct=3fed309212edca07cd3057d23ee85d2a; sid=7913l753; bp_video_offset_39362173=884953491410255958; PVID=4; b_lsid=7A2C107BE_18CF6A671CE; innersign=0")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("bilibili_rank: 发送请求错误: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("bilibili_rank: 读取响应失败: %v", err)
	}

	return body, nil
}
func parseJSON(body []byte) ([]*BRank, string, error) {
	var ret Ret
	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, "", fmt.Errorf("bilibili_rank: JSON 解析失败: %v", err)
	}

	var data []*BRank
	var hotinfoStr string
	now := time.Now().Unix()

	for _, result := range ret.Data.List {
		tmp := &BRank{
			UpdateVer:   now,
			Title:       result.Title,
			Author:      result.Owner.Name,
			Url:         result.ShortLinkV2,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, tmp)
		hotinfoStr = result.Title + result.TName + hotinfoStr
	}

	return data, hotinfoStr, nil
}
func checkAndUpdateRedis(ctx context.Context, redisClient *redis.Client, hashStr string) (bool, error) {
	value, err := redisClient.Get(ctx, redisKeyBilibiliRankHot).Result()
	if err == redis.Nil {
		return setRedisKey(ctx, redisClient, redisKeyBilibiliRankHot, hashStr)
	}

	if err != nil {
		return false, fmt.Errorf("bilibili_rank: 从 Redis 获取数据失败: %v", err)
	}

	if hashStr != value {
		return setRedisKey(ctx, redisClient, redisKeyBilibiliRankHot, hashStr)
	}

	return false, nil
}
func setRedisKey(ctx context.Context, redisClient *redis.Client, key, value string) (bool, error) {
	if err := redisClient.Set(ctx, key, value, 0).Err(); err != nil {
		return false, fmt.Errorf("bilibili_rank: 设置 Redis 键失败: %v", err)
	}
	return true, nil
}
func storeData(ctx context.Context, data []*BRank) {
	if err := model.Conn.Create(data).Error; err != nil {
		handleError(err, "bilibili_rank: 将新数据写入数据库失败")
		return
	}

	HotDataJson, err := json.Marshal(data)
	if err != nil {
		handleError(err, "bilibili_rank: 将热搜数据转换为JSON格式失败")
		return
	}

	if err := model.RedisClient.Set(ctx, redisKeyBilibiliRankHotData, HotDataJson, 24*time.Hour).Err(); err != nil {
		handleError(err, "bilibili_rank: 将最新热搜数据写入 Redis 失败")
	}
}
func handleError(err error, msg string) {
	logrus.Error(msg, err)
}
func Refresh() ([]BRank, error) {
	var bilibiliList []BRank
	ctx := context.Background()

	hotDataJson, err := model.RedisClient.Get(ctx, redisKeyBilibiliRankHotData).Result()
	if err == redis.Nil {
		logrus.Error("bilibili: 刷新 - Redis 中没有找到数据", err)
		return bilibiliList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("bilibili: 刷新 - 从 Redis 获取数据失败", err)
		return bilibiliList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err := json.Unmarshal([]byte(hotDataJson), &bilibiliList); err != nil {
		logrus.Error("bilibili: 刷新 - 反序列化 JSON 数据失败", err)
		return bilibiliList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	printRefreshedData(bilibiliList)
	return bilibiliList, nil
}
func printRefreshedData(data []BRank) {
	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range data {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.Tag)
	}
}
