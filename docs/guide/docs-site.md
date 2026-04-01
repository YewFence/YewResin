# 文档站维护

文档站使用 [VitePress](https://vitepress.dev/) 构建，源文件在 `docs/` 目录下。

## 目录结构

```
docs/
├── .vitepress/
│   ├── config.ts          # 站点配置（导航栏、侧边栏、搜索等）
│   ├── dist/              # 构建产物（gitignore）
│   └── cache/             # 构建缓存（gitignore）
├── package.json           # VitePress 依赖
├── index.md               # 首页
├── guide/                 # 指南文档
│   ├── getting-started.md
│   ├── how-it-works.md
│   ├── gist-logging.md
│   ├── scheduling.md
│   ├── recovery.md
│   └── development.md
└── reference/             # 参考文档
    └── configuration.md
```

## 环境要求

- **Node.js 22+**
- **pnpm**（包管理器）

## 本地开发

```bash
cd docs

# 安装依赖
pnpm install

# 启动开发服务器（热更新）
pnpm dev

# 构建静态站点
pnpm build

# 本地预览构建结果
pnpm preview
```

开发服务器默认在 `http://localhost:5173` 启动，修改 `.md` 文件会自动热更新。

## 添加新页面

1. 在 `docs/guide/` 或 `docs/reference/` 下创建新的 `.md` 文件
2. 在 `docs/.vitepress/config.ts` 的 `sidebar` 中添加对应的链接

## 部署

文档站通过 GitHub Actions 自动部署到 GitHub Pages：

- **触发条件**：push 到 `main` 分支且 `docs/` 目录有变更，或手动触发
- **工作流文件**：`.github/workflows/docs.yml`

首次使用需要在 GitHub 仓库中配置：

1. 进入仓库 **Settings → Pages**
2. **Source** 选择 **GitHub Actions**
3. 推送到 `main` 后自动部署

站点地址：`https://yewfence.github.io/YewResin/`
