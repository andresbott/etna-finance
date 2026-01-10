<template>
  <button 
    :class="['btn', `btn--${variant}`, { 'btn--disabled': disabled }]"
    :disabled="disabled"
    @click="handleClick"
  >
    <slot></slot>
  </button>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  variant: {
    type: String,
    default: 'primary',
    validator: (value) => ['primary', 'secondary', 'danger'].includes(value)
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['click'])

const handleClick = (event) => {
  if (!props.disabled) {
    emit('click', event)
  }
}
</script>

<style scoped>
.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 500;
  transition: all 0.2s;
}

.btn--primary {
  background-color: #335c67;
  color: white;
}

.btn--primary:hover:not(.btn--disabled) {
  background-color: #2a4a53;
}

.btn--secondary {
  background-color: #e09f3e;
  color: white;
}

.btn--secondary:hover:not(.btn--disabled) {
  background-color: #c78933;
}

.btn--danger {
  background-color: #9e2a2b;
  color: white;
}

.btn--danger:hover:not(.btn--disabled) {
  background-color: #7e2122;
}

.btn--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>


