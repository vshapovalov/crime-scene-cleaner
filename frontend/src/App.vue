<script setup>
import { computed, onMounted, ref } from 'vue'
import { ApplyTranslation, GetGameStatus } from '../wailsjs/go/main/App'
import Button from './components/ui/button/Button.vue'
import NativeSelect from './components/ui/select/NativeSelect.vue'

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

const versionText = computed(() => {
  const version = status.value?.game?.version
  return version ? `версія гри: ${version}` : 'версія гри: не знайдено'
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
      <div class="game-logo">Crime Scene Cleaner</div>
      <div class="subtitle">Українська локалізація</div>
    </section>

    <section class="agreement-panel">
      <h1>угода з користувачем</h1>
      <p>
        Патч замінює вибраний текстовий ресурс гри на український переклад.
        Оригінальна озвучка та звукові банки не змінюються.
      </p>
      <div class="game-status" :class="{ missing: !status.game.installed }">
        <span>{{ status.game.message }}</span>
        <small v-if="status.game.path">{{ status.game.path }}</small>
      </div>
    </section>

    <section class="control-bar">
      <div class="left-controls" aria-hidden="true"></div>

      <div class="right-controls">
        <label class="sr-only" for="target-language">Мова для заміни</label>
        <NativeSelect id="target-language" v-model="selectedTarget">
          <option v-for="target in targets" :key="target.value" :value="target.value">
            {{ target.label }}
          </option>
        </NativeSelect>
        <Button variant="primary" type="button" :disabled="!canApply" @click="applyTranslation">
          {{ applying ? 'Застосування...' : 'Застосувати переклад' }}
        </Button>
      </div>
    </section>

    <footer class="footer">
      <span>{{ versionText }}</span>
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
