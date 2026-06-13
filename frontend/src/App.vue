<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  ApplyTranslation,
  GetEditorToolingStatus,
  GetGameStatus,
  InstallEditorTooling,
  LoadTranslationEditor,
  SaveTranslationEditor,
} from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'
import Button from './components/ui/button/Button.vue'
import Select from './components/ui/select/Select.vue'
import SelectContent from './components/ui/select/SelectContent.vue'
import SelectItem from './components/ui/select/SelectItem.vue'
import SelectTrigger from './components/ui/select/SelectTrigger.vue'
import SelectValue from './components/ui/select/SelectValue.vue'
import gameHeader from './assets/images/crime-scene-cleaner-header.jpg'

const targets = [
  { value: 'english', label: 'замість англійської' },
  { value: 'polish', label: 'замість польської' },
]

const selectedTarget = ref('english')
const loading = ref(true)
const applying = ref(false)
const view = ref('main')
const showEditorButton = ref(false)
const editorLoading = ref(false)
const editorSaving = ref(false)
const editorRows = ref([])
const editorProgress = ref({ stage: '', percent: 0, message: '' })
const status = ref({
  game: {
    installed: false,
    path: '',
    version: '',
    message: 'Перевіряємо Steam...',
  },
})
const notice = ref('')
const error = ref('')
let unsubscribeProgress = null

const isWailsRuntime = () => Boolean(window?.go?.main?.App)

const gameInfoText = computed(() => {
  if (loading.value) return 'перевіряємо гру...'
  if (!status.value?.game?.installed) return 'гра не знайдена'

  const version = status.value.game.version || 'невідома'
  const path = status.value.game.path || 'шлях не визначено'
  return `версія гри: ${version} · ${path}`
})

const gameMissing = computed(() => {
  return !loading.value && !status.value?.game?.installed
})

const canApply = computed(() => {
  return status.value?.game?.installed && !loading.value && !applying.value
})

const editorBusy = computed(() => editorLoading.value || editorSaving.value)

async function refreshStatus() {
  loading.value = true
  error.value = ''
  if (!isWailsRuntime()) {
    status.value = {
      game: {
        installed: false,
        path: '',
        version: '',
        message: 'Запустіть desktop-збірку Wails для перевірки Steam',
      },
    }
    loading.value = false
    return
  }
  try {
    status.value = await GetGameStatus()
  } catch (err) {
    error.value = formatError(err)
  } finally {
    loading.value = false
  }
}

async function applyTranslation() {
  if (!isWailsRuntime()) {
    error.value = 'Застосування перекладу доступне тільки у desktop-збірці'
    return
  }
  applying.value = true
  notice.value = ''
  error.value = ''
  try {
    const result = await ApplyTranslation({ target: selectedTarget.value })
    notice.value = result?.message || 'Переклад застосовано'
    await refreshStatus()
  } catch (err) {
    error.value = formatError(err)
  } finally {
    applying.value = false
  }
}

async function openEditor() {
  notice.value = ''
  error.value = ''
  if (!isWailsRuntime()) {
    error.value = 'Редактор перекладу доступний тільки у desktop-збірці'
    return
  }

  editorProgress.value = { stage: 'tooling', percent: 0, message: 'Перевіряємо інструменти' }
  let tooling = await GetEditorToolingStatus()
  if (!tooling.ready) {
    if (!tooling.pythonAvailable) {
      error.value = 'Python не знайдено. Встановіть Python і повторіть спробу.'
      return
    }
    const accepted = window.confirm('Для редагування bundle потрібно встановити UnityPy. Встановити зараз?')
    if (!accepted) {
      return
    }
    try {
      editorLoading.value = true
      tooling = await InstallEditorTooling()
    } catch (err) {
      error.value = formatError(err)
      return
    } finally {
      editorLoading.value = false
    }
    if (!tooling.ready) {
      error.value = tooling.message || 'Інструменти редактора не готові'
      return
    }
  }

  view.value = 'editor'
  await loadEditorRows()
}

async function loadEditorRows() {
  editorLoading.value = true
  editorRows.value = []
  editorProgress.value = { stage: 'load', percent: 0, message: 'Готуємо bundle' }
  try {
    const data = await LoadTranslationEditor()
    editorRows.value = data?.rows || []
    notice.value = `Завантажено рядків: ${editorRows.value.length}`
  } catch (err) {
    error.value = formatError(err)
  } finally {
    editorLoading.value = false
  }
}

