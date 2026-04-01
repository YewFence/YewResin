# YewResin Go 版本 TODO

## 待实现功能

- [x] Gist 日志上传（参考 src/06-gist.sh）
- [x] 并行停止/启动服务（goroutine + sync.WaitGroup）
- [x] 日志文件输出（当前只输出到终端）
- [x] 交叉编译脚本 / Makefile
- [x] GitHub Actions CI/CD

## 可选改进

- [x] 单元测试
- [ ] 支持 XDG 规范配置目录（如 `~/.config/yewresin/.env`），改善 Homebrew 等包管理器安装后的配置体验
- [ ] JSON 日志格式输出选项
- [ ] 引入 pinact github action 自动 pin action 版本

## 新功能
- [ ] 明确指定次序的优先服务启停(应用场景先启动数据库再启动AI网关再启动依赖AI的服务)
- [ ] 支持自定义启停脚本名称
- [ ] 自动安装/卸载 systemd timer
