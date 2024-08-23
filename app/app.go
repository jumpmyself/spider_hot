package app

import (
	"spider_hot/app/model"
	"spider_hot/app/router"
	"spider_hot/app/schedule"
	"spider_hot/app/tools"
)

func Start() {
	tools.InitFile("app/log/", "")

	model.NewMySql()
	model.Redis()
	defer func() {
		model.RedisClose()
		model.Close()
	}()
	//爬虫定时器启动
	taskRun()
	//服务器必须最后启动
	router.Router()
}

func taskRun() {
	schedule.Run()
}
