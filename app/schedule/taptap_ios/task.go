package taptap_ios

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"spider_hot/app/model"
	"spider_hot/app/tools"
	"strconv"
	"time"
)

const api = "https://www.taptap.cn/webapiv2/app-top/v2/hits?platform=ios&type_name=hot&X-UA=V%3D1%26PN%3DWebApp%26LANG%3Dzh_CN%26VN_CODE%3D102%26LOC%3DCN%26PLT%3DPC%26DS%3DiOS%26UID%3D6ce81371-f373-4937-9350-9fda7c6503d3%26OS%3DWindows%26OSV%3D10%26DT%3DPC"

// Run 定时任务，每5分钟获取一次数据
func Run() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			GetInfo(nil)
		}
	}
}

// Do 手动触发数据获取
func Do() {
	GetInfo(nil)
}

// getFirstTag 获取标签列表中的第一个标签
func getFirstTag(tagList []Tags) string {
	if len(tagList) > 0 {
		return tagList[0].Value
	}
	return ""
}

// GetInfo 获取TapTap iOS热门数据并存储
func GetInfo(c *gin.Context) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", api, nil)
	if err != nil {
		logrus.Error("TapTap_ios:Error creating request:", err)
		return
	}
	setRequestHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		logrus.Error("TapTap_ios:Error sending request:", err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.Error("TapTap_ios:Error reading response body:", err)
		return
	}

	var ret Ret
	err = json.Unmarshal(body, &ret)
	if err != nil {
		logrus.Error("TapTap_ios:Error unmarshalling response body:", err)
		return
	}

	data := parseResponse(ret)

	hotinfoStr := generateHotInfoString(data)
	hashStr := tools.Sha256Hash(hotinfoStr)

	cacheKey := "TapTapIos_hot"
	value, err := model.RedisClient.Get(context.Background(), cacheKey).Result()
	if err != nil && err != redis.Nil {
		logrus.Error("TapTapIos:Failed to get value from Redis:", err)
	}

	if value == "" || value != hashStr {
		updateDataInDatabaseAndCache(data, hashStr, cacheKey)
	} else {
		updateExistingData()
	}
	// 构建返回给前端的部分数据
	var partialData []model.HtmlData
	for _, item := range data {
		partialData = append(partialData, model.HtmlData{
			Title: item.Title,
			Url:   item.Url,
			Hot:   strconv.FormatInt(item.PlayTotal, 10),
		})
	}

	// 返回部分数据给前端
	c.JSON(http.StatusOK, tools.ECode{
		Message: "",
		Data:    partialData,
	})
}

// setRequestHeaders 设置请求头
func setRequestHeaders(request *http.Request) {
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	request.Header.Add("Cookie", "web_app_uuid=6ce81371-f373-4937-9350-9fda7c6503d3; apk_download_url_postfix=/seo-bing_index; XSRF-TOKEN=vhakkd8mfwsqkh1m3vdw; Hm_lvt_536e39e029b26af67aecbc444cdbd996=1705125030,1705287396; _clck=p4hbt9%7C2%7Cfif%7C0%7C1454; locale=zh_CN; tap_theme=light; _gid=GA1.2.1336325194.1705287923; _ga=GA1.1.53696830.1703473690; Hm_lpvt_536e39e029b26af67aecbc444cdbd996=1705298863; acw_tc=ac11000117053007345444115e005c190d8f03a9dda3eafa725decaf7c2177; ssxmod_itna=iq+xuDcD0GqYwuDlfYoGk7titDCSD9CCx3gGQrD05x+eGzDAxn40iDto=NnE7fCbrvdfFm8wqPjefmDbpkD6rpbsemDB3DEx0=qtmnr4GGUxBYDQxAYDGDDPDo2PD1D3qDkD7h6CMy1qGWDm4kDWPDYxDrjOKDRxi7DDHQ8x07DQyk3DDz3BE4kDGiPi7+8c71atvxobDxjxG1H40HYQL4jPo69f0zoRfL4f+ODlK2DC91c25Ha4GdXc5z48RDfB0DbBwq+hDdKii5kQG4KDb9Uf0PYh04G2=xzUrltWRai7h9DDalHhUDD=; ssxmod_itna2=iq+xuDcD0GqYwuDlfYoGk7titDCSD9CCx3gG7DnF8vT5Dskk3DLnciTYbGlFqn4x8tDZ3Y681Y7nx6sqQjoypiKoDlbtqKxgCiUxQ=BW57wKR/dfFwqm/qWyqvPfUTUz0V/M4+CtQ4hUBLIg/2hZQ7QQtM=e8+DDauGI4SdNYYjkRrU6SYdQQuGMD=QiFlLgFGKiMhGMb1ozA2XpthKEBoK8+ZpOx1LYEpuHlqbDP/UYy6904P7spf96+GHueFAgY8Y9G+Kg4M+bM/tp1F7Y7dbhQAiva7j9Oqmj9zUr5f/dz2h8bst4Ypf=5EmL0B4tc6jaA6YYR7KijvQ2KqQAoIE7DeP2KC7ECjmsjhSx+MBDQlxogDfGeueen0bQ4vkxvjmDIAGZj6/2mMCeqPf4xRI0011e+l+dW5jO3oI3vrNXY5mcx5afAaoem60jhSQ0GpWvmDcrbv7XX/Iq0IOW9F4+3c4CEGNWmsia7UpGnNVnxKQPQGhlBvGYmxeK/iIROP=8UuWoN0IvY74G=l/YReiYM0xdioxcxHAZayztOphHqjvxn3LmKe8Yo4oqyTqm=7Cmi+dLBmbvMsQ=sBPYycGViQ4e6RMj6PGDolF5sykv7cpqg3xEzQZYp4DQFe8GXf2juIYKb4t5=4Nve+b//yKVo8+BF9mkVHt2pCDDFqD+EDxD; _clsk=vthhpz%7C1705300917137%7C1%7C0%7Cy.clarity.ms%2Fcollect; _ga_6G9NWP07QM=GS1.1.1705300921.6.0.1705300921.0.0.0")
}

