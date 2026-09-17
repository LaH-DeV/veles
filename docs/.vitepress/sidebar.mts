import fs from 'node:fs'
import path from 'node:path'
import type { DefaultTheme } from 'vitepress'

type SidebarItem = DefaultTheme.SidebarItem

const documentationDir = path.resolve(__dirname, '../documentation')

function titleFor(filePath: string): string {
  const contents = fs.readFileSync(filePath, 'utf8')
  const heading = contents.match(/^#\s+(.+)$/m)?.[1]

  return heading?.replace(/^\d+\.\s+/, '').trim() ?? path.basename(filePath, '.md')
}

function linkFor(filePath: string): string {
  const relativePath = path.relative(documentationDir, filePath)
  return `/${relativePath.replace(/\\/g, '/').replace(/\.md$/, '')}`
}

function directoryTitle(directory: string): string {
  const indexPath = path.join(directory, 'index.md')
  return fs.existsSync(indexPath) ? titleFor(indexPath) : path.basename(directory)
}

function entriesFor(directory: string): SidebarItem[] {
  const entries = fs.readdirSync(directory, { withFileTypes: true })
  // index.md is the directory's landing page and leads its section.
  const files = entries
    .filter((entry) => entry.isFile() && entry.name.endsWith('.md'))
    .sort((left, right) => {
      if (left.name === 'index.md') return -1
      if (right.name === 'index.md') return 1
      return left.name.localeCompare(right.name, undefined, { numeric: true })
    })
  const directories = entries
    .filter((entry) => entry.isDirectory())
    .sort((left, right) => left.name.localeCompare(right.name, undefined, { numeric: true }))

  return [
    ...files.map((entry) => {
      const filePath = path.join(directory, entry.name)
      return { text: titleFor(filePath), link: linkFor(filePath) }
    }),
    ...directories.map((entry) => ({
      text: directoryTitle(path.join(directory, entry.name)),
      items: entriesFor(path.join(directory, entry.name))
    }))
  ]
}

export function docsSidebarPlugin(): DefaultTheme.SidebarItem[] {
  return entriesFor(documentationDir)
}