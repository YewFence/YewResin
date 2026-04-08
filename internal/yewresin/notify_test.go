package yewresin

import "testing"

func TestNewNotifier(t *testing.T) {
	n := NewNotifier("http://localhost:8000", "tgram://bot/token", "myserver")

	if n.url != "http://localhost:8000" {
		t.Fatalf("url: got %q", n.url)
	}
	if n.notifyURL != "tgram://bot/token" {
		t.Fatalf("notifyURL: got %q", n.notifyURL)
	}
	if n.deviceName != "myserver" {
		t.Fatalf("deviceName: got %q", n.deviceName)
	}
}

func TestSendNotConfigured(t *testing.T) {
	n := NewNotifier("", "", "")
	// 未配置时 Send 应直接返回，不 panic
	n.Send("test", "body")
	n.Wait()
}

func TestWaitNoPending(t *testing.T) {
	n := NewNotifier("", "", "")
	// 没有发送过任何通知时 Wait 应立即返回
	n.Wait()
}
