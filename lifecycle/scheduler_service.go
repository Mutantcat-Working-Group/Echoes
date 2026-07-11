package lifecycle

import (
	"com.mutantcat.echoes/scheduler"
	"com.mutantcat.echoes/speaker"
	"com.mutantcat.echoes/status"
	"fmt"
	"github.com/blinkbean/dingtalk"
	"github.com/shirou/gopsutil/load"
	"runtime"
)

func RegisterWarning(server_name string, interval_time int, loadavg_max_percent float64, mem_used_percent float64, cpu_used_percent float64, notice_mod string, args ...string) {
	switch notice_mod {
	case "dingbot":
		if len(args) < 2 {
			fmt.Println("dingbot 模式需要 token 和 secret0 两个参数")
			return
		}
		bot := dingtalk.InitDingTalkWithSecret(args[0], args[1])
		f := DingBotWarning(server_name, bot, loadavg_max_percent, mem_used_percent, cpu_used_percent)
		scheduler.IntervalDo(interval_time, f)
	case "mail":
		cfg, err := parseMailConfig(args)
		if err != nil {
			fmt.Println("mail 配置错误:", err)
			return
		}
		f := MailWarning(server_name, cfg, loadavg_max_percent, mem_used_percent, cpu_used_percent)
		scheduler.IntervalDo(interval_time, f)
	case "jiang":
		if len(args) < 1 || args[0] == "" {
			fmt.Println("jiang 模式需要 token（SendKey）参数")
			return
		}
		c := speaker.NewJiang(args[0])
		f := JiangWarning(server_name, c, loadavg_max_percent, mem_used_percent, cpu_used_percent)
		scheduler.IntervalDo(interval_time, f)
	default:
		fmt.Println("未知的通知方式")
		return
	}
}

func RegisterNotice(server_name string, daily_time, notice_mod string, args ...string) {
	switch notice_mod {
	case "dingbot":
		if len(args) < 2 {
			fmt.Println("dingbot 模式需要 token 和 secret0 两个参数")
			return
		}
		bot := dingtalk.InitDingTalkWithSecret(args[0], args[1])
		go bot.SendTextMessage("您的服务器\"" + server_name + "\"已开启通知和告警服务。")
		f := DingBotNotice(server_name, bot)
		scheduler.DayDo(daily_time, f)
	case "mail":
		cfg, err := parseMailConfig(args)
		if err != nil {
			fmt.Println("mail 配置错误:", err)
			return
		}
		go func() {
			_ = speaker.SendMail(cfg, "服务器\""+server_name+"\"已开启通知和告警服务", "您的服务器\""+server_name+"\"已开启回声通知和告警服务。")
		}()
		f := MailNotice(server_name, cfg)
		scheduler.DayDo(daily_time, f)
	case "jiang":
		if len(args) < 1 || args[0] == "" {
			fmt.Println("jiang 模式需要 token（SendKey）参数")
			return
		}
		c := speaker.NewJiang(args[0])
		go func() {
			_ = c.Send("服务器\""+server_name+"\"已开启通知和告警服务", "您的服务器\""+server_name+"\"已开启回声通知和告警服务。")
		}()
		f := JiangNotice(server_name, c)
		scheduler.DayDo(daily_time, f)
	default:
		fmt.Println("未知的通知方式")
	}
}

// parseMailConfig 从 args 中解析邮件配置
// args[0]=token, args[1]=secret0, args[2]=smtp_user, args[3]=smtp_from, args[4]=smtp_to
func parseMailConfig(args []string) (speaker.MailConfig, error) {
	if len(args) < 5 {
		return speaker.MailConfig{}, fmt.Errorf("mail 模式需要 token(Host:Port)、secret0(Password)、smtp_user、smtp_from、smtp_to 五个参数")
	}
	cfg := speaker.MailConfig{
		Host:     args[0],
		Password: args[1],
		User:     args[2],
		From:     args[3],
		To:       args[4],
	}
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" || cfg.To == "" {
		return speaker.MailConfig{}, fmt.Errorf("mail 模式 Host/User/Password/To 不能为空")
	}
	return cfg, nil
}

