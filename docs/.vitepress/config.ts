import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/YewResin/',
  lang: 'zh-CN',
  title: 'YewResin',
  description: '自动化的 Docker Compose 服务备份工具，使用 Kopia 实现本地快照与云端同步',

  themeConfig: {
    nav: [
      { text: '指南', link: '/guide/getting-started' },
      { text: '配置参考', link: '/reference/configuration' },
      { text: 'GitHub', link: 'https://github.com/YewFence/YewResin' }
    ],

    sidebar: [
      {
        text: '指南',
        items: [
          { text: '快速开始', link: '/guide/getting-started' },
          { text: '工作原理', link: '/guide/how-it-works' },
          { text: 'Gist 日志推送', link: '/guide/gist-logging' },
          { text: '定时任务', link: '/guide/scheduling' },
          { text: '异地恢复', link: '/guide/recovery' },
          { text: '开发指南', link: '/guide/development' },
          { text: '文档站维护', link: '/guide/docs-site' }
        ]
      },
      {
        text: '参考',
        items: [
          { text: '配置项', link: '/reference/configuration' }
        ]
      }
    ],

    search: {
      provider: 'local'
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/YewFence/YewResin' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © YewFence'
    },

    docFooter: {
      prev: '上一页',
      next: '下一页'
    },

    outline: {
      label: '本页目录'
    },

    lastUpdated: {
      text: '最后更新'
    }
  }
})
