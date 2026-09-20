package router

import (
	"com.mutantcat.echoes/status"
	"github.com/gin-gonic/gin"
)

type InfoRouter struct {
	ServerName string
	// Cache 允许外部注入缓存实例，便于复用同一份快照（高并发场景）。
	// 若为 nil，则每次请求直接采集，性能等同于未启用缓存的旧行为。
	Cache *status.InfoCache
}

func (r *InfoRouter) PrepareRouter() error {
	return nil
}

func (r *InfoRouter) InitRouter(c *gin.Engine) error {
	c.Any("/ping", ping)
	c.Any("/info", r.getAllInfo)
	return nil
}

func (r *InfoRouter) DestroyRouter() error {
	return nil
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "pong",
	})
}

// getAllInfo 返回系统信息。
// 优先读缓存：缓存命中时不再触发 CPU 采样和磁盘 IO，可支撑高并发探针。
func (r *InfoRouter) getAllInfo(c *gin.Context) {
	var (
		sys status.LSysInfo
		dsk *status.DiskSnapshot
	)
	if r.Cache != nil {
		sys, dsk, _ = r.Cache.Get()
	} else {
		// 兼容未启用缓存的旧调用路径。
		sys = status.GetSysInfo()
		if d := status.GetDiskInfo(); d != nil {
			dsk = &status.DiskSnapshot{
				Path:              d.Path,
				Fstype:            d.Fstype,
				Total:             d.Total,
				Free:              d.Free,
				Used:              d.Used,
				UsedPercent:       d.UsedPercent,
				InodesTotal:       d.InodesTotal,
				InodesUsed:        d.InodesUsed,
				InodesFree:        d.InodesFree,
				InodesUsedPercent: d.InodesUsedPercent,
			}
		}
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"system": sys,
			"disk":   dsk,
		},
	})
}
