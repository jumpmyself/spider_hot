package taptap_android

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

const api = "https://www.taptap.cn/webapiv2/app-top/v2/hits?platform=android&type_name=hot&X-UA=V%3D1%26PN%3DWebApp%26LANG%3Dzh_CN%26VN_CODE%3D102%26LOC%3DCN%26PLT%3DPC%26DS%3DiOS%26UID%3D6ce81371-f373-4937-9350-9fda7c6503d3%26OS%3DWindows%26OSV%3D10%26DT%3DPC"

func Run() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		GetInfo(nil)
	}
}

func Do() {
	GetInfo(nil)
}

func getFirstTag(tagList []Tags) string {
	if len(tagList) > 0 {
		return tagList[0].Value
	}
	return ""
}

func GetInfo(c *gin.Context) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", api, nil)
	if err != nil {
		logrus.Error("TapTap_android: Error creating request:", err)
		return
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	request.Header.Add("Cookie", "web_app_uuid=6ce81371-f373-4937-9350-9fda7c6503d3; ...")

	response, err := client.Do(request)
	if err != nil {
		logrus.Error("TapTap_android: Error sending request:", err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.Error("TapTap_android: Error reading response body:", err)
		return
	}

	var ret Ret
	err = json.Unmarshal(body, &ret)
	if err != nil {
		logrus.Error("TapTap_android: Error unmarshalling response body:", err)
		return
	}

	data := make([]*TaptapAndroid, 0)
	var hotinfoStr string
	ver := time.Now().Unix()
	for _, result := range ret.Data.List {
		url := "https://www.taptap.cn/app/" + strconv.Itoa(int(result.App.Id))
		tmp := TaptapAndroid{
			UpdateVer:   ver,
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
		hotinfoStr = result.App.Title + url + hotinfoStr
	}

	hashStr := tools.Sha256Hash(hotinfoStr)
	value, err := model.RedisClient.Get(context.Background(), "TapTapAndroid_hot").Result()
	if err != nil && err != redis.Nil {
		logrus.Error("TapTapAndroid: Failed to get value from Redis:", err)
		return
	}

	if value == "" || value != hashStr {
		err = model.RedisClient.Set(context.Background(), "TapTapAndroid_hot", hashStr, 0).Err()
		if err != nil {
			logrus.Error("TapTapAndroid: Failed to set value in Redis:", err)
			return
		}

		err = model.Conn.Create(&data).Error
		if err != nil {
			logrus.Error("TapTap_android: Failed to save data to database:", err)
			return
		}

		HotDataJson, err := json.Marshal(data)
		if err != nil {
			logrus.Error("TapTap_android: Failed to convert hot data to JSON:", err)
			return
		}
		err = model.RedisClient.Set(context.Background(), "taptapandroid_hot_data", HotDataJson, 24*time.Hour).Err()
		if err != nil {
			logrus.Error("TapTap_android: Failed to save hot data to Redis:", err)
		}
		return
	}

	var maxUpdateVer int64
	var updateSlice []TaptapAndroid
	err = model.Conn.Model(&TaptapAndroid{}).Select("MAX(update_ver)").Scan(&maxUpdateVer).Error
	if err != nil {
		logrus.Error("TapTap_android: Failed to get max update_ver:", err)
		return
	}

	model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice)
	for _, record := range updateSlice {
		record.UpdateVer = time.Now().Unix()
		record.UpdatedTime = time.Now()
		err = model.Conn.Save(&record).Error
		if err != nil {
			logrus.Error("TapTap_android: Failed to update record:", err)
			return
		}
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

func Refresh() []TaptapAndroid {
	var maxUpdateVer int64

	// 查询最大的 update_ver
	result := model.Conn.Model(&TaptapAndroid{}).Select("MAX(update_ver)").Scan(&maxUpdateVer)
	if result.Error != nil {
		logrus.Error("TapTap_android: Failed to get max update_ver:", result.Error)
		return nil
	}

	// 查询所有 update_ver 为最大值的记录
	var taptapandroidList []TaptapAndroid
	result = model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&taptapandroidList)
	if result.Error != nil {
		logrus.Error("TapTap_android: Failed to get records with max update_ver:", result.Error)
		return nil
	}

	fmt.Printf("Data with max update_ver (%d):\n", maxUpdateVer)
	return taptapandroidList
}
