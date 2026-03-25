<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Card from 'primevue/card'
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { useAccounts } from '@/composables/useAccounts'
import {
  ACCOUNT_TYPES,
  getAccountTypeLabel,
  getAccountTypeIcon,
} from '@/types/account'
import type { Account } from '@/types/account'

const router = useRouter()
const { accounts: accountProviders } = useAccounts()

const searchText = ref('')

const ALL_TYPE = '__all__'
const STORAGE_KEY = 'etna_nav_active_type'

const typeOptions = computed(() => {
  return [
    { label: 'All', value: ALL_TYPE, icon: 'list' },
    ...Object.values(ACCOUNT_TYPES).map(t => ({
      label: getAccountTypeLabel(t),
      value: t,
      icon: getAccountTypeIcon(t),
    })),
  ]
})

const activeType = ref(ALL_TYPE)

onMounted(() => {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored && Object.values(ACCOUNT_TYPES).includes(stored as any)) {
    activeType.value = stored
  }
})

watch(activeType, (v) => {
  localStorage.setItem(STORAGE_KEY, v)
})

interface FlatAccount extends Account {
  providerName: string
  providerIcon?: string
}

const flatAccounts = computed<FlatAccount[]>(() => {
  if (!accountProviders.value) return []

  const flat: FlatAccount[] = []
  for (const provider of accountProviders.value) {
    for (const account of provider.accounts) {
      flat.push({
        ...account,
        providerName: provider.name,
        providerIcon: provider.icon
      })
    }
  }
  return flat
})

const accountCountByType = computed(() => {
  const counts: Record<string, number> = { [ALL_TYPE]: flatAccounts.value.length }
  for (const account of flatAccounts.value) {
    counts[account.type] = (counts[account.type] || 0) + 1
  }
  return counts
})

const filteredAccounts = computed<FlatAccount[]>(() => {
  let result = flatAccounts.value

  if (activeType.value !== ALL_TYPE) {
    result = result.filter(a => a.type === activeType.value)
  }

  if (searchText.value.trim()) {
    const search = searchText.value.toLowerCase()
    result = result.filter(account => {
      const accountName = account.name.toLowerCase()
      const providerName = account.providerName.toLowerCase()
      const typeLabel = getAccountTypeLabel(account.type).toLowerCase()

      return accountName.includes(search) ||
             providerName.includes(search) ||
             typeLabel.includes(search)
    })
  }

  return result
})

const onRowClick = (event: { data: FlatAccount }) => {
  router.push(`/entries/${event.data.id}`)
}
</script>

<template>
  <ResponsiveHorizontal :leftSidebarCollapsed="true">
    <template #default>
      <div class="p-3">
        <div class="mb-2">
          <h1 class="flex align-items-center gap-3 m-0 mb-2">
            <i class="ti ti-list-search text-primary"></i>
            Account Browser
          </h1>
        </div>

        <div class="grid">
          <!-- Left panel: Account types -->
          <div class="col-12 md:col-3">
            <Card>
              <template #title>Account Types</template>
              <template #content>
                <div class="type-list">
                  <Button
                    v-for="opt in typeOptions"
                    :key="opt.value"
                    :label="opt.label"
                    :icon="`ti ti-${opt.icon}`"
                    class="type-item"
                    :class="{ 'type-item-active': activeType === opt.value }"
                    text
                    @click="activeType = opt.value"
                  >
                    <template #default>
                      <i :class="`ti ti-${opt.icon}`" class="p-button-icon"></i>
                      <span class="p-button-label">{{ opt.label }}</span>
                      <span v-if="accountCountByType[opt.value]" class="type-count">
                        {{ accountCountByType[opt.value] }}
                      </span>
                    </template>
                  </Button>
                </div>
              </template>
            </Card>
          </div>

          <!-- Right panel: Filtered accounts -->
          <div class="col-12 md:col-9">
            <Card>
              <template #content>
                <div class="mb-3">
                  <InputText
                    v-model="searchText"
                    placeholder="Search accounts, providers, or types..."
                    class="w-full"
                  />
                </div>

                <DataTable
                  :value="filteredAccounts"
                  :rows="20"
                  :paginator="filteredAccounts.length > 20"
                  striped-rows
                  @row-click="onRowClick"
                  class="account-table"
                >
                  <Column field="icon" header="" style="width: 3rem">
                    <template #body="slotProps">
                      <i
                        v-if="slotProps.data.icon"
                        :class="`ti ti-${slotProps.data.icon}`"
                        class="text-xl"
                      />
                      <i
                        v-else-if="getAccountTypeIcon(slotProps.data.type)"
                        :class="`ti ti-${getAccountTypeIcon(slotProps.data.type)}`"
                        class="text-xl text-color-secondary"
                      />
                    </template>
                  </Column>

                  <Column field="name" header="Account Name" sortable>
                    <template #body="slotProps">
                      <span class="font-medium">{{ slotProps.data.name }}</span>
                    </template>
                  </Column>

                  <Column field="providerName" header="Provider" sortable>
                    <template #body="slotProps">
                      <div class="flex align-items-center gap-2">
                        <i
                          v-if="slotProps.data.providerIcon"
                          :class="`ti ti-${slotProps.data.providerIcon}`"
                          class="text-color-secondary"
                        />
                        <span>{{ slotProps.data.providerName }}</span>
                      </div>
                    </template>
                  </Column>

                  <Column field="type" header="Type" sortable>
                    <template #body="slotProps">
                      <span class="text-color-secondary">{{ getAccountTypeLabel(slotProps.data.type) }}</span>
                    </template>
                  </Column>

                  <Column field="currency" header="Currency" sortable>
                    <template #body="slotProps">
                      <span class="text-color-secondary">{{ slotProps.data.currency }}</span>
                    </template>
                  </Column>
                </DataTable>
              </template>
            </Card>
          </div>
        </div>
      </div>
    </template>
  </ResponsiveHorizontal>
</template>

<style scoped>
.type-list {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.type-item {
  justify-content: flex-start;
  padding: 0.5rem;
  width: 100%;
  border-radius: 4px;
}

.type-item :deep(.p-button-label) {
  font-weight: 500;
  flex: 1;
  text-align: left;
}

.type-item :deep(.p-button-icon) {
  font-size: 1.35rem;
}

.type-item-active {
  background-color: var(--primary-100);
  color: var(--primary-700);
}

.type-count {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-color-secondary);
  background-color: var(--surface-100);
  padding: 0.15rem 0.5rem;
  border-radius: 10px;
  margin-left: auto;
}

.type-item-active .type-count {
  background-color: var(--primary-200);
  color: var(--primary-700);
}

.account-table :deep(tbody tr) {
  cursor: pointer;
}

.account-table :deep(tbody tr:hover) {
  background-color: var(--surface-card);
}

.account-table :deep(th:last-child),
.account-table :deep(th:last-child .p-datatable-column-header-content),
.account-table :deep(td:last-child) {
  text-align: right;
  justify-content: flex-end;
}
</style>
