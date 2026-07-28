package status

import (
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// cacheTTL 控制 /info 接口返回的状态快照有效期。
// 在 TTL 窗口内的并发请求会复用同一份快照，避免 CPU 采样、磁盘 IO 等重型调用被打爆。
// 默认值由 NewCache 调用方传入；这里仅作为兜底，避免 0 值导致"永远不过期"或"每次都重新采集"。
const defaultCacheTTL = 30 * time.Second

// InfoCache 是系统状态快照的并发安全缓存。
//
// 设计要点：
//   - 读路径完全不调用 gopsutil，全部命中内存快照，单次响应在微秒级。
//   - 使用 atomic.Pointer 持有快照，refresh 与并发读之间无需加锁即可保证可见性。
//   - 默认惰性刷新：访问时若过期则同步刷新；可配合 StartBackgroundRefresh 周期预热。
//   - 当 refresh 正在进行时，多个并发请求通过 refreshing 单飞标志合并为一次实际采集。
type InfoCache struct {
	ttl        time.Duration
	snap       atomic.Pointer[snapshot]
	refreshing atomic.Bool // 防重入：避免并发请求各自触发一次采集
}

// snapshot 是某次采集产出的不可变快照。
// 一旦写入原子指针后就不再被修改，保证读路径零锁。
type snapshot struct {
	sysInfo LSysInfo
	disk    *DiskSnapshot
	built   time.Time
}

// DiskSnapshot 是 *disk.UsageStat 的轻量副本，避免对外暴露 gopsutil 类型。
type DiskSnapshot struct {
	Path              string  `json:"path"`
	Fstype            string  `json:"fstype"`
	Total             uint64  `json:"total"`
	Free              uint64  `json:"free"`
	Used              uint64  `json:"used"`
	UsedPercent       float64 `json:"usedPercent"`
	InodesTotal       uint64  `json:"inodesTotal"`
	InodesUsed        uint64  `json:"inodesUsed"`
	InodesFree        uint64  `json:"inodesFree"`
	InodesUsedPercent float64 `json:"inodesUsedPercent"`
}

// NewInfoCache 构造一个缓存实例，ttl 为 0 时使用 defaultCacheTTL。
func NewInfoCache(ttl time.Duration) *InfoCache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	c := &InfoCache{ttl: ttl}
	// 预热一次首屏快照，避免第一次 /info 请求延迟。
	c.doRefresh()
	return c
}

// TTL 返回当前缓存有效期。
func (c *InfoCache) TTL() time.Duration {
	return c.ttl
}

// Get 返回当前快照；若已过期则触发一次惰性刷新。
// 返回值第三位表示本次是否实际触发了一次采集。
func (c *InfoCache) Get() (LSysInfo, *DiskSnapshot, bool) {
	snap := c.snap.Load()
	if snap != nil && time.Since(snap.built) < c.ttl {
		return snap.sysInfo, snap.disk, false
	}
	c.doRefresh()
	snap = c.snap.Load()
	if snap == nil {
		// refresh 失败且无旧快照：兜底采集一次，确保接口不返回 nil。
		fallback := collectOnce()
		c.snap.Store(fallback)
		return fallback.sysInfo, fallback.disk, true
	}
	return snap.sysInfo, snap.disk, true
}

// doRefresh 重新采集一次并原子替换快照。
// 使用 refreshing 标志做单飞控制：同一时刻只会有一次实际的采集调用。
func (c *InfoCache) doRefresh() {
	if !c.refreshing.CompareAndSwap(false, true) {
		// 已有其他 goroutine 正在刷新，等待其完成即可，避免重复采集。
		deadline := time.Now().Add(c.ttl)
		for {
			s := c.snap.Load()
			if s != nil && time.Since(s.built) < c.ttl {
				return
			}
			if time.Now().After(deadline) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	defer c.refreshing.Store(false)
	c.snap.Store(collectOnce())
}

// StartBackgroundRefresh 启动一个后台 goroutine 周期刷新缓存；
// 返回的 stop 函数用于停止该 goroutine，便于测试和优雅退出。
func (c *InfoCache) StartBackgroundRefresh() (stop func()) {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(c.ttl)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				c.doRefresh()
			}
		}
	}()
	return func() {
		close(stopCh)
		<-doneCh
	}
}

// collectOnce 执行一次实际的状态采集并打包为 snapshot。
func collectOnce() *snapshot {
	return &snapshot{
		sysInfo: GetSysInfo(),
		disk:    toDiskSnapshot(GetDiskInfo()),
		built:   time.Now(),
	}
}

// toDiskSnapshot 将 gopsutil 返回的 *disk.UsageStat 转换为本地结构体，
// 避免外部依赖泄漏到缓存层 / API 层。
func toDiskSnapshot(d *disk.UsageStat) *DiskSnapshot {
	if d == nil {
		return nil
	}
	return &DiskSnapshot{
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