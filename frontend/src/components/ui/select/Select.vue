<script setup>
import {
  SelectContent,
  SelectIcon,
  SelectItem,
  SelectItemIndicator,
  SelectItemText,
  SelectPortal,
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectViewport,
} from 'radix-vue'
import { cn } from '../../../lib/utils'

defineProps({
  modelValue: {
    type: String,
    required: true,
  },
  items: {
    type: Array,
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
  <SelectRoot :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <SelectTrigger
      :class="cn('inline-flex h-10 min-w-[206px] items-center justify-between gap-3 rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-950 shadow-sm outline-none transition-colors hover:bg-zinc-50 focus:ring-2 focus:ring-zinc-400 disabled:cursor-not-allowed disabled:opacity-50', $props.class)"
      aria-label="Мова для заміни"
    >
      <SelectValue />
      <SelectIcon class="select-chevron" aria-hidden="true">⌄</SelectIcon>
    </SelectTrigger>

    <SelectPortal>
      <SelectContent
        class="z-50 min-w-[206px] overflow-hidden rounded-md border border-zinc-200 bg-white text-zinc-950 shadow-lg"
        position="popper"
        :side-offset="6"
      >
        <SelectViewport class="p-1">
          <SelectItem
            v-for="item in items"
            :key="item.value"
            :value="item.value"
            class="relative flex h-9 cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none data-[highlighted]:bg-zinc-100 data-[highlighted]:text-zinc-950 data-[state=checked]:font-medium"
          >
            <span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
              <SelectItemIndicator>✓</SelectItemIndicator>
            </span>
            <SelectItemText>{{ item.label }}</SelectItemText>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
