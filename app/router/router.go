package router

import (
	"github.com/gin-gonic/gin"
	"spider_hot/app/middleware"
	"spider_hot/app/schedule/baidu"
	"spider_hot/app/schedule/bilibili"
	"spider_hot/app/schedule/bilibili_rank"
	"spider_hot/app/schedule/douyin"
	"spider_hot/app/schedule/gongren"
	"spider_hot/app/schedule/guangming"
	"spider_hot/app/schedule/hyper"
	"spider_hot/app/schedule/jiefang"
	"spider_hot/app/schedule/jiefangjunbao"
	"spider_hot/app/schedule/jingji"
	"spider_hot/app/schedule/juejin"
	"spider_hot/app/schedule/lol"
	"spider_hot/app/schedule/qingnian"
	"spider_hot/app/schedule/renmin"
	"spider_hot/app/schedule/taptap_android"
	"spider_hot/app/schedule/taptap_ios"
	"spider_hot/app/schedule/tengxun"
	"spider_hot/app/schedule/toutiao"
	"spider_hot/app/schedule/wangyi"
	"spider_hot/app/schedule/weibo"
	"spider_hot/app/schedule/zhihu"
)

func Router() {

	r := gin.Default()
	r.Use(middleware.Cors())
	r.Use(middleware.LogMiddleware())

	r.GET("/baidu", baidu.GetInfo)
	r.GET("/bilibili", bilibili.GetInfo)
	r.GET("/bilibilirank", bilibili_rank.GetInfo)
	r.GET("/douyin", douyin.GetInfo)
	r.GET("/gongren", gongren.GetInfo)
	r.GET("/guangming", guangming.GetInfo)
	r.GET("/hyper", hyper.GetInfo)
	r.GET("/jiefang", jiefang.GetInfo)
	r.GET("/junbao", jiefangjunbao.GetInfo)
	r.GET("/jingji", jingji.GetInfo)
	r.GET("/juejin", juejin.GetInfo)
	r.GET("/lol", lol.GetInfo)
	r.GET("/qingnian", qingnian.GetInfo)
	r.GET("/renmin", renmin.GetInfo)
	r.GET("/taptapand", taptap_android.GetInfo)
	r.GET("/taptapios", taptap_ios.GetInfo)
	r.GET("/tengxun", tengxun.GetInfo)
	r.GET("/toutiao", toutiao.GetInfo)
	r.GET("/wangyi", wangyi.GetInfo)
	r.GET("/weibo", weibo.GetInfo)
	r.GET("/zhihu", zhihu.GetInfo)

	if err := r.Run(":8080"); err != nil {
		panic("gin 启动失败！")
	}
}
