<script setup>
import { ref } from 'vue'
import Button from 'primevue/button'
import Menu from 'primevue/menu'
import EntryDialog from '@/views/entries/EntryDialog.vue'
import TransferDialog from './TransferDialog.vue'
import StockDialog from './StockDialog.vue'


/* Internal state for menu and dialog visibility */
const menu = ref(null)
const dialogs = ref({
  expense: false,
  income: false,
  transfer: false,
  stock: false
})

/* Toggle function to control menu visibility */
const toggleMenu = (event) => {
  menu.value.toggle(event)
}

/* Open the respective dialog when a menu item is clicked */
const openDialog = (dialogType) => {
  dialogs.value[dialogType] = true
}

/* Setup menu items with their corresponding dialog actions */
const menuItems = ref([
  {
    label: 'Add Expense',
    icon: 'pi pi-minus',
    command: () => openDialog('expense')
  },
  {
    label: 'Add Income',
    icon: 'pi pi-plus',
    command: () => openDialog('income')
  },
  {
    label: 'Add Transfer',
    icon: 'pi pi-arrow-right-arrow-left',
    command: () => openDialog('transfer')
  },

  {
    label: 'Stock Operation',
    icon: 'pi pi-chart-line',
    command: () => openDialog('stock')
  },
  { separator: true },
  {
    label: 'CSV import',
    icon: 'pi pi-bolt',
    command: () => openDialog('transfer')
  }
])
</script>

<template>
    <div class="add-entry-menu">
        <!-- Add Entry Menu Button -->
        <Button
            label=""
            icon="pi pi-plus"
            @click="toggleMenu"
            aria-haspopup="true"
            aria-controls="overlay_menu"
        />
        <Menu ref="menu" :model="menuItems" :popup="true" id="overlay_menu" />

        <!-- Expense Dialog -->
        <EntryDialog
            v-model:visible="dialogs.expense"
            :isEdit="false"
            entryType="expense"
            @update:visible="dialogs.expense = $event"
        />

        <!-- Income Dialog -->
        <EntryDialog
            v-model:visible="dialogs.income"
            :isEdit="false"
            entryType="income"
            @update:visible="dialogs.income = $event"
        />

        <!-- Transfer Dialog -->
        <TransferDialog
            v-model:visible="dialogs.transfer"
            :isEdit="false"
            :entryId="selectedEntry?.id"
            :description="selectedEntry?.description"
            :amount="selectedEntry?.amount"
            :date="selectedEntry?.date"
            :targetAccountId="selectedEntry?.targetAccountId"
            :originAccountId="selectedEntry?.originAccountId"
            :categoryId="selectedEntry?.categoryId"
            @update:visible="dialogs.transfer = $event"
        />

        <!-- Stock Dialog -->
        <StockDialog
            v-model:visible="dialogs.stock"
            :isEdit="false"
            :entryId="selectedEntry?.id"
            :description="selectedEntry?.description"
            :amount="selectedEntry?.amount"
            :stockAmount="selectedEntry?.stockAmount"
            :date="selectedEntry?.date"
            :type="selectedEntry?.type"
            :targetAccountId="selectedEntry?.targetAccountId"
            :originAccountId="selectedEntry?.originAccountId"
            :categoryId="selectedEntry?.categoryId"
            @update:visible="dialogs.stock = $event"
        />
    </div>
</template>

<style scoped>
.add-entry-menu {
    display: flex;
    justify-content: center;
    padding: 1rem;
}
</style>