func DingBotWarning(server_name string, bot *dingtalk.DingTalk, loadavg_max_percent, mem_used_percent, cpu_used_percent float64) func() {
	return func() {
		info := status.GetSysInfo()
		avg, _ := load.Avg()
		cpuNum := runtime.NumCPU()
		loadavg_max := float64(cpuNum) * loadavg_max_percent / 100
		if avg.Load1 > loadavg_max || avg.Load5 > loadavg_max || avg.Load15 > loadavg_max {
			bot.SendMarkDownMessage("服务器告警", "⚠️<font color=\"#d30c0c\">【警告】</font>您的云服务器\""+server_name+"\"当前负载过高，当前负载为<font color=\"#d30c0c\">"+fmt.Sprintf("%.2f", avg.Load1)+"</font>，请及时检查系统是否存在问题。")
		}
		if info.MemUsedPercent > mem_used_percent {
			bot.SendMarkDownMessage("服务器告警", "⚠️<font color=\"#d30c0c\">【警告】</font>您的云服务器\""+server_name+"\"当前内存使用率为<font color=\"#d30c0c\">"+fmt.Sprintf("%.2f", info.MemUsedPercent)+"%</font>，请及时检查系统是否存在问题。")
		}
		if info.CpuUsedPercent > cpu_used_percent {
			bot.SendMarkDownMessage("服务器告警", "⚠️<font color=\"#d30c0c\">【警告】</font>您的云服务器\""+server_name+"\"当前CPU使用率为<font color=\"#d30c0c\">"+fmt.Sprintf("%.2f", info.CpuUsedPercent)+"%</font>，请及时检查系统是否存在问题。")
		}
		return
	}
}

func DingBotNotice(server_name string, bot *dingtalk.DingTalk) func() {
	return func() {
		i := status.GetSysInfo()
		d := status.GetDiskInfo()
		message := speaker.GetMarkDownInfoBySysInfoAndDiskInfo(i, d, server_name)
		bot.SendMarkDownMessage(server_name, message)
	}
}

// MailWarning 是 mail 渠道的周期告警检查函数
func MailWarning(server_name string, cfg speaker.MailConfig, loadavg_max_percent, mem_used_percent, cpu_used_percent float64) func() {
	return func() {
		info := status.GetSysInfo()
		avg, _ := load.Avg()
		cpuNum := runtime.NumCPU()
		loadavg_max := float64(cpuNum) * loadavg_max_percent / 100
		if avg.Load1 > loadavg_max || avg.Load5 > loadavg_max || avg.Load15 > loadavg_max {
			subject := "服务器告警：负载过高 - " + server_name
			body := fmt.Sprintf("【警告】您的云服务器\"%s\"当前负载过高\n\n当前负载 Load1=%.2f，阈值=%.2f，请及时检查系统是否存在问题。", server_name, avg.Load1, loadavg_max)
			_ = speaker.SendMail(cfg, subject, body)
		}
		if info.MemUsedPercent > mem_used_percent {
			subject := "服务器告警：内存使用率过高 - " + server_name
			body := fmt.Sprintf("【警告】您的云服务器\"%s\"当前内存使用率过高\n\n当前使用率 %.2f%%，阈值 %.2f%%，请及时检查系统是否存在问题。", server_name, info.MemUsedPercent, mem_used_percent)
			_ = speaker.SendMail(cfg, subject, body)
		}
		if info.CpuUsedPercent > cpu_used_percent {
			subject := "服务器告警：CPU使用率过高 - " + server_name
			body := fmt.Sprintf("【警告】您的云服务器\"%s\"当前CPU使用率过高\n\n当前使用率 %.2f%%，阈值 %.2f%%，请及时检查系统是否存在问题。", server_name, info.CpuUsedPercent, cpu_used_percent)
			_ = speaker.SendMail(cfg, subject, body)
		}
	}
}

