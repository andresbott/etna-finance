<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from "vue";

type MobileBehavior = 'menu' | 'content';

interface Props {
  leftSidebarCollapsed?: boolean;
  rightSidebarCollapsed?: boolean;
  mobileBehavior?: MobileBehavior;
}

const props = withDefaults(defineProps<Props>(), {
  leftSidebarCollapsed: false,
  rightSidebarCollapsed: false,
  mobileBehavior: 'menu',
});

type Emits = {
  'update:leftSidebarCollapsed': [value: boolean];
  'update:rightSidebarCollapsed': [value: boolean];
}

const emit = defineEmits<Emits>();

const windowWidth = ref<number>(window.innerWidth);
const isMobile = computed<boolean>(() => windowWidth.value < 600);
const isDesktop = computed<boolean>(() => windowWidth.value >= 900);

const leftSidebarStatus = computed({
  get(): boolean | undefined {
    return props.leftSidebarCollapsed;
  },
  set(val: boolean): void {
    emit("update:leftSidebarCollapsed", val);
  },
});

const rightSidebarStatus = computed({
  get(): boolean | undefined {
    return props.rightSidebarCollapsed;
  },
  set(val: boolean): void {
    emit("update:rightSidebarCollapsed", val);
  },
});

const updateWindowWidth = (): void => {
  windowWidth.value = window.innerWidth;
};

onMounted(() => {
  window.addEventListener("resize", updateWindowWidth);
});

onUnmounted(() => {
  window.removeEventListener("resize", updateWindowWidth);
});

interface StyleObject {
  [key: string]: string | undefined;
}

const containerStyle = computed<StyleObject>(() => {
  if (isDesktop.value) {
    return {
      maxWidth: "1100px",
      margin: "0 auto",
    };
  }
  return {};
});

const leftSidebarStyle = computed<StyleObject>(() => {
  if (isMobile.value && props.mobileBehavior === "menu") {
    return {
      display: leftSidebarStatus.value ? "none" : "block",
      position: "fixed",
      top: "0",
      left: "0",
      height: "100vh",
      zIndex: "1000",
      backgroundColor: "var(--surface-overlay)",
    };
  }
  return {
    display: "block",
    width: leftSidebarStatus.value ? "0px" : "auto",
  };
});

const rightSidebarStyle = computed<StyleObject>(() => {
  if (isMobile.value && props.mobileBehavior === "menu") {
    return {
      display: rightSidebarStatus.value ? "none" : "block",
      position: "fixed",
      top: "0",
      right: "0",
      height: "100vh",
      zIndex: "1000",
      backgroundColor: "var(--surface-overlay)",
    };
  }
  return {
    display: "block",
    width: rightSidebarStatus.value ? "0px" : "auto",
  };
});
</script>

<template>
  <div class="cc-rh" :style="containerStyle">
    <div class="cc-rh-left" :style="leftSidebarStyle">
      <slot name="left"></slot>
    </div>

    <div class="cc-rh-main">
      <slot></slot>
    </div>

    <div class="cc-rh-right" :style="rightSidebarStyle">
      <slot name="right"></slot>
    </div>
  </div>
</template>

<style>
.cc-rh {
  display: grid;
  width: 100%;
  transition: all 0.3s ease;
}

/* Desktop layout (900px and above) */
@media (min-width: 900px) {
  .cc-rh {
    grid-template-columns: auto 1fr auto;
    gap: 1rem;
  }
}

/* Tablet layout (600px to 899px) */
@media (min-width: 600px) and (max-width: 899px) {
  .cc-rh {
    grid-template-columns: auto 1fr auto;
    gap: 1rem;
  }
}

/* Mobile layout (below 600px) */
@media (max-width: 599px) {
  .cc-rh {
    grid-template-columns: 1fr;
  }

  .cc-rh.menu-behavior .cc-rh-left,
  .cc-rh.menu-behavior .cc-rh-right {
    position: fixed;
    top: 0;
    height: 100vh;
    z-index: 1000;
    background-color: var(--surface-overlay);
  }

  .cc-rh.menu-behavior .cc-rh-left {
    left: 0;
  }

  .cc-rh.menu-behavior .cc-rh-right {
    right: 0;
  }

  .cc-rh.content-behavior {
    grid-template-rows: auto 1fr auto;
  }
}

.cc-rh-left,
.cc-rh-right {
  transition: all 0.3s ease;
  overflow: hidden;
}

.cc-rh-main {
  width: 100%;
  min-width: 0; /* Prevents content from overflowing */
}
</style>
