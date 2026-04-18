<script setup>
import { computed, onMounted, reactive } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { AnalyzeCatalog, DeleteRule, ExportNovel, LoadState, SaveRule } from '../wailsjs/go/main/App'

const state = reactive({
  loading: true,
  savingRule: false,
  analyzing: false,
  exporting: false,
  rules: [],
  analysis: null,
  progress: null,
  error: '',
  success: '',
})

const crawlForm = reactive({
  catalogUrl: 'https://www.zhswx.com/chapter/67027.html',
  ruleId: '',
  maxChapters: 0,
  skipFilteredTitle: true,
})

const ruleForm = reactive(createEmptyRule())

const selectedRule = computed(() => state.rules.find((rule) => rule.id === crawlForm.ruleId) ?? null)

onMounted(async () => {
  await refreshState()
  EventsOn('crawl-progress', (payload) => {
    state.progress = payload
  })
})

function createEmptyRule() {
  return {
    id: '',
    name: '',
    matchDomains: 'example.com',
    catalogSectionHeadingText: '',
    catalogSectionContainer: '',
    catalogChapterLinkSelector: '',
    chapterTitleSelector: '',
    chapterContentSelector: '',
    nextPageSelector: '',
    nextChapterSelector: '',
    contentCleanupSelectors: '',
    contentStopTexts: '',
    skipChapterTitlePatterns: '',
    notes: '',
  }
}

function splitLines(text) {
  return text.split('\n').map((item) => item.trim()).filter(Boolean)
}

function applyRuleToForm(rule) {
  Object.assign(ruleForm, {
    id: rule.id ?? '',
    name: rule.name ?? '',
    matchDomains: (rule.matchDomains ?? []).join('\n'),
    catalogSectionHeadingText: rule.catalogSectionHeadingText ?? '',
    catalogSectionContainer: rule.catalogSectionContainer ?? '',
    catalogChapterLinkSelector: rule.catalogChapterLinkSelector ?? '',
    chapterTitleSelector: rule.chapterTitleSelector ?? '',
    chapterContentSelector: rule.chapterContentSelector ?? '',
    nextPageSelector: rule.nextPageSelector ?? '',
    nextChapterSelector: rule.nextChapterSelector ?? '',
    contentCleanupSelectors: (rule.contentCleanupSelectors ?? []).join('\n'),
    contentStopTexts: (rule.contentStopTexts ?? []).join('\n'),
    skipChapterTitlePatterns: (rule.skipChapterTitlePatterns ?? []).join('\n'),
    notes: rule.notes ?? '',
  })
}

function resetRuleForm() {
  Object.assign(ruleForm, createEmptyRule())
}

async function refreshState() {
  state.loading = true
  try {
    const result = await LoadState()
    state.rules = result.rules ?? []
    if (!crawlForm.ruleId && state.rules.length > 0) {
      crawlForm.ruleId = state.rules[0].id
      applyRuleToForm(state.rules[0])
    }
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.loading = false
  }
}

function pickRule(rule) {
  crawlForm.ruleId = rule.id
  applyRuleToForm(rule)
  state.error = ''
  state.success = ''
}

async function analyzeCatalog() {
  state.error = ''
  state.success = ''
  state.analysis = null
  state.analyzing = true

  try {
    state.analysis = await AnalyzeCatalog({
      catalogUrl: crawlForm.catalogUrl,
      ruleId: crawlForm.ruleId,
    })
    state.success = `目录分析完成，共识别 ${state.analysis.chapterCount} 章。`
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.analyzing = false
  }
}

async function saveRule() {
  state.error = ''
  state.success = ''
  state.savingRule = true

  try {
    const payload = {
      id: ruleForm.id.trim(),
      name: ruleForm.name.trim(),
      matchDomains: splitLines(ruleForm.matchDomains),
      catalogSectionHeadingText: ruleForm.catalogSectionHeadingText.trim(),
      catalogSectionContainer: ruleForm.catalogSectionContainer.trim(),
      catalogChapterLinkSelector: ruleForm.catalogChapterLinkSelector.trim(),
      chapterTitleSelector: ruleForm.chapterTitleSelector.trim(),
      chapterContentSelector: ruleForm.chapterContentSelector.trim(),
      nextPageSelector: ruleForm.nextPageSelector.trim(),
      nextChapterSelector: ruleForm.nextChapterSelector.trim(),
      contentCleanupSelectors: splitLines(ruleForm.contentCleanupSelectors),
      contentStopTexts: splitLines(ruleForm.contentStopTexts),
      skipChapterTitlePatterns: splitLines(ruleForm.skipChapterTitlePatterns),
      notes: ruleForm.notes.trim(),
    }

    const result = await SaveRule(payload)
    state.rules = result.rules ?? []
    const savedRule = state.rules.find((rule) => rule.id === payload.id) ?? state.rules.at(-1)
    if (savedRule) {
      crawlForm.ruleId = savedRule.id
      applyRuleToForm(savedRule)
    }
    state.success = '规则已保存。'
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.savingRule = false
  }
}

