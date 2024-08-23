package tengxun

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
	client := &http.Client{}
	request, err := createRequest()
	if err != nil {
		logrus.Error("TENGXUN: 创建HTTP请求失败:", err)
		return
	}

	resp, err := client.Do(request)
	if err != nil {
		logrus.Error("TENGXUN: HTTP请求失败:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error("TENGXUN: 读取响应信息失败:", err)
		return
	}

	var response Response
	if err = json.Unmarshal(body, &response); err != nil {
		logrus.Error("TENGXUN: 解码JSON失败:", err)
		return
	}

	data, hashStr := processResponseData(response)
	updateDataInRedisAndDB(data, hashStr)
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

func createRequest() (*http.Request, error) {
	request, err := http.NewRequest("GET", viper.GetString("hot_api.tengxun"), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	request.Header.Add("Cookie", "eas_sid=t1X6d9g1A066f0s8w8i1F667X5; pgv_pvid=6565687608; fqm_pvqid=52ef1eb5-8a6f-4225-892a-28f4159266d3; RK=G9dxSYYZMH; ptcz=ab896a6eff7abb6a845f88d684c1e74b359e745f26561f9dcca7ed629e9161bb; qq_domain_video_guid_verify=5cd8cff73065f805; _clck=248b6v|1|fh2|0; pac_uid=0_de0fac31bcdbe; iip=0; _qimei_uuid42=1810909212310040746e10d6df61ae6e145e57681f; _qimei_fingerprint=2ff9e6b26248f3363f4963ad3bc30d9d; _qimei_q36=; _qimei_h38=8f956087746e10d6df61ae6e0200000b818109; lcad_o_minduid=H8mSY6u63HVd_odIIGDWfQlCTzEmfxhK; lcad_appuser=2B14C47E068A38AF; logintype=1; wap_refresh_token=76_eNagPIhJMQbJoHr_tBSuaaRNo_RLuyO9x40Q0gyDNtUs2nP1AHBX3t52KbvgUgP3Ui04I70XDRuoV532jUOVTUfQj4Oar3-5nZJbycsRYHo; wap_encrypt_logininfo=ASuZHXPxJsxaHE13GyDl4zKJmmb%2B8%2BhuFltjqfhXW18%2BaxvOXYTThIvNY%2Fm%2BugBRQO3SHD9H1nCCgEhGFIx%2BoOJ6pTP9ap4JZ%2BOkQnkiKLBX; pgv_info=ssid=s8322238889; ts_last=news.qq.com/topboard.shtml; ts_refer=www.bing.com/; ts_uid=7516076138; lcad_Lturn=577; lcad_LKBturn=976; lcad_LPVLturn=13; lcad_LPLFturn=212; lcad_LPSJturn=618; lcad_LBSturn=267; lcad_LVINturn=435; lcad_LDERturn=747")
	return request, nil
}

func processResponseData(response Response) ([]*TengXun, string) {
	realtimeData := response.Hotlist
	data := make([]*TengXun, 0)
	now := time.Now().Unix()
	var hotinfoStr string

	for _, list := range realtimeData {
		tmp := TengXun{
			UpdateVer:   now,
			Title:       list.Title,
			Time:        list.Time,
			Url:         list.ShareUrl,
			Hot:         list.ReadCount,
			Source:      list.Source,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, &tmp)
		hotinfoStr += list.Title + list.ShareUrl
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	return data, hashStr
}

func updateDataInRedisAndDB(data []*TengXun, hashStr string) {
	ctx := context.Background()

	value, err := model.RedisClient.Get(ctx, "tengxun_hot").Result()
	if err != nil && err != redis.Nil {
		logrus.Error("TENGXUN: 从Redis获取数据失败:", err)
		return
	}

	if err == redis.Nil || hashStr != value {
		if err = model.RedisClient.Set(ctx, "tengxun_hot", hashStr, 0).Err(); err != nil {
			logrus.Error("TENGXUN: 更新Redis数据失败:", err)
			return
		}

		if err = model.Conn.Create(data).Error; err != nil {
			logrus.Error("TENGXUN: 保存数据到数据库失败:", err)
			return
		}

		hotDataJson, err := json.Marshal(data)
		if err != nil {
			logrus.Error("TENGXUN: 将热搜数据转换为JSON格式失败:", err)
			return
		}

		if err = model.RedisClient.Set(ctx, "tengxun_hot_data", hotDataJson, 24*time.Hour).Err(); err != nil {
			logrus.Error("TENGXUN: 将最新热搜数据写入Redis失败:", err)
			return
		}
	} else {
		updateExistingData()
	}
}

func updateExistingData() {
	var maxUpdateVer int64

	if err := model.Conn.Model(&TengXun{}).Select("MAX(update_ver) AS max_update_ver").Scan(&maxUpdateVer).Error; err != nil {
		logrus.Error("TENGXUN: 获取数据库中最大update_ver失败:", err)
		return
	}

	var updateSlice []TengXun
	if err := model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice).Error; err != nil {
		logrus.Error("TENGXUN: 获取具有最大update_ver的记录失败:", err)
		return
	}

	now := time.Now()
	for _, record := range updateSlice {
		record.UpdateVer = now.Unix()
		record.UpdatedTime = now

		if err := model.Conn.Save(&record).Error; err != nil {
			logrus.Error("TENGXUN: 更新记录失败:", err)
			return
		}
	}
}

func Refresh() ([]TengXun, error) {
	var tengxunList []TengXun
	ctx := context.Background()

	hotDataJson, err := model.RedisClient.Get(ctx, "tengxun_hot_data").Result()
	if err == redis.Nil {
		logrus.Error("TENGXUN: 刷新 - Redis中没有找到数据")
		return tengxunList, fmt.Errorf("no data found in Redis")
	} else if err != nil {
		logrus.Error("TENGXUN: 刷新 - 从Redis获取数据失败", err)
		return tengxunList, fmt.Errorf("failed to get data from Redis: %v", err)
	}

	if err = json.Unmarshal([]byte(hotDataJson), &tengxunList); err != nil {
		logrus.Error("TENGXUN: 刷新 - 反序列化JSON数据失败", err)
		return tengxunList, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	// 打印查询结果
	fmt.Printf("Refreshed data from Redis:\n")
	for _, item := range tengxunList {
		fmt.Printf("Title: %s, Url: %s, Hot: %s\n", item.Title, item.Url, item.Hot)
	}

	return tengxunList, nil
}
