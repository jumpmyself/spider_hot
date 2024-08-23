package renmin

import (
	"bytes"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/goccy/go-json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"spider_hot/app/model"
	"spider_hot/app/tools"
	"strconv"
	"strings"
	"time"
)

func Run() {
	ticker := time.NewTicker(24 * time.Hour)
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

type PageInfo struct {
	Text string `json:"text"`
	Link string `json:"link"`
}

func GetLink() []PageInfo {
	currentData := time.Now().Format("2006-01/02")
	api := fmt.Sprintf("http://paper.people.com.cn/rmrb/html/%s/nbs.D110000renmrb_01.htm", currentData)

	// 发起get请求
	response, err := http.Get(api)
	if err != nil {
		logrus.Error("人民日报：http请求失败", err)
		return nil
	}
	defer response.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.Error("人民日报：读取响应内容失败", err)
		return nil
	}

	// 将html字符串加载到goquery文档中
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		logrus.Error("人民日报：将html字符串加载到goquery文档中失败", err)
		return nil
	}

	// 定义一个切片用于保存pageinfo的结构体对象
	var pagesInfo []PageInfo

	// 找到url和题目
	doc.Find(".swiper-container div.swiper-slide").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")
		text := s.Find("a").Text()
		// 判断是否包含广告关键字
		if strings.Contains(text, "广告") {
			return
		}
		// 实例化pageInfo结构体对象并存储到切片中
		page := PageInfo{
			Text: text,
			Link: link,
		}
		pagesInfo = append(pagesInfo, page)
	})
	// 返回板块名和链接列表
	return pagesInfo
}

func GetNews(link string) ([]string, []string) {
	currentData := time.Now().Format("2006-01/02")
	api := fmt.Sprintf("http://paper.people.com.cn/rmrb/html/%s/%s", currentData, link)

	// 发起get请求
	response, err := http.Get(api)
	if err != nil {
		logrus.Error("人民日报:http请求失败", err)
		return nil, nil
	}
	defer response.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.Error("人民日报:读取响应内容失败", err)
		return nil, nil
	}

	// 将html字符串加载到goquery文档中
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		logrus.Error("人民日报:将html字符串加载到goquery文档中错误", err)
		return nil, nil
	}

	// 定义切片保存标题和链接列表
	var titles []string
	var links []string
	doc.Find("div.news ul.news-list li").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")
		title := s.Find("a").Text()
		if strings.Contains(title, "责编") || strings.Contains(title, "图片报道") {
			return
		}

		// 将标题和链接存入切片
		titles = append(titles, title)
		links = append(links, link)
	})

	// 在函数末尾返回标题和链接列表
	return links, titles

}

func GetInfo(c *gin.Context) {
	// 获取板块名和链接列表
	links := GetLink()
	data := make([]*Renmin, 0)
	var hotinfoStr string

	// 遍历链接列表并处理
	for _, link := range links {
		// 获取新闻标题和链接列表
		urls, titles := GetNews(link.Link)
		fmt.Println("titles", titles)

		// 遍历标题和链接列表并存入数据库
		for i, title := range titles {
			currentDate := time.Now().Format("2006-01/02")
			fullUrl := fmt.Sprintf("http://paper.people.com.cn/rmrb/html/%s/%s", currentDate, urls[i])

			// 创建Renmin结构体对象并存入数据切片
			renMin := Renmin{
				UpdateVer:   time.Now().Unix(),
				Title:       title,
				URL:         fullUrl,
				Version:     link.Text,
				CreatedTime: time.Now(),
				UpdatedTime: time.Now(),
			}
			data = append(data, &renMin)
			hotinfoStr = title + fullUrl + hotinfoStr
		}
	}

	// 计算哈希值
	hashStr := tools.Sha256Hash(hotinfoStr)

	// 检查Redis中的哈希值
	value, err := model.RedisClient.Get(context.Background(), "renmin_hot").Result()
	if err == redis.Nil {
		// Redis中无数据，写入新的哈希值和热搜数据
		err = model.RedisClient.Set(context.Background(), "renmin_hot", hashStr, 0).Err()
		if err != nil {
			logrus.Error("人民日报:设置redis中哈希值失败", err)
			return
		}
		model.Conn.Create(data)

		// 存储最新的热搜数据到 Redis
		HotDataJson, err := json.Marshal(data)
		if err != nil {
			logrus.Error("人民日报：将热搜数据转换为JSON格式失败", err)
			return
		}
		err = model.RedisClient.Set(context.Background(), "renmin_hot_data", HotDataJson, 24*time.Hour).Err()
		if err != nil {
			logrus.Error("人民日报：将最新热搜数据写入 Redis 失败", err)
		}
	} else if err != nil {
		// 获取Redis中的哈希值出错
		logrus.Error("人民日报:获取redis哈希值失败", err)
	} else {
		// Redis中有数据，比较哈希值
		if hashStr != value {
			// 哈希值不一致，更新Redis中的哈希值和数据库中的数据
			err = model.RedisClient.Set(context.Background(), "renmin_hot", hashStr, 0).Err()
			if err != nil {
				logrus.Error("人民日报:更新redis哈希值失败", err)
			}
			err = model.Conn.Create(data).Error
			if err != nil {
				logrus.Error("人民日报:将数据保存到数据库失败", err)
			}

			// 存储最新的热搜数据到 Redis
			HotDataJson, err := json.Marshal(data)
			if err != nil {
				logrus.Error("人民日报：将热搜数据转换为JSON格式失败", err)
				return
			}
			err = model.RedisClient.Set(context.Background(), "renmin_hot_data", HotDataJson, 24*time.Hour).Err()
			if err != nil {
				logrus.Error("人民日报：将最新热搜数据写入 Redis 失败", err)
			}
		} else {
			// 哈希值一致，更新数据库中的数据时间戳
			var updataSlice []Renmin
			model.Conn.Where("update_ver = ?", time.Now().Unix()).Find(&updataSlice)
			for _, record := range updataSlice {
				record.UpdateVer = time.Now().Unix()
				record.UpdatedTime = time.Now()
				err = model.Conn.Save(&record).Error
				if err != nil {
					logrus.Error("人民日报:更新数据库数据失败", err)
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

func Refresh() []Renmin {
	var maxUpdateVer int64

	// 查询最大的 update_ver
	result := model.Conn.Model(&Renmin{}).Select("MAX(update_ver)").Scan(&maxUpdateVer)
	if result.Error != nil {
		logrus.Error("人民日报: 查询最大的 update_ver 失败", result.Error)
		return nil
	}

	// 查询所有 update_ver 为最大值的记录
	var renminList []Renmin
	result = model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&renminList)
	if result.Error != nil {
		logrus.Error("人民日报: 查询最大 update_ver 的记录失败", result.Error)
		return nil
	}

	// 打印查询结果
	fmt.Printf("Data with max update_ver (%d):\n", maxUpdateVer)
	return renminList
}
