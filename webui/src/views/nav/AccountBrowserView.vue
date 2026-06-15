<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useRouter } from 'vue-router'
import Column from 'primevue/column'
import TreeTable from 'primevue/treetable'
import InputText from 'primevue/inputtext'
import MultiSelect from 'primevue/multiselect'
import Button from 'primevue/button'
import Card from 'primevue/card'
import { ResponsiveHorizontal } from '@/components/layout'
import { useAccounts } from '@/composables/useAccounts'
import { useSettingsStore } from '@/store/settingsStore'
import AccountDialog from '@/views/accounts/AccountDialog.vue'
import AccountProviderDialog from '@/views/accounts/AccountProviderDialog.vue'
import CsvProfileDialog from '@/views/csvimport/CsvProfileDialog.vue'
import DeleteDialog from '@/components/common/ConfirmDialog.vue'
import { getAccountBalances } from '@/lib/api/report'
import { toLocalDateString } from '@/utils/date'
import { formatAmount } from '@/utils/currency'
import {
  ACCOUNT_TYPES,
  getAccountTypeLabel,
  getAccountTypeIcon,
} from '@/types/account'
import type { Account } from '@/types/account'

const router = useRouter()
const settings = useSettingsStore()

// Account types that only make sense when the investment-instruments feature is
// enabled; hidden otherwise (mirrors the old settings/accounts behavior).
const instrumentAccountTypes: string[] = [ACCOUNT_TYPES.INVESTMENT, ACCOUNT_TYPES.RESTRICTED_STOCK]

const {
  accounts: accountProviders,
  deleteAccount,
  deleteAccountProvider,
  updateAccount,
} = useAccounts()

// Current balance per account, fetched in one request and re-fetched whenever
// the set of accounts changes.
const allAccountIds = computed(() =>
  (accountProviders.value ?? []).flatMap(p => p.accounts.map(a => a.id)).sort((a, b) => a - b)
)

const { data: balances } = useQuery({
  queryKey: ['account-balances', allAccountIds],
  queryFn: () => getAccountBalances(allAccountIds.value, toLocalDateString(new Date())),
  enabled: computed(() => allAccountIds.value.length > 0),
})

function accountBalance(id: number): number {
  return balances.value?.[id] ?? 0
}

const searchText = ref('')
const selectedProviders = ref<string[]>([])
const selectedTypes = ref<string[]>([])
const selectedCurrencies = ref<string[]>([])

const providerOptions = computed(() => {
  if (!accountProviders.value) return []
  return accountProviders.value.map(p => ({ label: p.name, value: p.name }))
})

const typeOptions = computed(() =>
  Object.values(ACCOUNT_TYPES)
    .filter(t => settings.investmentInstruments || !instrumentAccountTypes.includes(t))
    .map(t => ({
      label: getAccountTypeLabel(t),
      value: t,
    }))
)

const currencyOptions = computed(() => {
  const currencies = new Set<string>()
  for (const provider of accountProviders.value ?? []) {
    for (const account of provider.accounts) {
      if (account.currency) currencies.add(account.currency)
    }
  }
  return Array.from(currencies)
    .sort()
    .map(c => ({ label: c, value: c }))
})

const hasActiveFilters = computed(() =>
  searchText.value.trim() !== '' ||
  selectedProviders.value.length > 0 ||
  selectedTypes.value.length > 0 ||
  selectedCurrencies.value.length > 0
)

// Filters live in a collapsible bar toggled by the filter button (mirrors the
// entries view). Start expanded when filters are already active (e.g. on reload
// with query state in the future).
const filtersExpanded = ref(hasActiveFilters.value)

function accountMatches(account: Account, providerName: string): boolean {
  if (!settings.investmentInstruments && instrumentAccountTypes.includes(account.type)) {
    return false
  }
  if (selectedTypes.value.length > 0 && !selectedTypes.value.includes(account.type)) {
    return false
  }
  if (selectedCurrencies.value.length > 0 && !selectedCurrencies.value.includes(account.currency)) {
    return false
  }
  if (searchText.value.trim()) {
    const search = searchText.value.toLowerCase()
    const matchesSearch =
      account.name.toLowerCase().includes(search) ||
      providerName.toLowerCase().includes(search) ||
      getAccountTypeLabel(account.type).toLowerCase().includes(search) ||
      (account.currency || '').toLowerCase().includes(search)
    if (!matchesSearch) return false
  }
  return true
}

