package status

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// defaultCacheTTL 是 NewInfoCache 在调用方传入 <=0 时的兜底有效期。
const defaultCacheTTL = 30 * time.Second

// InfoCache 是系统状态快照的并发安全缓存。
//
// 设计要点：
//   - 读路径命中内存快照，不调用 gopsutil；单次响应在微秒级。
//   - 快照通过 atomic.Pointer 持有，写新快照与并发读之间无需加锁即可保证可见性。
//   - 惰性刷新：Get 时发现过期则触发一次采集；可配合 StartBackgroundRefresh 周期预热。
//   - 同一时刻至多一次实际采集：通过 refreshing 单飞 + doneCh 唤醒等待者，
//     避免高并发下出现"惊群"重复采集或大量 goroutine 空转。
//   - 构造时不采集，避免启动阶段被 cpu.Percent / disk.Usage 阻塞
//     （后者在 NFS 等挂载点上可能 hang）。
type InfoCache struct {
	ttl  time.Duration
	snap atomic.Pointer[snapshot]

	mu         sync.Mutex
	refreshing bool
	doneCh     chan struct{} // 非 nil 表示有正在进行的采集，采集完成后被关闭
}

// snapshot 是不可变快照，写入 atomic指针后只读。
type snapshot struct {
	sysInfo LSysInfo
	disk    *DiskSnapshot
	built   time.Time
}

// DiskSnapshot 是 *disk.UsageStat 的本地副本，避免 gopsutil 类型泄漏到 API 层。
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

// NewInfoCache 构造缓存实例。ttl <= 0 时使用 defaultCacheTTL。
// 注意：构造函数不采集，首份快照由首次 Get 或后台预热产生，避免启动阻塞。
func NewInfoCache(ttl time.Duration) *InfoCache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &InfoCache{ttl: ttl}
}

// TTL 返回当前缓存有效期。
func (c *InfoCache) TTL() time.Duration {
	return c.ttl
}

// Get 返回当前快照。若缓存为空或已过期，则触发一次惰性刷新。
// 返回值第三位表示本次是否实际触发了一次采集。
func (c *InfoCache) Get() (LSysInfo, *DiskSnapshot, bool) {
	snap := c.snap.Load()
	if snap != nil && time.Since(snap.built) < c.ttl {
		return snap.sysInfo, snap.disk, false
	}
	refresh := c.doRefresh()
	snap = c.snap.Load()
	if snap == nil {
		// 极端情况下 doRefresh 未能产出快照（例如被快速 stop）：兜底采集一次。
		snap = collectOnce()
		c.snap.Store(snap)
		return snap.sysInfo, snap.disk, true
	}
	return snap.sysInfo, snap.disk, refresh
}

// doRefresh 保证同一次"过期窗口"内至多执行一次实际采集。
//
// - 若当前无采集在进行：本 goroutine 成为采集者，完成后唤醒所有等待者。
// - 当前已有采集在进行：阻塞等待其完成（而非空转），避免 goroutine 堆积。
func (c *InfoCache) doRefresh() bool {
	c.mu.Lock()
	if c.refreshing {
		ch := c.doneCh
		c.mu.Unlock()
		// 等待采集者完成；采集者关闭 ch 后所有等待者被唤醒。
		<-ch
		return false
	}
	c.refreshing = true
	c.doneCh = make(chan struct{})
	c.mu.Unlock()

	// 实际采集在锁外进行，避免阻塞其他 goroutine 的"加入等待"动作。
	snap := collectOnce()

	c.mu.Lock()
	c.refreshing = false
	close(c.doneCh)
	c.doneCh = nil
	c.mu.Unlock()

	c.snap.Store(snap)
	return true
}

// StartBackgroundRefresh 启动后台 goroutine，先立即采集一次，然后按 ttl 周期刷新。
// 返回的 stop 函数用于停止后台 goroutine（通过 channel 通知，阻塞至 goroutine 退出）。
func (c *InfoCache) StartBackgroundRefresh() (stop func()) {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		// 立即预热一次，避免首次请求承担采集延迟。
		c.doRefresh()
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

// collectOnce 执行一次实际采集并打包为 snapshot。
func collectOnce() *snapshot {
	return &snapshot{
		sysInfo: GetSysInfo(),
		disk:    toDiskSnapshot(GetDiskInfo()),
		built:   time.Now(),
	}
}

// toDiskSnapshot 将 gopsutil 返回的 *disk.UsageStat 转换为本地结构体。
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
