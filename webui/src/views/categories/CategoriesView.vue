<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { RouterView } from 'vue-router'

const route = useRoute()
const router = useRouter()

const navItems = [
    {
        group: 'Categories',
        items: [
            { label: 'Expense', icon: 'pi-folder-open', name: 'expense-categories' },
            { label: 'Income', icon: 'pi-folder-open', name: 'income-categories' },
        ]
    },
    {
        group: 'Automation',
        items: [
            { label: 'Matching Rules', icon: 'pi-bolt', name: 'category-rules' },
        ]
    }
]

const isActive = (name: string) => route.name === name
</script>

<template>
    <div class="main-app-content">
        <h1>Categories</h1>
        <div class="grid">
            <!-- Left nav panel -->
            <div class="col-12 md:col-3">
                <div class="categories-nav">
                    <template v-for="section in navItems" :key="section.group">
                        <div class="nav-group-label">{{ section.group }}</div>
                        <div
                            v-for="item in section.items"
                            :key="item.name"
                            class="nav-item"
                            :class="{ 'nav-item--active': isActive(item.name) }"
                            @click="router.push({ name: item.name })"
                        >
                            <i :class="['pi', item.icon]"></i>
                            <span>{{ item.label }}</span>
                        </div>
                    </template>
                </div>
            </div>

            <!-- Right content panel -->
            <div class="col-12 md:col-9">
                <RouterView />
            </div>
        </div>
    </div>
</template>

<style scoped>
.categories-nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0.5rem 0;
}

.nav-group-label {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-color-secondary);
    padding: 0.75rem 0.75rem 0.25rem;
}

.nav-item {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.6rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    border-left: 3px solid transparent;
    color: var(--text-color-secondary);
    transition: background 0.15s;
}

.nav-item:hover {
    background: var(--surface-50);
    color: var(--text-color);
}

.nav-item--active {
    background: var(--surface-100);
    border-left-color: var(--primary-color);
    color: var(--text-color);
    font-weight: 600;
}
</style>