// TreeTable data: provider parent nodes with their (filtered) accounts as
// children. Providers with no matching accounts are dropped.
const treeTableData = computed(() => {
  if (!accountProviders.value) return []

  const tree = []
  for (const provider of accountProviders.value) {
    if (selectedProviders.value.length > 0 && !selectedProviders.value.includes(provider.name)) {
      continue
    }

    const children = provider.accounts
      .filter(account => accountMatches(account, provider.name))
      .map(account => ({
        key: `account-${account.id}`,
        data: { kind: 'account', providerId: provider.id, ...account },
      }))

    if (children.length === 0) continue

    tree.push({
      key: `provider-${provider.id}`,
      data: {
        kind: 'provider',
        id: provider.id,
        name: provider.name,
        description: provider.description,
        icon: provider.icon,
      },
      children,
    })
  }
  return tree
})

// Sum of account balances grouped by currency, over only the accounts visible
// in the (filtered) tree. Sorted by currency for a stable display order.
const totalsByCurrency = computed(() => {
  const totals: Record<string, number> = {}
  for (const provider of treeTableData.value) {
    for (const child of provider.children ?? []) {
      const currency = child.data.currency || 'CHF'
      totals[currency] = (totals[currency] ?? 0) + accountBalance(child.data.id)
    }
  }
  return Object.entries(totals)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([currency, total]) => ({ currency, total }))
})

// Auto-expand all provider nodes whenever the filtered tree changes.
const expandedKeys = ref<Record<string, boolean>>({})
watch(treeTableData, (data) => {
  expandedKeys.value = data.reduce((acc, node) => {
    acc[node.key] = true
    return acc
  }, {} as Record<string, boolean>)
}, { immediate: true })

function clearFilters() {
  searchText.value = ''
  selectedProviders.value = []
  selectedTypes.value = []
  selectedCurrencies.value = []
}

// Toggle the filter bar. Collapsing it clears any active filters so hidden
// filters can't keep silently narrowing the list.
function toggleFilters() {
  filtersExpanded.value = !filtersExpanded.value
  if (!filtersExpanded.value) clearFilters()
}

function openAccount(id: number) {
  router.push(`/entries/${id}`)
}

// --- Add / edit / delete actions (mirrors settings/accounts) ---
const selectedItem = ref<any>(null)

const accountDialogVisible = ref(false)
const isEdit = ref(false)
const selectedAccount = ref<any>(null)

const providerDialogVisible = ref(false)
const isEditProvider = ref(false)
const selectedProvider = ref<any>(null)

const deleteAccountDialogVisible = ref(false)
const deleteProviderDialogVisible = ref(false)

function openNewProviderDialog() {
  selectedProvider.value = null
  isEditProvider.value = false
  providerDialogVisible.value = true
}

function editProvider(provider: any) {
  selectedProvider.value = provider
  isEditProvider.value = true
  providerDialogVisible.value = true
}

function addAccountToProvider(provider: any) {
  selectedAccount.value = {
    providerId: provider.data.id,
    icon: 'wallet',
  }
  isEdit.value = false
  accountDialogVisible.value = true
}

function editAccount(account: any) {
  selectedAccount.value = account
  isEdit.value = true
  accountDialogVisible.value = true
}

function showDeleteAccountDialog(account: any) {
  selectedItem.value = account
  deleteAccountDialogVisible.value = true
}

function showDeleteProviderDialog(provider: any) {
  selectedItem.value = provider
  deleteProviderDialogVisible.value = true
}

async function handleDeleteAccount() {
  if (selectedItem.value) {
    await deleteAccount(selectedItem.value.id)
    deleteAccountDialogVisible.value = false
  }
}

async function handleDeleteProvider() {
  if (selectedItem.value) {
    await deleteAccountProvider(selectedItem.value.id)
    deleteProviderDialogVisible.value = false
  }
}

async function toggleFavorite(account: any) {
  await updateAccount({
    id: account.id,
    favorite: !account.favorite,
  })
}

const profileDialogVisible = ref(false)
const profileDialogAccount = ref<any>(null)

function editImportProfile(account: any) {
  profileDialogAccount.value = account
  profileDialogVisible.value = true
}
</script>