// parseResponse 解析API响应并生成TaptapIos数据列表
func parseResponse(ret Ret) []*TaptapIos {
	data := make([]*TaptapIos, 0)
	for _, result := range ret.Data.List {
		url := "https://www.taptap.cn/app/" + strconv.Itoa(int(result.App.Id))
		tmp := TaptapIos{
			UpdateVer:   time.Now().Unix(),
			Title:       result.App.Title,
			Tag:         getFirstTag(result.App.Tags),
			Url:         url,
			Score:       result.App.Stat.Rating.Score,
			PlayTotal:   result.App.Stat.PlayTotal,
			RecText:     result.App.RecText,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
		}
		data = append(data, &tmp)
	}
	return data
}

// generateHotInfoString 生成热搜信息的哈希字符串
func generateHotInfoString(data []*TaptapIos) string {
	var hotinfoStr string
	for _, item := range data {
		hotinfoStr = item.Title + item.Url + hotinfoStr
	}
	return hotinfoStr
}

// updateDataInDatabaseAndCache 更新数据库和缓存中的数据
func updateDataInDatabaseAndCache(data []*TaptapIos, hashStr, cacheKey string) {
	ctx := context.Background()

	// 更新 Redis 中的哈希值
	err := model.RedisClient.Set(ctx, cacheKey, hashStr, 0).Err()
	if err != nil {
		logrus.Error("TapTapIos:Failed to set hash value in Redis:", err)
		return
	}

	// 将新的数据插入数据库
	err = model.Conn.Create(&data).Error
	if err != nil {
		logrus.Error("TapTapIos:Failed to insert data into database:", err)
		return
	}

	// 存储最新的热搜数据到 Redis
	hotDataJson, err := json.Marshal(data)
	if err != nil {
		logrus.Error("TapTapIos:Failed to marshal hot data to JSON:", err)
		return
	}

	err = model.RedisClient.Set(ctx, "taptapios_hot_data", hotDataJson, 24*time.Hour).Err()
	if err != nil {
		logrus.Error("TapTapIos:Failed to set hot data in Redis:", err)
	}
}

// updateExistingData 更新已有数据的时间戳
func updateExistingData() {
	var maxUpdateVer int64

	// 查询当前数据中最大的 UpdateVer
	err := model.Conn.Model(&TaptapIos{}).Select("MAX(update_ver) as max_update_ver").Scan(&maxUpdateVer).Error
	if err != nil {
		logrus.Error("TapTapIos:Failed to get max update_ver from database:", err)
		return
	}

	// 查询所有 UpdateVer 等于最大值的数据
	var updateSlice []TaptapIos
	err = model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice).Error
	if err != nil {
		logrus.Error("TapTapIos:Failed to get records with max update_ver:", err)
		return
	}

	// 更新这些记录的 UpdateVer 和 UpdatedTime
	for _, record := range updateSlice {
		record.UpdateVer = time.Now().Unix()
		record.UpdatedTime = time.Now()

		err := model.Conn.Save(&record).Error
		if err != nil {
			logrus.Error("TapTapIos:Failed to update record:", err)
			return
		}
	}
}

// Refresh 刷新最新的数据
func Refresh() []TaptapIos {
	var maxUpdateVer int64

	// 查询当前数据中最大的 UpdateVer
	err := model.Conn.Model(&TaptapIos{}).Select("MAX(update_ver) as max_update_ver").Scan(&maxUpdateVer).Error
	if err != nil {
		logrus.Error("TapTapIos:Failed to get max update_ver from database:", err)
		return nil
	}

	// 查询所有 UpdateVer 等于最大值的数据
	var taptapiosList []TaptapIos
	err = model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&taptapiosList).Error
	if err != nil {
		logrus.Error("TapTapIos:Failed to get records with max update_ver:", err)
		return nil
	}

	fmt.Printf("Data with max update_ver (%d):\n", maxUpdateVer)
	return taptapiosList
}
