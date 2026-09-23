import { readFileSync, readdirSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

/**
 * 全仓守卫：scoped 样式里禁止 `:global(.dark) X` 这种「前缀式」写法。
 *
 * 为什么（实测，@vue/compiler-sfc@3.5.26 `dist/compiler-sfc.cjs.js:8133`）：
 *     if (value === ":global" || value === "::v-global") {
 *       selector.replaceWith(n.nodes[0]);   // 整个复杂选择器 → 只剩括号里那部分
 *       return false;                        // 中止遍历 ⇒ 后面跟的全丢
 *     }
 * 即 `:global(.dark) .foo` 恒等于 `:global(.dark)`，编译成**裸 `.dark`**：
 * 后代选择器与 data-v 属性一起丢失，声明落到 <html> 上。
 * 对可继承属性（color / 自定义属性）会被元素自身声明或 body 既有色压掉 ⇒
 * **编译不报错、运行不报错，只是暗色覆盖静默失效**。本仓曾因此在生产上
 * 出现「深色下标签页文字不可读」（SettingsView.vue 的注释记录了这次事故）。
 *
 * 正解：写普通的 `.dark X`（编译成 `.dark X[data-v-xxx]`）；
 * 确实需要完全不带 scoped 的全局规则时，把**整个选择器**包进括号：`:global(.dark .foo)`。
 *
 * ⚠️ 断言前必须剥掉 CSS 注释：注释里为了解释「为什么不能这么写」会引用坏写法本身。
 * ⚠️ 允许 `:global(.dark) { ... }`（单独使用）—— 那种编译成 `.dark { ... }` 是正确的。
 */
function vueFiles(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      out.push(...vueFiles(full))
    } else if (entry.name.endsWith('.vue')) {
      out.push(full)
    }
  }
  return out
}

/** 去掉 CSS 注释（只处理 `/* ... *\/`，够用且不会误伤选择器）。 */
function stripCssComments(text: string): string {
  return text.replace(/\/\*[\s\S]*?\*\//g, '')
}

describe('dark mode scoped selectors', () => {
  const srcRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
  const files = vueFiles(srcRoot)

  it('扫到了足够多的 .vue 文件（防止递归失效导致空跑通过）', () => {
    // 这条是「守卫的守卫」：若目录结构变化导致扫不到文件，上面的断言会变成真空通过。
    expect(files.length).toBeGreaterThan(100)
  })

  it('没有 `:global(.dark) X` 前缀式写法', () => {
    // 括号后跟了非 `{`/`}` 的实义字符 ⇒ 说明后面还挂了后代选择器，会被编译器吞掉。
    const offending = /:global\(\s*\.dark\s*\)\s*[^{}\s]/
    const hits: string[] = []

    for (const file of files) {
      const content = stripCssComments(readFileSync(file, 'utf8'))
      content.split('\n').forEach((line, index) => {
        if (offending.test(line)) {
          hits.push(`${relative(srcRoot, file).replace(/\\/g, '/')}:${index + 1}  ${line.trim()}`)
        }
      })
    }

    // 把命中行带进断言消息，失败时不用再手工 grep 一遍。
    expect(hits).toEqual([])
  })

  it('暗色覆盖用的是普通 `.dark X`（编译后会带上 data-v 属性）', () => {
    // 正向样本：至少存在一处普通写法，避免有人「一刀切」把暗色规则全删了。
    const withPlainDark = files.filter((file) =>
      /(^|\n)\s*\.dark\s+[^\s{]/.test(stripCssComments(readFileSync(file, 'utf8')))
    )
    expect(withPlainDark.length).toBeGreaterThan(0)
  })
})
