package yewresin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Notifier Apprise 通知发送器
type Notifier struct {
	url        string // Apprise 服务地址
	notifyURL  string // 通知目标 URL
	deviceName string // 设备名称（可选）
	wg         sync.WaitGroup
}

// NewNotifier 创建通知器
func NewNotifier(url, notifyURL, deviceName string) *Notifier {
	return &Notifier{
		url:        url,
		notifyURL:  notifyURL,
		deviceName: deviceName,
	}
}

// Send 发送通知
func (n *Notifier) Send(title, body string) {
	if n.url == "" || n.notifyURL == "" {
		slog.Debug("通知未配置，跳过发送", "title", title)
		return
	}

	// 如果配置了设备名称，添加到标题前
	if n.deviceName != "" {
		title = fmt.Sprintf("[%s] %s", n.deviceName, title)
	}

	n.wg.Add(1)
	go n.sendAsync(title, body)
}

// sendAsync 异步发送通知（不阻塞主流程）
func (n *Notifier) sendAsync(title, body string) {
	defer n.wg.Done()

	payload := map[string]string{
		"urls":  n.notifyURL,
		"title": title,
		"body":  body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("序列化通知数据失败", "error", err)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(n.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Warn("发送通知失败", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("通知服务返回错误", "status", resp.StatusCode)
		return
	}

	slog.Debug("通知发送成功", "title", title)
}

// Wait 等待所有通知发送完成
func (n *Notifier) Wait() {
	n.wg.Wait()
}
