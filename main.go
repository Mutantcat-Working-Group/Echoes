package main

import (
	"com.mutantcat.echoes/lifecycle"
	"com.mutantcat.echoes/router"
	"flag"
	"time"
)

func main() {
	//-server_name                服务器名称
	server_name := flag.String("server_name", "", "服务器名称")
	//-daily_time				  每天通知的的时间
	daily_time := flag.String("daily_time", "09:30", "每天通知信息的时间")
	//-interval_time              每次自检的时间间隔（秒）
	interval_time := flag.Int("interval_time", 30, "每次自检的时间间隔（秒）")
	//-loadavg_max_percent        负载率告警阈值最（百分比）
	loadavg_max_percent := flag.Float64("loadavg_max_percent", 70, "负载率告警阈值最（百分比）")
	//-mem_used_percent           内存告警阈值（百分比）
	mem_used_percent := flag.Float64("mem_used_percent", 90, "内存告警阈值（百分比）")
	//-cpu_used_percent           CPU使用率告警阈值（百分比）
	cpu_used_percent := flag.Float64("cpu_used_percent", 90, "CPU使用率告警阈值（百分比）")
	//-pin_enable                 探针模式是否开启(0/1)
	pin_enable := flag.Int("pin_enable", 1, "探针模式是否开启(1/0)")
	//-port                       主动服务的端口
	port := flag.String("port", "9966", "主动服务的端口")
	//-notice_mod                 通知方式
	notice_mod := flag.String("notice_mod", "", "通知方式")
	//-ding_token                 群钉钉机器人token
	token := flag.String("token", "", "群钉钉机器人token")
	//-ding_secret                群钉钉机器人secret
	secret0 := flag.String("secret0", "", "群钉钉机器人secret")
	//-smtp_user                  mail通知的发件账号（仅mail模式需要）
	smtp_user := flag.String("smtp_user", "", "mail通知的发件账号")
	//-smtp_from                  mail通知的发件来源邮箱（仅mail模式需要）
	smtp_from := flag.String("smtp_from", "", "mail通知的发件来源邮箱")
	//-smtp_to                    mail通知的收件邮箱（仅mail模式需要）
	smtp_to := flag.String("smtp_to", "", "mail通知的收件邮箱")
	//-help                       是否帮助模式（0/1）
	help := flag.Int("help", 0, "帮助信息模式(0/1)")
	flag.Parse() //解析命令行参数

	if help != nil && *help == 1 {
		flag.PrintDefaults() //输出帮助信息
		return
	}

	if *server_name == "" {
		// 获得标识当前时间的字符串
		*server_name = "echoes_server_" + time.Now().String()
	}

	if (notice_mod == nil || *notice_mod == "") && (pin_enable == nil || *pin_enable == 0) {
		// 任何模式都没开启 直接关闭程序
		return
	}

	gin := lifecycle.InitGin()
	lifecycle.RegisterRouter(gin, &router.InfoRouter{})

	// 如果开启了pin模式但是没开启通知模式 使用pin模式阻塞进程
	if *pin_enable == 1 && (notice_mod == nil || *notice_mod == "") {
		lifecycle.StartGin(gin, *port)
	}

	// 如果开启了通知模式没开启pin模式 使用通知模式阻塞进程
	if *pin_enable == 0 && (notice_mod != nil && *notice_mod != "") {
		lifecycle.RegisterWarning(*server_name, *interval_time, *loadavg_max_percent, *mem_used_percent, *cpu_used_percent, *notice_mod, *token, *secret0, *smtp_user, *smtp_from, *smtp_to)
		lifecycle.RegisterNotice(*server_name, *daily_time, *notice_mod, *token, *secret0, *smtp_user, *smtp_from, *smtp_to)
	}

	// 如果同时开启了通知模式和pin模式 使用通知模式阻塞进程
	if *pin_enable == 1 && (notice_mod != nil && *notice_mod != "") {
		go lifecycle.StartGin(gin, *port)
		lifecycle.RegisterWarning(*server_name, *interval_time, *loadavg_max_percent, *mem_used_percent, *cpu_used_percent, *notice_mod, *token, *secret0, *smtp_user, *smtp_from, *smtp_to)
		lifecycle.RegisterNotice(*server_name, *daily_time, *notice_mod, *token, *secret0, *smtp_user, *smtp_from, *smtp_to)
	}
}