async function removeRule() {
  if (!ruleForm.id) {
    return
  }

  state.error = ''
  state.success = ''

  try {
    const result = await DeleteRule(ruleForm.id)
    state.rules = result.rules ?? []
    if (crawlForm.ruleId === ruleForm.id) {
      crawlForm.ruleId = state.rules[0]?.id ?? ''
    }
    if (state.rules.length > 0) {
      applyRuleToForm(state.rules[0])
    } else {
      resetRuleForm()
    }
    state.success = '规则已删除。'
  } catch (error) {
    state.error = error?.message ?? String(error)
  }
}

async function exportNovel() {
  state.error = ''
  state.success = ''
  state.exporting = true
  state.progress = null

  try {
    const result = await ExportNovel({
      catalogUrl: crawlForm.catalogUrl,
      ruleId: crawlForm.ruleId,
      maxChapters: Number(crawlForm.maxChapters) || 0,
      skipFilteredTitle: crawlForm.skipFilteredTitle,
    })
    state.success = `已导出 ${result.exportedCount} 章到 ${result.filePath}`
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.exporting = false
  }
}
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <p class="eyebrow">Wails + Vue</p>
        <h1>小说提取器</h1>
        <p class="subtitle">按站点规则抓取目录、正文和分页内容，并导出为 TXT。</p>
      </div>

      <section class="panel">
        <div class="panel-head">
          <h2>现有规则</h2>
          <button class="ghost" type="button" @click="resetRuleForm">新增</button>
        </div>

        <div class="rule-list">
          <button
            v-for="rule in state.rules"
            :key="rule.id"
            class="rule-chip"
            :class="{ active: crawlForm.ruleId === rule.id }"
            type="button"
            @click="pickRule(rule)"
          >
            <span>{{ rule.name }}</span>
            <small>{{ (rule.matchDomains ?? []).join(', ') }}</small>
          </button>
        </div>
      </section>

      <section class="panel compact">
        <h2>当前思路</h2>
        <p>目录页只负责拿到章节链接。</p>
        <p>正文页先提取正文，再跟着“下一页”抓同一章的后续分页。</p>
        <p>分页结束后才进入下一章，所以像二三书库这种站也能连续导出。</p>
      </section>
    </aside>

    <main class="content">
      <section class="hero">
        <div>
          <p class="eyebrow">抓取任务</p>
          <h2>先分析目录，再一键导出</h2>
        </div>
        <div v-if="state.progress" class="status-box">
          <strong>{{ state.progress.message }}</strong>
          <span v-if="state.progress.chapterTitle">{{ state.progress.chapterTitle }}</span>
          <span>{{ state.progress.current }}/{{ state.progress.total }}</span>
        </div>
      </section>

      <section class="panel task-grid">
        <label class="field span-2">
          <span>目录地址</span>
          <input v-model="crawlForm.catalogUrl" placeholder="输入目录页 URL" />
        </label>

        <label class="field">
          <span>站点规则</span>
          <select v-model="crawlForm.ruleId">
            <option v-for="rule in state.rules" :key="rule.id" :value="rule.id">
              {{ rule.name }}
            </option>
          </select>
        </label>

        <label class="field">
          <span>导出章节数</span>
          <input v-model="crawlForm.maxChapters" type="number" min="0" placeholder="0 表示全部" />
        </label>

        <label class="toggle span-2">
          <input v-model="crawlForm.skipFilteredTitle" type="checkbox" />
          <span>跳过“请假 / 总结 / 感言”等标题匹配的章节</span>
        </label>

        <div class="actions span-2">
          <button type="button" class="primary" :disabled="state.analyzing || state.loading" @click="analyzeCatalog">
            {{ state.analyzing ? '分析中...' : '分析目录' }}
          </button>
          <button type="button" class="accent" :disabled="state.exporting || state.loading" @click="exportNovel">
            {{ state.exporting ? '导出中...' : '导出 TXT' }}
          </button>
        </div>

        <p v-if="state.error" class="message error">{{ state.error }}</p>
        <p v-if="state.success" class="message success">{{ state.success }}</p>
      </section>

      <section class="two-col">
        <div class="panel">
          <div class="panel-head">
            <h2>规则编辑</h2>
            <span v-if="selectedRule">当前：{{ selectedRule.name }}</span>
          </div>

          <div class="form-grid">
            <label class="field">
              <span>规则 ID</span>
              <input v-model="ruleForm.id" placeholder="留空将自动生成" />
            </label>

            <label class="field">
              <span>规则名称</span>
              <input v-model="ruleForm.name" placeholder="例如：宙斯小说网" />
            </label>

            <label class="field span-2">
              <span>匹配域名</span>
              <textarea v-model="ruleForm.matchDomains" rows="3" placeholder="每行一个，如 zhswx.com"></textarea>
            </label>

            <label class="field">
              <span>目录章节选择器</span>
              <input v-model="ruleForm.catalogChapterLinkSelector" placeholder="td.chapterlist a" />
            </label>

            <label class="field">
              <span>正文选择器</span>
              <input v-model="ruleForm.chapterContentSelector" placeholder="#content" />
            </label>

            <label class="field">
              <span>目录标题文本</span>
              <input v-model="ruleForm.catalogSectionHeadingText" placeholder="可选，例如：《青山》正文" />
            </label>

            <label class="field">
              <span>目录区块选择器</span>
              <input v-model="ruleForm.catalogSectionContainer" placeholder="可选，例如：div.section-box" />
            </label>

            <label class="field">
              <span>下一页选择器</span>
              <input v-model="ruleForm.nextPageSelector" placeholder="a:contains('下一页')" />
            </label>

            <label class="field">
              <span>下一章选择器</span>
              <input v-model="ruleForm.nextChapterSelector" placeholder="a:contains('下一章')" />
            </label>

            <label class="field span-2">
              <span>清理选择器</span>
              <textarea v-model="ruleForm.contentCleanupSelectors" rows="3" placeholder="每行一个，例如 script"></textarea>
            </label>

            <label class="field span-2">
              <span>正文停止文本</span>
              <textarea v-model="ruleForm.contentStopTexts" rows="3" placeholder="每行一个，例如：（本章未完，请点击下一页继续阅读）"></textarea>
            </label>

            <label class="field span-2">
              <span>跳过标题正则</span>
              <textarea v-model="ruleForm.skipChapterTitlePatterns" rows="4" placeholder="每行一个，例如 请假"></textarea>
            </label>

            <label class="field span-2">
              <span>备注</span>
              <textarea v-model="ruleForm.notes" rows="3" placeholder="记录这个站的特殊规则"></textarea>
            </label>
          </div>

          <div class="actions">
            <button type="button" class="primary" :disabled="state.savingRule" @click="saveRule">
              {{ state.savingRule ? '保存中...' : '保存规则' }}
            </button>
            <button type="button" class="ghost danger" :disabled="!ruleForm.id" @click="removeRule">
              删除规则
            </button>
          </div>
        </div>

        <div class="panel preview">
          <div class="panel-head">
            <h2>目录预览</h2>
            <span v-if="state.analysis">{{ state.analysis.chapterCount }} 章</span>
          </div>

          <template v-if="state.analysis">
            <div class="preview-head">
              <strong>{{ state.analysis.novelTitle }}</strong>
              <p>{{ state.analysis.ruleName }}</p>
            </div>

            <div class="chapter-list">
              <article v-for="chapter in state.analysis.chapters.slice(0, 40)" :key="chapter.url" class="chapter-card">
                <strong>{{ chapter.title }}</strong>
                <small>{{ chapter.url }}</small>
              </article>
            </div>
          </template>

          <p v-else class="empty">点一次“分析目录”，这里会显示识别到的章节列表。</p>
        </div>
      </section>
    </main>
  </div>
</template>
