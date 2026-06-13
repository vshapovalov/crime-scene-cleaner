<script setup>
import { computed } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  modelValue: {
    type: String,
    default: '',
  },
  class: {
    type: String,
    default: '',
  },
})

const emits = defineEmits(['update:modelValue'])

const classes = computed(() => cn(
  'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
  props.class,
))
</script>

<template>
  <input
    :class="classes"
    :value="modelValue"
    @input="emits('update:modelValue', $event.target.value)"
  />
</template>
