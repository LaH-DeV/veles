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
  // the chapters link the spec and the checklist at the repository's root,
  // which resolve when the Markdown is read on GitHub but are not pages here
  ignoreDeadLinks: [/\/veles-(spec|checklist)$/],
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    sidebar: docsSidebarPlugin(),
  },
  markdown: {
    languages: [velesLanguage as any],
  },
})
