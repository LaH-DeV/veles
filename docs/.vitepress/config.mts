import { defineConfig } from 'vitepress'
import { docsSidebarPlugin } from './sidebar.mts'
import velesGrammar from '../../editors/vscode/syntaxes/veles.tmLanguage.json'

const velesLanguage = {
  ...velesGrammar,
  name: 'Veles',
  aliases: ['veles', 'vs'],
}

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: 'Veles Docs',
  description: 'Veles language documentation',
  srcDir: './documentation',
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    sidebar: docsSidebarPlugin(),
  },
  markdown: {
    languages: [velesLanguage as any],
  },
})
