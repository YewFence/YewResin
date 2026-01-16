package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

// GistManager 管理 GitHub Gist 日志上传
type GistManager struct {
	token         string
	gistID        string
	logPrefix     string
	maxLogs       int
	keepFirstFile bool
	client        *http.Client
}

// NewGistManager 创建 Gist 管理器
func NewGistManager(token, gistID, logPrefix string, maxLogs int, keepFirstFile bool) *GistManager {
	return &GistManager{
		token:         token,
		gistID:        gistID,
		logPrefix:     logPrefix,
		maxLogs:       maxLogs,
		keepFirstFile: keepFirstFile,
		client:        &http.Client{Timeout: 30 * time.Second},
	}
}

// IsConfigured 检查是否已配置 Gist
func (g *GistManager) IsConfigured() bool {
	return g.token != "" && g.gistID != ""
}

// Upload 上传日志到 Gist
func (g *GistManager) Upload(logContent string, success bool, startTime time.Time, duration time.Duration) error {
	if !g.IsConfigured() {
		return nil
	}

	slog.Info("上传日志到 GitHub Gist...")

	// 生成文件名
	timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s-%s.log", g.logPrefix, timestamp)

	// 构建日志内容
	statusText := "✅ 成功"
	if !success {
		statusText = "⚠️ 有警告"
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	var durationStr string
	if hours > 0 {
		durationStr = fmt.Sprintf("%d 小时 %d 分 %d 秒", hours, minutes, seconds)
	} else if minutes > 0 {
		durationStr = fmt.Sprintf("%d 分 %d 秒", minutes, seconds)
	} else {
		durationStr = fmt.Sprintf("%d 秒", seconds)
	}

	content := fmt.Sprintf(`========================================
YewResin Docker 备份日志
========================================
执行状态: %s
开始时间: %s
耗时: %s
结束时间: %s
========================================
详细日志:
========================================
%s
`, statusText, startTime.Format("2006-01-02 15:04:05"), durationStr,
		time.Now().Format("2006-01-02 15:04:05"), logContent)

	// 构建请求 payload
	payload := map[string]interface{}{
		"files": map[string]interface{}{
			filename: map[string]string{
				"content": content,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	// 发送请求
	url := fmt.Sprintf("https://api.github.com/gists/%s", g.gistID)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gist API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	slog.Info("✓ 日志已上传到 Gist", "url", fmt.Sprintf("https://gist.github.com/%s", g.gistID))

	// 上传成功后清理旧日志
	if err := g.cleanupOldLogs(); err != nil {
		slog.Warn("清理旧日志失败", "error", err)
	}

	return nil
}

// gistResponse Gist API 响应结构
type gistResponse struct {
	ID    string                 `json:"id"`
	Files map[string]interface{} `json:"files"`
}

// cleanupOldLogs 清理旧的 Gist 日志文件
func (g *GistManager) cleanupOldLogs() error {
	if g.maxLogs <= 0 {
		return nil
	}

	slog.Info("检查 Gist 日志数量...")

	// 获取 Gist 信息
	url := fmt.Sprintf("https://api.github.com/gists/%s", g.gistID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("获取 Gist 信息失败: %d", resp.StatusCode)
	}

	var gist gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		return err
	}

	// 获取所有文件名并排序
	var allFiles []string
	for filename := range gist.Files {
		allFiles = append(allFiles, filename)
	}
	sort.Strings(allFiles)

	// 如果启用了保留第一个文件，从列表中排除
	filesToConsider := allFiles
	if g.keepFirstFile && len(allFiles) > 0 {
		firstFile := allFiles[0]
		slog.Info("保留第一个文件", "file", firstFile)
		filesToConsider = allFiles[1:]
	}

	// 只统计匹配前缀的日志文件
	var logFiles []string
	for _, f := range filesToConsider {
		if strings.HasPrefix(f, g.logPrefix) {
			logFiles = append(logFiles, f)
		}
	}

	// 如果文件数量未超过限制，跳过清理
	if len(logFiles) <= g.maxLogs {
		slog.Info("当前日志数量未超过限制，无需清理",
			"current", len(logFiles), "max", g.maxLogs)
		return nil
	}

	// 计算需要删除的文件数量（删除最旧的）
	deleteCount := len(logFiles) - g.maxLogs
	filesToDelete := logFiles[:deleteCount]

	slog.Info("需要删除旧日志文件", "count", deleteCount)

	// 构建删除 payload（将文件内容设为 null 表示删除）
	deleteFiles := make(map[string]interface{})
	for _, f := range filesToDelete {
		deleteFiles[f] = nil
	}

	payload := map[string]interface{}{
		"files": deleteFiles,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 发送删除请求
	req, err = http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err = g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除旧日志失败: %s", string(body))
	}

	slog.Info("✓ 已清理旧日志文件", "count", deleteCount)
	return nil
}
