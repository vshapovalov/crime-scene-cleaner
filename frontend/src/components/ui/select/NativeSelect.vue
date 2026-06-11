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
    :class="cn('h-9 min-w-[176px] cursor-pointer border border-[#9a9a9a] bg-[#eeeeee] px-[18px] text-[13px] text-[#101010] outline-none', $props.class)"
    @change="emit('update:modelValue', $event.target.value)"
  >
    <slot />
  </select>
</template>
