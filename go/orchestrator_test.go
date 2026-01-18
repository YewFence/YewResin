package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeDocker struct {
	services           []*Service
	checkErr           error
	discoverErr        error
	stopParallelErrs   []error
	startParallelErrs  []error
	stopErrs           map[string]error
	startErrs          map[string]error
	stopCalls          []string
	startCalls         []string
	stopParallelCalled int
	startParallelCalled int
}

func (f *fakeDocker) CheckDependencies() error {
	return f.checkErr
}

func (f *fakeDocker) DiscoverServices() ([]*Service, error) {
	if f.discoverErr != nil {
		return nil, f.discoverErr
	}
	return f.services, nil
}

func (f *fakeDocker) StopParallel(services []*Service) []error {
	f.stopParallelCalled++
	return f.stopParallelErrs
}

func (f *fakeDocker) Stop(svc *Service) error {
	f.stopCalls = append(f.stopCalls, svc.Name)
	if f.stopErrs == nil {
		return nil
	}
	return f.stopErrs[svc.Name]
}

func (f *fakeDocker) StartParallel(services []*Service) []error {
	f.startParallelCalled++
	return f.startParallelErrs
}

func (f *fakeDocker) Start(svc *Service) error {
	f.startCalls = append(f.startCalls, svc.Name)
	if f.startErrs == nil {
		return nil
	}
	return f.startErrs[svc.Name]
}

type fakeKopia struct {
	checkErr   error
	createErr  error
	createCalls []string
}

func (f *fakeKopia) CheckRepository() error {
	return f.checkErr
}

func (f *fakeKopia) CreateSnapshot(path string) error {
	f.createCalls = append(f.createCalls, path)
	return f.createErr
}

type notifyCall struct {
	title string
	body  string
}

type fakeNotifier struct {
	sends     []notifyCall
	waitCount int
}

func (f *fakeNotifier) Send(title, body string) {
	f.sends = append(f.sends, notifyCall{title: title, body: body})
}

func (f *fakeNotifier) Wait() {
	f.waitCount++
}

type uploadCall struct {
	content  string
	success  bool
	start    time.Time
	duration time.Duration
}

type fakeGist struct {
	uploads   []uploadCall
	uploadErr error
}

func (f *fakeGist) Upload(logContent string, success bool, startTime time.Time, duration time.Duration) error {
	f.uploads = append(f.uploads, uploadCall{
		content:  logContent,
		success:  success,
		start:    startTime,
		duration: duration,
	})
	return f.uploadErr
}

func TestOrchestratorRunSuccessDryRun(t *testing.T) {
	// dry-run 模式下完整流程应走通，且不依赖真实外部服务
	baseDir := t.TempDir()
	lockPath := filepath.Join(t.TempDir(), "lock")

	cfg := &Config{
		BaseDir:          baseDir,
		ExpectedRemote:   "remote",
		PriorityServices: []string{"gateway"},
		LockFile:         lockPath,
	}

	docker := &fakeDocker{
		services: []*Service{
			{Name: "gateway"},
			{Name: "db"},
		},
	}
	kopia := &fakeKopia{}
	notifier := &fakeNotifier{}
	gist := &fakeGist{}

	LogWriter = NewLogCapture(io.Discard)

	// 手动组装 Orchestrator，注入 fake 依赖
	o := &Orchestrator{
		cfg:      cfg,
		dryRun:   true,
		docker:   docker,
		kopia:    kopia,
		notifier: notifier,
		gist:     gist,
	}

	if err := o.Run(); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// 快照应该被调用一次
	if len(kopia.createCalls) != 1 || kopia.createCalls[0] != baseDir {
		t.Fatalf("expected snapshot created for %q, got %v", baseDir, kopia.createCalls)
	}

	// 普通服务走并行停止，优先服务走顺序停止
	if docker.stopParallelCalled != 1 {
		t.Fatalf("expected StopParallel called once, got %d", docker.stopParallelCalled)
	}
	if len(docker.stopCalls) != 1 || docker.stopCalls[0] != "gateway" {
		t.Fatalf("expected Stop called for gateway, got %v", docker.stopCalls)
	}
	if docker.startParallelCalled != 1 {
		t.Fatalf("expected StartParallel called once, got %d", docker.startParallelCalled)
	}
	if len(docker.startCalls) != 1 || docker.startCalls[0] != "gateway" {
		t.Fatalf("expected Start called for gateway, got %v", docker.startCalls)
	}

	if notifier.waitCount != 1 {
		t.Fatalf("expected notifier Wait called once, got %d", notifier.waitCount)
	}
	// 起始通知与完成通知都应发送
	if len(notifier.sends) < 2 {
		t.Fatalf("expected at least 2 notifications, got %d", len(notifier.sends))
	}

	// 成功时应上传日志
	if len(gist.uploads) != 1 {
		t.Fatalf("expected gist upload called once, got %d", len(gist.uploads))
	}
	if !gist.uploads[0].success {
		t.Fatalf("expected gist upload success flag true")
	}
}

