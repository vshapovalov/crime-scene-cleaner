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
  'inline-flex h-9 min-w-[210px] items-center justify-center border border-[#9a9a9a] bg-[#eeeeee] px-[18px] text-base text-[#101010] outline-none transition-opacity disabled:cursor-not-allowed disabled:opacity-60',
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
