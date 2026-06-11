<script setup>
import { cn } from '../../../lib/utils'

defineProps({
  modelValue: {
    type: String,
    required: true,
  },
  class: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['update:modelValue'])
</script>

<template>
  <select
    :value="modelValue"
    :class="cn('h-10 min-w-[190px] cursor-pointer rounded-md border border-zinc-200 bg-white px-3 text-sm text-zinc-950 shadow-sm outline-none transition-colors focus-visible:ring-2 focus-visible:ring-zinc-400', $props.class)"
    @change="emit('update:modelValue', $event.target.value)"
  >
    <slot />
  </select>
</template>
