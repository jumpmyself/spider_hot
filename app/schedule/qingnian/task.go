package qingnian

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/goccy/go-json"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"spider_hot/app/model"
	"spider_hot/app/tools"
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

type PageInfo struct {
	Text string `json:"text"`
	Link string `json:"link"`
}

func GetLink() []PageInfo {
	currentDate := time.Now().Format("2006-01/02")
	api := fmt.Sprintf("https://zqb.cyol.com/html/%s/nbs.D110000zgqnb_01.htm", currentDate)

	// 发起get请求
	response, err := http.Get(api)
	if err != nil {
		logrus.Error("中国青年报：http请求失败", err)
		return nil
	}
	defer response.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.Error("中国青年报：读取响应内容失败", err)
		return nil
	}

	// 将html字符串加载到goquery文档中
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		logrus.Error("中国青年报：将数据加载到goquery文档中失败", err)
		return nil
	}

	// 定义一个切片用于保存pageinfo的结构体对象
	var pagesInfo []PageInfo
	doc.Find("div.list_r div.l_c.l_c1 div#pageList ul li a#pageLink").Each(func(i int, s *goquery.Selection) {
		link := s.AttrOr("href", "")
		text := s.Text()

		// 实例化pageinfo结构体对象并存储到切片中
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
	currentDate := time.Now().Format("2006-01/02")
	api := fmt.Sprintf("https://zqb.cyol.com/html/%s/%s", currentDate, link)

	// 发起get请求
	response, err := http.Get(api)
	if err != nil {
		logrus.Error("中国青年报: 获取新闻列表失败", err)
		return nil, nil
	}
	defer response.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.Error("中国青年报: 读取响应内容失败", err)
		return nil, nil
	}

	// 将html字符串加载到goquery文档中
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		logrus.Error("中国青年报: 将数据加载到goquery文档中失败", err)
		return nil, nil
	}

	// 定义切片保存标题和链接
	var titles []string
	var links []string

	// 选择不同版的不同内容的标题和链接
	doc.Find("div#ozoom div.list_t div.list_l div.l_c div#titleList ul li").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")
		title := strings.TrimSpace(s.Find("a").Text())

		// 判断板块名是否有广告关键字，如果有就跳过
		if !strings.Contains(title, "图片新闻") {
			// 将标题和链接存入切片
			titles = append(titles, title)
			links = append(links, link)
		}
	})

	// 在函数末尾返回标题和链接列表
	return titles, links
}

func GetInfo(c *gin.Context) {
	// 获取板块名和链接列表
	links := GetLink()
	data := make([]*QingNian, 0)
	var hotinfoStr string

	// 遍历链接列表并处理
	for _, link := range links {
		// 获取新闻标题和链接列表
		titles, urls := GetNews(link.Link)

		// 遍历标题和链接列表并存入数据库
		for i, title := range titles {
			currentDate := time.Now().Format("2006-01/02")
			fullUrl := fmt.Sprintf("https://zqb.cyol.com/html/%s/%s", currentDate, urls[i])

			// 创建people结构体对象并存入数据切片
			qingnian := QingNian{
				UpdateVer:   time.Now().Unix(),
				Title:       title,
				URL:         fullUrl,
				Version:     link.Text,
				CreatedTime: time.Now(),
				UpdatedTime: time.Now(),
			}
			data = append(data, &qingnian)
			hotinfoStr = title + fullUrl + hotinfoStr
		}
	}

	// 计算哈希值
	hashStr := tools.Sha256Hash(hotinfoStr)

	// 获取 Redis 中保存的哈希值
	value, err := model.RedisClient.Get(context.Background(), "qingnian_hot").Result()
	if err == redis.Nil {
		// Redis 中不存在该键值对，第一次存入数据
		logrus.Error("中国青年报: redis中没有数据", err)
		err = model.RedisClient.Set(context.Background(), "qingnian_hot", hashStr, 0).Err()
		if err != nil {
			logrus.Error("中国青年报: 设置redis数据失败", err)
			return
		}
		// 存储数据到数据库
		model.Conn.Create(data)

		// 存储最新的热搜数据到 Redis
		HotDataJson, err := json.Marshal(data)
		if err != nil {
			logrus.Error("中国青年报: 将热搜数据转换为JSON格式失败", err)
			return
		}
		err = model.RedisClient.Set(context.Background(), "qingnian_hot_data", HotDataJson, 24*time.Hour).Err()
		if err != nil {
			logrus.Error("中国青年报: 将最新热搜数据写入 Redis 失败", err)
		}
	} else if err != nil {
		logrus.Error("中国青年报: 从 Redis 获取数据失败", err)
	} else {
		// Redis 中存在数据，比较哈希值是否变化
		if hashStr != value {
			err = model.RedisClient.Set(context.Background(), "qingnian_hot", hashStr, 0).Err()
			if err != nil {
				logrus.Error("中国青年报: 设置redis数据失败", err)
			}
			err = model.Conn.Create(data).Error
			if err != nil {
				logrus.Error("中国青年报: 将数据存到数据库失败", err)
			}
			// 存储最新的热搜数据到 Redis
			HotDataJson, err := json.Marshal(data)
			if err != nil {
				logrus.Error("中国青年报: 将热搜数据转换为JSON格式失败", err)
				return
			}
			err = model.RedisClient.Set(context.Background(), "qingnian_hot_data", HotDataJson, 24*time.Hour).Err()
			if err != nil {
				logrus.Error("中国青年报: 将最新热搜数据写入 Redis 失败", err)
			}
		} else {
			// 哈希值未变化，更新数据的版本号和更新时间
			var maxUpdateVer int64
			var updateSlice []QingNian

			// 查询当前最大的 update_ver
			model.Conn.Model(&QingNian{}).Select("MAX(update_ver) as max_update_ver").Scan(&maxUpdateVer)

			// 查询所有 update_ver 为最大值的记录
			model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&updateSlice)

			// 更新数据的版本号和更新时间
			for _, record := range updateSlice {
				record.UpdateVer = time.Now().Unix()
				record.UpdatedTime = time.Now()
				err := model.Conn.Save(&record).Error
				if err != nil {
					logrus.Error("中国青年报: 更新qingnian_hot数据失败", err)
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

func Refresh() []QingNian {
	var maxUpdateVer int64

	// 查询最大的 update_ver
	result := model.Conn.Model(&QingNian{}).Select("MAX(update_ver) as max_update_ver").Scan(&maxUpdateVer)
	if result.Error != nil {
		logrus.Error("中国青年报: 查询最大的 update_ver 失败", result.Error)
		return nil
	}

	// 查询所有 update_ver 为最大值的记录
	var qingnianList []QingNian
	result = model.Conn.Where("update_ver = ?", maxUpdateVer).Find(&qingnianList)
	if result.Error != nil {
		logrus.Error("中国青年报: 查询所有 update_ver 为最大值的记录失败", result.Error)
		return nil
	}

	// 打印查询结果
	fmt.Printf("Data with max update_ver (%d):\n", maxUpdateVer)
	return qingnianList
}
