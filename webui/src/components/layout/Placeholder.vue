<template>
  <div
    class="container"
    :style="containerStyle"
    @mouseover="handleHover(true)"
    @mouseleave="handleHover(false)"
  >
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

interface Props {
  width?: number | string;
  height?: number | string;
  color?: number | string;
  showBorder?: boolean;
  showBackgroundOnHover?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  width: "100%",
  height: "100%",
  color: "",
  showBorder: false,
  showBackgroundOnHover: false,
});

defineOptions({ name: 'LayoutPlaceholder' });

const pastelColors = [
  "#FFB3BA", "#FFDFBA", "#FFFFBA", "#BAFFC9", "#BAE1FF",
  "#D4A5A5", "#FFD1DC", "#E2CFCF", "#B5EAD7", "#FFE156",
  "#F7B7A3", "#FF8B94", "#A7C7E7", "#C5E1A5", "#F3C1C6",
  "#F5E6A7", "#B0E0E6", "#D8BFD8", "#D4A017", "#E0BBE4",
  "#957DAD", "#D291BC", "#FEC8D8", "#FFDFD3", "#E6E6FA",
  "#C2DFFF", "#B5D8CC", "#FFC3A0", "#FFDAC1", "#D4A5A5",
  "#A2D5C6", "#E6A4B4", "#BFD8D2", "#F5CBA7", "#F7D7A3",
];

const isHovered = ref(false);

const widthValue = computed((): string => {
  return typeof props.width === "number" ? `${props.width}px` : props.width as string;
});

const heightValue = computed((): string => {
  return typeof props.height === "number" ? `${props.height}px` : props.height as string;
});

const backgroundColor = computed((): string => {
  if (!props.color) return "transparent";
  if (
    typeof props.color === "string" &&
    /^#([0-9A-F]{3}){1,2}$/i.test(props.color)
  ) {
    return props.color;
  }
  if (
    typeof props.color === "number" &&
    props.color >= 0 &&
    props.color < pastelColors.length
  ) {
    return pastelColors[props.color];
  }
  return "transparent";
});

const containerStyle = computed((): Record<string, string> => {
  return {
    backgroundColor: props.showBackgroundOnHover
      ? isHovered.value
        ? backgroundColor.value
        : "transparent"
      : backgroundColor.value,
    width: widthValue.value,
    height: heightValue.value,
    border: props.showBorder ? "1px dashed #636363" : "none",
    transition: "background-color 0.3s ease-in-out",
  };
});

function handleHover(state: boolean): void {
  isHovered.value = state;
}
</script>

<style scoped>
.container {
  display: flex;
  justify-content: center;
  align-items: center;
  text-align: center;
}
</style>
