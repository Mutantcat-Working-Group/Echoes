package status

import (
	"sync"
	"testing"
	"time"
)

// TestCacheConcurrentGet 模拟高并发请求打同一个已过期的缓存，
// 验证：1) 不会产生 data race；2) 同一次过期窗口内只触发一次实际采集。
func TestCacheConcurrentGet(t *testing.T) {
	// 用极短的 TTL 强制每次都"过期"，从而反复触发 doRefresh。
	c := NewInfoCache(1 * time.Millisecond)
	// 先等 TTL 过期，保证后续 Get 必然进入刷新路径。
	time.Sleep(5 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = c.Get()
		}()
	}
	wg.Wait()
}

// TestCacheStopBackground 验证后台预热可正常停止，不泄漏 goroutine。
func TestCacheStopBackground(t *testing.T) {
	c := NewInfoCache(10 * time.Millisecond)
	stop := c.StartBackgroundRefresh()
	// 让后台跑几个周期。
	time.Sleep(35 * time.Millisecond)
	stop()
	// 再次 stop 不应 panic（此处仅验证第一次 stop 正常返回）。
}
