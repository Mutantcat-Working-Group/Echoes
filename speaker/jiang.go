package speaker

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Jiang 是 Server 酱（方糖）的统一推送客户端
// 文档：https://sct.ftqq.com/
// 只需要一个 SendKey，对应启动参数 -token
type Jiang struct {
	SendKey string
	client  *http.Client
}

// NewJiang 创建一个 Server 酱客户端
func NewJiang(sendKey string) *Jiang {
	return &Jiang{
		SendKey: sendKey,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Send 发送一条消息（desp 可以为空）
func (j *Jiang) Send(title, desp string) error {
	if j.SendKey == "" {
		return fmt.Errorf("Server 酱 SendKey 为空")
	}
	form := url.Values{}
	form.Set("title", title)
	form.Set("desp", desp)

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("https://sctapi.ftqq.com/%s.send", j.SendKey),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("Server 酱构建请求: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("Server 酱请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Server 酱响应 %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
