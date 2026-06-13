<script setup lang="ts">
import { computed, onMounted } from "vue";

interface Props {
  fullHeight?: boolean;
  centerContent?: boolean;
  contentWidth?: number | string | null;
  contentMaxWidth?: number | string | null;
  verticalCenterContent?: boolean;
  minHeight?: number | string | null;
  leftMinWidth?: number | string | null;
  leftMaxWidth?: number | string | null;
  rightMinWidth?: number | string | null;
  rightMaxWidth?: number | string | null;
  containerWidth?: number | string;
  containerMaxWidth?: number | string | null;
  sidebarOpen?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  fullHeight: false,
  centerContent: false,
  contentWidth: null,
  contentMaxWidth: null,
  verticalCenterContent: false,
  minHeight: null,
  leftMinWidth: null,
  leftMaxWidth: null,
  rightMinWidth: null,
  rightMaxWidth: null,
  containerWidth: "100%",
  containerMaxWidth: null,
  sidebarOpen: true,
});

type Emits = {
  'update:sidebarOpen': [value: boolean];
}

const emit = defineEmits<Emits>();

// Create a computed property for two-way binding with sidebarOpen.
const sidebarStatus = computed({
  get(): boolean | undefined {
    return props.sidebarOpen;
  },
  set(val: boolean): void {
    emit("update:sidebarOpen", val);
  },
});

// On mount, if sidebarOpen is not provided by the parent, determine a default based on window width.
onMounted(() => {
  if (props.sidebarOpen === null) {
    sidebarStatus.value = window.innerWidth >= 768;
  }
});

// Helper function to convert a number or string to a valid CSS value.
function convertToCss(value: number | string | null | undefined): string | undefined {
  if (!value) return undefined;
  return typeof value === "number" ? value + "px" : value;
}

// Determine the sidebar's fixed width when open.
const leftWidth = computed((): string => {
  return (
    convertToCss(props.leftMinWidth) ||
    convertToCss(props.leftMaxWidth) ||
    "auto"
  );
});

// Compute the overall container style.
// When the sidebar is closed, its column width is set to 0px.
const containerStyle = computed((): Record<string, string | undefined> => {
  return {
    gridTemplateColumns: `${sidebarStatus.value ? leftWidth.value : "0px"} 1fr auto`,
    width: convertToCss(props.containerWidth) || "100%",
    maxWidth: convertToCss(props.containerMaxWidth),
  };
});
</script>

<template>
  <!-- The container uses containerStyle to set grid columns and overall sizing -->
  <div
    class="cc-horizontal"
    :class="{ 'full-height': fullHeight }"
    :style="containerStyle"
  >
    <!-- Left Sidebar -->
    <div class="c-left">
      <slot name="left"></slot>
    </div>
    <div
      class="cc-main"
      :class="{
        'center-content': centerContent,
        'vertical-center': verticalCenterContent,
      }"
      :style="{ minHeight: minHeight ?? undefined }"
    >
      <div
        :style="{
          maxWidth: contentMaxWidth ?? undefined,
          width: contentWidth ? contentWidth : undefined,
        }"
      >
        <slot></slot>
      </div>
    </div>
    <!-- Right Section -->
    <div
      class="c-right"
      :style="{ minWidth: rightMinWidth ?? undefined, maxWidth: rightMaxWidth ?? undefined }"
    >
      <slot name="right"></slot>
    </div>
  </div>
</template>

<style>
/* Animate the grid-template-columns to smoothly collapse/expand the sidebar */
.cc-horizontal {
  width: 100%;
  display: grid;
  grid-template-columns: auto 1fr auto;
  transition: grid-template-columns 0.3s ease;
}
.cc-horizontal.full-height {
  height: 100%;
}

.cc-main {
  display: grid;
  place-items: start;
  width: auto;
}
.cc-main.center-content {
  place-items: start center;
}
.cc-main.vertical-center,
.c-right,
.c-left {
  align-items: center;
}
.cc-main > div {
  /* height: 100%; */
}
.cc-main.vertical-center > div {
  height: auto;
}

.c-left,
.c-right {
  display: grid;
  grid-auto-flow: column;
}
</style>
