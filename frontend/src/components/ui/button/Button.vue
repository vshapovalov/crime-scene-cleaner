<script setup>
import { computed } from 'vue'
import { cva } from 'class-variance-authority'
import { cn } from '../../../lib/utils'

const props = defineProps({
  variant: {
    type: String,
    default: 'default',
  },
  class: {
    type: String,
    default: '',
  },
})

const buttonVariants = cva(
  'inline-flex h-10 min-w-[210px] items-center justify-center rounded-md border border-zinc-200 bg-zinc-950 px-5 text-sm font-medium text-white shadow-sm outline-none transition-colors hover:bg-zinc-800 focus-visible:ring-2 focus-visible:ring-zinc-400 disabled:cursor-not-allowed disabled:opacity-50',
  {
    variants: {
      variant: {
        default: '',
        primary: 'min-w-[222px]',
      },
    },
  },
)

const classes = computed(() => cn(buttonVariants({ variant: props.variant }), props.class))
</script>

<template>
  <button :class="classes">
    <slot />
  </button>
</template>
