<template>
  <Teleport to="body">
    <div v-if="open" class="modal active" @click.self="$emit('close')">
      <div class="modal-content">
        <h2>{{ title }}</h2>
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{ open: boolean; title: string }>();
defineEmits<{ close: [] }>();
</script>
