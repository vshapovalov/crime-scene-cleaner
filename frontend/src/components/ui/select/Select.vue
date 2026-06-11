<script setup>
import { SelectRoot, useForwardPropsEmits } from 'radix-vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: undefined,
  },
})

const emits = defineEmits(['update:modelValue'])
const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <SelectRoot v-bind="forwarded">
    <slot />
  </SelectRoot>
</template>
