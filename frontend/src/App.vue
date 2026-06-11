<script setup>
import { computed, onMounted, ref } from 'vue'
import { ApplyTranslation, GetGameStatus } from '../wailsjs/go/main/App'
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

function formatError(err) {
  if (!err) return 'Невідома помилка'
  if (typeof err === 'string') return err
  return err.message || JSON.stringify(err)
}

onMounted(refreshStatus)
</script>

<template>
  <main class="app-shell">
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
      <div class="left-controls" aria-hidden="true"></div>

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
        <span>v1.0.0</span>
      </span>
    </footer>

    <div v-if="notice || error" class="toast" :class="{ error: !!error }">
      {{ error || notice }}
    </div>
  </main>
</template>