// MailNotice 是 mail 渠道的每日概况推送函数
func MailNotice(server_name string, cfg speaker.MailConfig) func() {
	return func() {
		i := status.GetSysInfo()
		d := status.GetDiskInfo()
		subject := "服务器每日概况 - " + server_name
		body := fmt.Sprintln("服务器\""+server_name+"\"运行概况：") +
			fmt.Sprintf("运行时长：%d天%d小时%d分钟%d秒\n", i.Days, i.Hours, i.Minutes, i.Seconds) +
			fmt.Sprintf("内存使用率：%.2f%%\n", i.MemUsedPercent) +
			fmt.Sprintf("CPU使用率：%.2f%%\n", i.CpuUsedPercent) +
			fmt.Sprintf("磁盘使用率：%.2f%%\n", d.UsedPercent) +
			fmt.Sprintf("已用内存：%dMB / %dMB\n", i.MemUsed, i.MemAll) +
			fmt.Sprintf("空闲内存：%dMB\n", i.MemFree) +
			fmt.Sprintf("CPU核心数：%d核\n", i.CpuCores) +
			fmt.Sprintf("已用磁盘：%dGB / %dGB\n", d.Used/1024/1024/1024, d.Total/1024/1024/1024) +
			fmt.Sprintf("空闲磁盘：%dGB\n", d.Free/1024/1024/1024)
		_ = speaker.SendMail(cfg, subject, body)
	}
}

// JiangWarning 是 Server 酱渠道的周期告警检查函数
func JiangWarning(server_name string, c *speaker.Jiang, loadavg_max_percent, mem_used_percent, cpu_used_percent float64) func() {
	return func() {
		info := status.GetSysInfo()
		avg, _ := load.Avg()
		cpuNum := runtime.NumCPU()
		loadavg_max := float64(cpuNum) * loadavg_max_percent / 100
		if avg.Load1 > loadavg_max || avg.Load5 > loadavg_max || avg.Load15 > loadavg_max {
			_ = c.Send("服务器告警：负载过高 - "+server_name, fmt.Sprintf("【警告】服务器\"%s\"当前负载过高\n\n当前 Load1=%.2f，阈值=%.2f，请及时检查。", server_name, avg.Load1, loadavg_max))
		}
		if info.MemUsedPercent > mem_used_percent {
			_ = c.Send("服务器告警：内存使用率过高 - "+server_name, fmt.Sprintf("【警告】服务器\"%s\"内存使用率高\n\n当前使用率 %.2f%%，阈值 %.2f%%，请及时检查。", server_name, info.MemUsedPercent, mem_used_percent))
		}
		if info.CpuUsedPercent > cpu_used_percent {
			_ = c.Send("服务器告警：CPU使用率过高 - "+server_name, fmt.Sprintf("【警告】服务器\"%s\"CPU使用率高\n\n当前使用率 %.2f%%，阈值 %.2f%%，请及时检查。", server_name, info.CpuUsedPercent, cpu_used_percent))
		}
	}
}

// JiangNotice 是 Server 酱渠道的每日概况推送函数
func JiangNotice(server_name string, c *speaker.Jiang) func() {
	return func() {
		i := status.GetSysInfo()
		d := status.GetDiskInfo()
		title := "服务器每日概况 - " + server_name
		desp := "服务器\"" + server_name + "\"运行概况：\n\n" +
			"- 运行时长：" + fmt.Sprintf("%d天%d小时%d分钟%d秒\n", i.Days, i.Hours, i.Minutes, i.Seconds) +
			"- 内存使用率：" + fmt.Sprintf("%.2f%%\n", i.MemUsedPercent) +
			"- CPU使用率：" + fmt.Sprintf("%.2f%%\n", i.CpuUsedPercent) +
			"- 磁盘使用率：" + fmt.Sprintf("%.2f%%\n", d.UsedPercent) +
			"- 已用内存：" + fmt.Sprintf("%dMB / %dMB\n", i.MemUsed, i.MemAll) +
			"- 空闲内存：" + fmt.Sprintf("%dMB\n", i.MemFree) +
			"- CPU核心数：" + fmt.Sprintf("%d核\n", i.CpuCores) +
			"- 已用磁盘：" + fmt.Sprintf("%dGB / %dGB\n", d.Used/1024/1024/1024, d.Total/1024/1024/1024) +
			"- 空闲磁盘：" + fmt.Sprintf("%dGB\n", d.Free/1024/1024/1024)
		_ = c.Send(title, desp)
	}
}