func TestOrchestratorRunStopParallelError(t *testing.T) {
	// 普通服务并行停止失败时，应中止流程并尝试恢复服务
	baseDir := t.TempDir()
	lockPath := filepath.Join(t.TempDir(), "lock")

	cfg := &Config{
		BaseDir:          baseDir,
		ExpectedRemote:   "remote",
		PriorityServices: []string{"gateway"},
		LockFile:         lockPath,
	}

	docker := &fakeDocker{
		services: []*Service{
			{Name: "gateway"},
			{Name: "db"},
		},
		stopParallelErrs: []error{errors.New("stop failed")},
	}
	kopia := &fakeKopia{}
	notifier := &fakeNotifier{}
	gist := &fakeGist{}

	LogWriter = NewLogCapture(io.Discard)

	// 手动组装 Orchestrator，注入 fake 依赖
	o := &Orchestrator{
		cfg:      cfg,
		dryRun:   true,
		docker:   docker,
		kopia:    kopia,
		notifier: notifier,
		gist:     gist,
	}

	if err := o.Run(); err == nil {
		t.Fatalf("expected Run to fail")
	}

	// 失败后应尝试启动服务回滚
	if docker.startParallelCalled != 1 {
		t.Fatalf("expected StartParallel called once after failure, got %d", docker.startParallelCalled)
	}
	// 早期失败不应上传日志
	if len(gist.uploads) != 0 {
		t.Fatalf("expected no gist upload on stop failure")
	}
}

func TestOrchestratorLockLifecycle(t *testing.T) {
	// 获取锁后应创建目录，释放锁后应删除
	lockPath := filepath.Join(t.TempDir(), "lock")
	cfg := &Config{
		BaseDir:        t.TempDir(),
		ExpectedRemote: "remote",
		LockFile:       lockPath,
	}

	o := &Orchestrator{cfg: cfg, dryRun: false}

	if err := o.acquireLock(); err != nil {
		t.Fatalf("acquireLock error: %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected lock dir created: %v", err)
	}

	o.releaseLock()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected lock dir removed, got %v", err)
	}
}

func TestOrchestratorAcquireLockAlreadyExists(t *testing.T) {
	// 锁目录已存在时应返回错误
	lockPath := filepath.Join(t.TempDir(), "lock")
	if err := os.Mkdir(lockPath, 0o755); err != nil {
		t.Fatalf("create lock dir: %v", err)
	}

	cfg := &Config{
		BaseDir:        t.TempDir(),
		ExpectedRemote: "remote",
		LockFile:       lockPath,
	}
	o := &Orchestrator{cfg: cfg, dryRun: false}

	if err := o.acquireLock(); err == nil {
		t.Fatalf("expected acquireLock to fail when lock exists")
	}
}
