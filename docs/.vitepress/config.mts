import { defineConfig } from 'vitepress'
import { docsSidebarPlugin } from './sidebar.mts'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Veles Docs",
  description: "Veles language documentation",
  srcDir: './documentation',
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    sidebar: docsSidebarPlugin(),
  }
})
