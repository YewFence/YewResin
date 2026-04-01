---
layout: home

hero:
  name: YewResin
  text: Docker 服务备份工具
  tagline: 自动化 Docker Compose 服务备份，使用 Kopia 实现本地快照与云端备份
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 配置参考
      link: /reference/configuration
    - theme: alt
      text: GitHub Repo
      link: https://github.com/YewFence/YewResin

features:
  - title: 一致性快照
    details: 自动停止所有 Docker Compose 服务后创建 Kopia 快照，确保数据完整性。支持优先级服务（如网关）的启停顺序控制。
  - title: 快速失败
    details: 服务停止失败时立即中止备份，避免在服务运行时备份导致数据损坏，已停止的服务会自动恢复。
  - title: 并行启停，服务状态一致
    details: 服务并行停止和启动，性能更优。只重启原本运行中的服务，不会启动原本停止的服务。
  - title: Gist 日志推送
    details: 将每次备份日志自动推送到 GitHub Gist，支持日志文件自动清理，方便远程查看和持久化。
  - title: 通知备份状态
    details: 支持 Apprise 通知，可通过 [YewFence/apprise](https://github.com/YewFence/apprise) 快速部署到 Vercel。
  - title: 多种运行模式
    details: 支持 dry-run 模式预览、跳过确认模式、锁机制防止重复运行，完美适配 cron 和 systemd timer。
---