<template>
  <ResponsiveHorizontal :leftSidebarCollapsed="true">
    <template #default>
      <div class="p-3">
        <div class="toolbar mb-3">
          <h1 class="flex align-items-center gap-3 m-0">
            <i class="ti ti-building-bank text-primary"></i>
            Accounts
          </h1>
          <div class="toolbar-actions">
            <Button
              :icon="filtersExpanded ? 'ti ti-filter-off' : 'ti ti-filter'"
              :severity="filtersExpanded ? 'primary' : 'secondary'"
              :outlined="!filtersExpanded"
              class="toolbar-btn"
              @click="toggleFilters"
              v-tooltip.bottom="'Filters'"
            />
            <Button
              label="Add provider"
              icon="ti ti-plus"
              class="toolbar-btn"
              @click="openNewProviderDialog"
            />
          </div>
        </div>

        <div v-if="filtersExpanded" class="filter-row mb-3">
          <InputText
            v-model="searchText"
            placeholder="Search accounts, providers, or types..."
            class="filter-input-search"
          />
          <MultiSelect
            v-model="selectedProviders"
            :options="providerOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Provider"
            class="filter-input"
            :showToggleAll="false"
            scrollHeight="20rem"
          />
          <MultiSelect
            v-model="selectedTypes"
            :options="typeOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Type"
            class="filter-input"
            :showToggleAll="false"
            scrollHeight="20rem"
          />
          <MultiSelect
            v-model="selectedCurrencies"
            :options="currencyOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Currency"
            class="filter-input"
            :showToggleAll="false"
            scrollHeight="20rem"
          />
          <Button
            v-if="hasActiveFilters"
            label="Clear"
            severity="secondary"
            text
            size="small"
            @click="clearFilters"
          />
        </div>

        <Card>
          <template #content>
            <TreeTable
              :value="treeTableData"
              :expandedKeys="expandedKeys"
              @update:expandedKeys="expandedKeys = $event"
              class="account-table p-treetable-sm"
            >
              <Column field="name" header="Account Name" expander>
                <template #body="{ node }">
                  <div
                    class="flex align-items-center gap-2"
                    :class="{ 'account-leaf': node.data.kind === 'account' }"
                    @click="node.data.kind === 'account' && openAccount(node.data.id)"
                  >
                    <i
                      v-if="node.data.icon"
                      :class="[`ti ti-${node.data.icon}`, 'text-xl', { 'text-color-secondary': node.data.kind === 'provider' }]"
                    />
                    <i
                      v-else-if="node.data.kind === 'account' && getAccountTypeIcon(node.data.type)"
                      :class="`ti ti-${getAccountTypeIcon(node.data.type)}`"
                      class="text-xl text-color-secondary"
                    />
                    <span :class="{ 'font-medium account-link': node.data.kind === 'account', 'font-bold': node.data.kind === 'provider' }">
                      {{ node.data.name }}
                    </span>
                    <i
                      v-if="node.data.kind === 'account' && node.data.notes?.trim()"
                      class="ti ti-help-circle text-color-secondary ml-2 cursor-help"
                      v-tooltip.right="node.data.notes"
                      @click.stop
                    />
                  </div>
                </template>
              </Column>

              <Column field="type" header="Type">
                <template #body="{ node }">
                  <span v-if="node.data.kind === 'account'" class="flex align-items-center gap-2 text-color-secondary">
                    <i :class="['ti', `ti-${getAccountTypeIcon(node.data.type)}`]"></i>
                    {{ getAccountTypeLabel(node.data.type) }}
                  </span>
                </template>
              </Column>

              <Column field="balance" header="Balance" class="bal-col">
                <template #body="{ node }">
                  <template v-if="node.data.kind === 'account'">
                    <span
                      class="font-semibold"
                      :style="accountBalance(node.data.id) < 0 ? { color: 'var(--c-red-600)' } : {}"
                    >{{ formatAmount(accountBalance(node.data.id)) }}</span>
                    <span
                      :class="['ml-1', accountBalance(node.data.id) < 0 ? '' : 'text-500']"
                      :style="accountBalance(node.data.id) < 0 ? { color: 'var(--c-red-600)' } : {}"
                    >{{ node.data.currency }}</span>
                  </template>
                </template>
              </Column>

              <Column bodyClass="actions-cell" style="width: 1%; white-space: nowrap">
                <template #body="{ node }">
                  <div
                    class="flex gap-2 justify-content-end w-full"
                    :class="{ 'actions-row--indent': node.data.kind === 'account' }"
                  >
                    <Button
                      v-if="node.data.kind === 'account'"
                      :icon="node.data.favorite ? 'ti ti-filled ti-star' : 'ti ti-star'"
                      text
                      rounded
                      class="p-1"
                      :class="{ 'favorite-active': node.data.favorite }"
                      @click.stop="toggleFavorite(node.data)"
                      v-tooltip.top="node.data.favorite ? 'Remove from favorites' : 'Add to favorites'"
                    />
                    <Button
                      v-if="node.data.kind === 'account'"
                      :icon="node.data.importProfileId ? 'ti ti-file-pencil' : 'ti ti-file-import'"
                      text
                      rounded
                      class="p-1"
                      @click.stop="editImportProfile(node.data)"
                      v-tooltip.top="node.data.importProfileId ? 'Edit CSV import profile' : 'Add CSV import profile'"
                    />
                    <Button
                      v-if="node.data.kind === 'provider'"
                      icon="ti ti-plus"
                      text
                      rounded
                      class="p-1"
                      @click.stop="addAccountToProvider(node)"
                      v-tooltip.top="'Add account'"
                    />
                    <Button
                      icon="ti ti-pencil"
                      text
                      rounded
                      class="p-1"
                      @click.stop="node.data.kind === 'provider' ? editProvider(node.data) : editAccount(node.data)"
                      v-tooltip.top="'Edit'"
                    />
                    <Button
                      icon="ti ti-trash"
                      text
                      rounded
                      severity="danger"
                      class="p-1"
                      :disabled="node.data.kind === 'provider' && node.children && node.children.length > 0"
                      @click.stop="node.data.kind === 'provider' ? showDeleteProviderDialog(node.data) : showDeleteAccountDialog(node.data)"
                      v-tooltip.top="'Delete'"
                    />
                  </div>
                </template>
              </Column>
            </TreeTable>
          </template>
        </Card>

        <Card v-if="totalsByCurrency.length > 0" class="totals-card mt-3">
          <template #content>
            <div class="totals-row">
              <span class="totals-label">Total by currency</span>
              <div class="totals-values">
                <span v-for="t in totalsByCurrency" :key="t.currency" class="totals-item">
                  <span
                    class="font-semibold"
                    :style="t.total < 0 ? { color: 'var(--c-red-600)' } : {}"
                  >{{ formatAmount(t.total) }}</span>
                  <span
                    :class="['ml-1', t.total < 0 ? '' : 'text-500']"
                    :style="t.total < 0 ? { color: 'var(--c-red-600)' } : {}"
                  >{{ t.currency }}</span>
                </span>
              </div>
            </div>
          </template>
        </Card>
      </div>
    </template>
  </ResponsiveHorizontal>

  <AccountDialog
    v-if="selectedAccount"
    v-model:visible="accountDialogVisible"
    :is-edit="isEdit"
    :account-id="selectedAccount?.id"
    :provider-id="selectedAccount?.providerId"
    :name="selectedAccount?.name"
    :currency="selectedAccount?.currency"
    :type="selectedAccount?.type"
    :icon="selectedAccount?.icon"
    :notes="selectedAccount?.notes"
  />

  <CsvProfileDialog
    v-if="profileDialogAccount"
    v-model:visible="profileDialogVisible"
    :account-id="profileDialogAccount.id"
    :profile-id="profileDialogAccount.importProfileId || 0"
  />

  <DeleteDialog
    v-if="selectedItem && !deleteProviderDialogVisible"
    v-model:visible="deleteAccountDialogVisible"
    :name="selectedItem.name"
    title="Delete Account"
    message="Are you sure you want to delete this account?"
    @confirm="handleDeleteAccount"
  />

  <DeleteDialog
    v-if="selectedItem && !deleteAccountDialogVisible"
    v-model:visible="deleteProviderDialogVisible"
    :name="selectedItem.name"
    title="Delete Account Provider"
    message="Are you sure you want to delete this account provider?"
    @confirm="handleDeleteProvider"
  />

  <AccountProviderDialog
    v-model:visible="providerDialogVisible"
    :is-edit="isEditProvider"
    :provider-id="selectedProvider?.id"
    :name="selectedProvider?.name"
    :description="selectedProvider?.description"
    :icon="selectedProvider?.icon"
  />
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-left: auto;
}

.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.filter-input {
  min-width: 10rem;
  max-width: 15rem;
}

.filter-input-search {
  min-width: 14rem;
  max-width: 22rem;
  flex: 1;
}

.account-leaf {
  cursor: pointer;
}

/* Signal that an account name is clickable, matching the balances view. */
.account-leaf:hover .account-link {
  text-decoration: underline;
}

.totals-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.totals-label {
  font-weight: 600;
}

.totals-values {
  display: flex;
  flex-wrap: wrap;
  gap: 1.5rem;
  margin-left: auto;
}

.totals-item {
  white-space: nowrap;
}

.favorite-active {
  color: #fbbf24 !important;
}

/* Keep the actions cells from stretching; the column width is pinned to 1%
   via the inline style so the Balance column extends toward the right edge. */
.account-table :deep(.actions-cell) {
  white-space: nowrap;
}

/* Right-justify the Balance column. The cell content sits in a flex wrapper,
   so text-align has no effect — align the flex container instead. */
.account-table :deep(.bal-col .p-treetable-column-header-content),
.account-table :deep(td.bal-col .p-treetable-body-cell-content) {
  justify-content: flex-end;
}
</style>
