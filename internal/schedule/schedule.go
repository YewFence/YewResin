package schedule

import (
	"fmt"
	"io"
	"strings"
)

const DefaultBackend = "cron"

type Options struct {
	Backend    string
	Expr       string
	OnCalendar string
	ConfigPath string
}

type backend interface {
	normalizeInstallOptions(*Options) error
	ensureSupported() error
	install(io.Writer, Options) error
	uninstall(io.Writer) error
	status(io.Writer) error
}

func NormalizeInstallOptions(opts *Options) error {
	if opts == nil {
		return fmt.Errorf("调度配置不能为空")
	}

	backendImpl, err := getBackend(opts.Backend)
	if err != nil {
		return err
	}

	opts.Backend = normalizeBackendName(opts.Backend)
	return backendImpl.normalizeInstallOptions(opts)
}

func Install(stdout io.Writer, opts Options) error {
	return runAction(opts.Backend, func(backendImpl backend) error {
		return backendImpl.install(stdout, opts)
	})
}

func Uninstall(stdout io.Writer, backendName string) error {
	return runAction(backendName, func(backendImpl backend) error {
		return backendImpl.uninstall(stdout)
	})
}

func Status(stdout io.Writer, backendName string) error {
	return runAction(backendName, func(backendImpl backend) error {
		return backendImpl.status(stdout)
	})
}

func runAction(backendName string, action func(backend) error) error {
	backendImpl, err := getBackend(backendName)
	if err != nil {
		return err
	}
	if err := backendImpl.ensureSupported(); err != nil {
		return err
	}
	return action(backendImpl)
}

func getBackend(name string) (backend, error) {
	switch normalizeBackendName(name) {
	case "cron":
		return cronBackend{}, nil
	case "systemd-user":
		return systemdUserBackend{}, nil
	default:
		return nil, fmt.Errorf("不支持的调度后端: %s（仅支持 cron、systemd-user）", name)
	}
}

func normalizeBackendName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return DefaultBackend
	}
	return name
}