async function saveEditorRows() {
  editorSaving.value = true
  notice.value = ''
  error.value = ''
  editorProgress.value = { stage: 'save', percent: 0, message: 'Готуємо збереження' }
  try {
    await SaveTranslationEditor(editorRows.value)
    notice.value = 'Переклад збережено та експортовано для English/Polish'
  } catch (err) {
    error.value = formatError(err)
  } finally {
    editorSaving.value = false
  }
}

function backToMain() {
  notice.value = ''
  error.value = ''
  view.value = 'main'
  editorRows.value = []
  editorProgress.value = { stage: '', percent: 0, message: '' }
}

function formatError(err) {
  if (!err) return 'Невідома помилка'
  if (typeof err === 'string') return err
  return err.message || JSON.stringify(err)
}

onMounted(() => {
  refreshStatus()
  if (isWailsRuntime()) {
    unsubscribeProgress = EventsOn('translation-editor-progress', (progress) => {
      editorProgress.value = progress
    })
  }
})

onUnmounted(() => {
  if (unsubscribeProgress) unsubscribeProgress()
})
</script>

<template>
  <main v-if="view === 'main'" class="app-shell">
    <section class="hero-panel">
      <img class="game-image" :src="gameHeader" alt="Crime Scene Cleaner" />
    </section>

    <section class="agreement-panel">
      <p>
        Автор патча не несе відповідальності за будь-яку шкоду, спричинену
        використанням цього патча. Ви застосовуєте його на власний страх і ризик.
      </p>
    </section>

    <section class="control-bar">
      <div class="left-controls">
        <Button v-if="showEditorButton" type="button" @click="openEditor">
          Редагувати переклад
        </Button>
      </div>

      <div class="right-controls">
        <Select v-model="selectedTarget">
          <SelectTrigger>
            <SelectValue placeholder="Оберіть мову" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="target in targets" :key="target.value" :value="target.value">
              {{ target.label }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Button type="button" :disabled="!canApply" @click="applyTranslation">
          {{ applying ? 'Застосування...' : 'Застосувати переклад' }}
        </Button>
      </div>
    </section>

    <footer class="footer">
      <span class="game-info" :class="{ missing: gameMissing }">{{ gameInfoText }}</span>
      <span class="footer-meta">
        <a href="mailto:shapovalov.v@gmail.com">shapovalov.v@gmail.com</a>
        <button class="version-button" type="button" @click="showEditorButton = !showEditorButton">
          v1.0.0
        </button>
      </span>
    </footer>

    <div v-if="notice || error" class="toast" :class="{ error: !!error }">
      {{ error || notice }}
    </div>
  </main>

  <main v-else class="editor-shell">
    <section class="editor-toolbar">
      <div class="editor-actions">
        <Button type="button" :disabled="editorBusy" @click="backToMain">Назад</Button>
        <Button type="button" :disabled="editorBusy || editorRows.length === 0" @click="saveEditorRows">
          {{ editorSaving ? 'Збереження...' : 'Зберегти' }}
        </Button>
      </div>
      <div class="editor-summary">
        <span>Рядків: {{ editorRows.length }}</span>
        <span v-if="editorProgress.message">{{ editorProgress.message }}</span>
      </div>
    </section>

    <section v-if="editorBusy" class="progress-panel">
      <div class="progress-label">
        <span>{{ editorProgress.message || 'Працюємо з bundle...' }}</span>
        <span>{{ editorProgress.percent }}%</span>
      </div>
      <div class="progress-track">
        <div class="progress-value" :style="{ width: `${editorProgress.percent}%` }"></div>
      </div>
    </section>

    <section class="editor-table-panel">
      <table class="translation-table">
        <thead>
          <tr>
            <th>Table</th>
            <th>ID</th>
            <th>Original</th>
            <th>Translation</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in editorRows" :key="`${row.table}:${row.id}`">
            <td>{{ row.table }}</td>
            <td>{{ row.id }}</td>
            <td class="original-text">{{ row.original }}</td>
            <td>
              <textarea v-model="row.text" rows="2"></textarea>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!editorBusy && editorRows.length === 0" class="empty-editor">
        Немає рядків для редагування.
      </div>
    </section>

    <div v-if="notice || error" class="toast" :class="{ error: !!error }">
      {{ error || notice }}
    </div>
  </main>
</template>
