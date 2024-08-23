package schedule

import (
	"fmt"
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

var Task []func()

func Run() {
	Task = []func(){
		baidu.Run,
		bilibili.Run,
		bilibili_rank.Run,
		douyin.Run,
		gongren.Run,
		guangming.Run,
		hyper.Run,
		jiefang.Run,
		jiefangjunbao.Run,
		jingji.Run,
		juejin.Run,
		lol.Run,
		qingnian.Run,
		renmin.Run,
		taptap_android.Run,
		taptap_ios.Run,
		tengxun.Run,
		toutiao.Run,
		wangyi.Run,
		weibo.Run,
		zhihu.Run,
	}

	//设置waitgroup的计数器为并发次数
	for i := 0; i < len(Task); i++ {
		go func(index int) {
			Task[index]()
			//执行并发任务
			fmt.Printf("并发任务%d 执行\n", index)
			//模拟任务执行时间
			//这里可以替换为你的实际任务执行逻辑
			//例如，调用一个函数或执行一段代码
			//...
			//完成并发任务，减少waitgroup的计数器
		}(i)
	}
	//等待所有的并发任务执行完成

	fmt.Println("所有并发任务已完成")
}
