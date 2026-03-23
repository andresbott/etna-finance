<script setup>
import { ref, watch, computed, nextTick } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import Message from 'primevue/message'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'
import { useQueryClient, useMutation } from '@tanstack/vue-query'
import AccountSelector from '@/components/AccountSelector.vue'
import { useInstruments } from '@/composables/useInstruments'
import { createStockForfeit } from '@/lib/api/Entry'
import { useEntryMutations } from '@/composables/useEntryMutations'
import { useDateFormat } from '@/composables/useDateFormat'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId,
    toDateString
} from '@/composables/useEntryDialogForm'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'
import { useLots } from '@/composables/useLots'

const queryClient = useQueryClient()
const backendError = ref('')
const { instruments: instrumentsData } = useInstruments()
const { updateEntry } = useEntryMutations()

const createMutation = useMutation({
    mutationFn: createStockForfeit,
    onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['entries'] })
        queryClient.invalidateQueries({ queryKey: ['portfolio-positions'] })
        queryClient.invalidateQueries({ queryKey: ['portfolio-lots'] })
    }
})

const isSaving = computed(() => createMutation.isPending.value)
const { pickerDateFormat, dateValidation } = useDateFormat()

const instruments = computed(() => instrumentsData.value ?? [])

const instrumentOptions = computed(() =>
    instruments.value.map((s) => ({ label: `${s.symbol} – ${s.name}`, value: s.id }))
)

const instrumentById = computed(() =>
    Object.fromEntries((instruments.value ?? []).map((i) => [i.id, i]))
)

const props = defineProps({
    visible: { type: Boolean, default: false },
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    instrumentId: { type: Number, default: null },
    description: { type: String, default: '' },
    date: { type: Date, default: () => new Date() },
    accountId: { type: Number, default: null },
    notes: { type: String, default: '' },
    initialLotAllocations: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:visible'])
const formKey = ref(0)
const formErrors = ref({})

watch(() => props.visible, (v) => {
    if (!v) {
        backendError.value = ''
    } else {
        formErrors.value = {}
    }
})

const formValues = ref({
    instrumentId: props.instrumentId,
    description: props.description,
    notes: props.notes,
    date: getDateOnly(props.date),
    AccountId: getFormattedAccountId(props.accountId)
})

// Reset form when dialog opens
watch(
    () => [props.visible, props.instrumentId, props.description, props.date, props.accountId],
    async () => {
        if (props.visible) {
            await nextTick()
            formValues.value = {
                instrumentId: props.instrumentId,
                description: props.description,
                notes: props.notes,
                date: getDateOnly(props.date),
                AccountId: getFormattedAccountId(props.accountId)
            }
            formKey.value++
        }
    }
)

function getDefaultDescriptionForInstrument(id) {
    const symbol = instrumentById.value[id]?.symbol ?? ''
    return symbol ? `Forfeit ${symbol}` : ''
}

watch(
    () => formValues.value.instrumentId,
    (instrumentId) => {
        if (instrumentId != null && instrumentId >= 1) {
            const keepExisting = props.isEdit && instrumentId === props.instrumentId
            const currentDesc = (formValues.value.description ?? '').toString().trim()
            if (!keepExisting || !currentDesc) {
                formValues.value = { ...formValues.value, description: getDefaultDescriptionForInstrument(instrumentId) }
                formKey.value++
            }
        }
    }
)

// ---------------------------------------------------------------------------
// Lot selection
// ---------------------------------------------------------------------------
const lotAllocations = ref([])

const accountIdRef = computed(() => extractAccountId(formValues.value.AccountId))
const instrumentIdRef = computed(() => formValues.value.instrumentId ?? null)

const { lots } = useLots(accountIdRef, instrumentIdRef)

const visibleLots = computed(() =>
    lots.value.filter(l => getLotAvailable(l) > 0)
)

const totalAvailable = computed(() =>
    visibleLots.value.reduce((sum, l) => sum + getLotAvailable(l), 0)
)

// When editing, each lot's "available" quantity is its current DB quantity plus
// whatever was already allocated to this forfeit (which was subtracted when the forfeit was created).
const initialAllocMap = computed(() => {
    const map = {}
    for (const a of (props.initialLotAllocations ?? [])) {
        map[a.lotId] = (map[a.lotId] || 0) + a.quantity
    }
    return map
})

function getLotAvailable(lot) {
    return lot.quantity + (initialAllocMap.value[lot.id] || 0)
}

function getLotQty(lotId) {
    const entry = lotAllocations.value.find(a => a.lotId === lotId)
    return entry ? entry.quantity : 0
}

function setLotQty(lotId, qty) {
    const idx = lotAllocations.value.findIndex(a => a.lotId === lotId)
    if (idx >= 0) {
        if (!qty || qty <= 0) {
            lotAllocations.value.splice(idx, 1)
        } else {
            lotAllocations.value[idx].quantity = qty
        }
    } else if (qty > 0) {
        lotAllocations.value.push({ lotId, quantity: qty })
    }
}

const allocatedTotal = computed(() =>
    lotAllocations.value.reduce((sum, a) => sum + (a.quantity || 0), 0)
)

const lotSelectionError = computed(() => {
    if (allocatedTotal.value === 0) return 'Please allocate shares to at least one lot'
    return null
})

const hasLotError = computed(() => lotSelectionError.value !== null)

watch(
    () => [props.visible, formValues.value.instrumentId, formValues.value.AccountId],
    ([visible]) => {
        if (visible && props.isEdit && props.initialLotAllocations?.length > 0) {
            lotAllocations.value = props.initialLotAllocations.map(a => ({ lotId: a.lotId, quantity: a.quantity }))
        } else {
            lotAllocations.value = []
        }
    }
)

function applyAll() {
    const newAllocs = []
    for (const lot of visibleLots.value) {
        const avail = getLotAvailable(lot)
        if (avail > 0) {
            newAllocs.push({ lotId: lot.id, quantity: avail })
        }
    }
    lotAllocations.value = newAllocs
}

function clearAllocations() {
    lotAllocations.value = []
}

// ---------------------------------------------------------------------------
// Validation & Save
// ---------------------------------------------------------------------------
function validate() {
    const errors = {}
    const v = formValues.value
    if (!v.instrumentId || v.instrumentId < 1) errors.instrumentId = 'Instrument is required'
    if (!v.description?.toString().trim()) errors.description = 'Description is required'
    if (!v.date) errors.date = 'Date is required'
    const acctId = extractAccountId(v.AccountId)
    if (acctId == null) errors.AccountId = 'Account is required'
    if (hasLotError.value) errors.lots = lotSelectionError.value
    return errors
}

const doSave = async () => {
    formErrors.value = validate()
    if (Object.keys(formErrors.value).length > 0) return

    const v = formValues.value
    const description = (v.description ?? '').toString().trim()
    const notes = (v.notes ?? '').toString()
    const instrumentId = Number(v.instrumentId)
    const date = v.date
    const accountId = extractAccountId(v.AccountId)

    const activeAllocations = lotAllocations.value.filter(a => a.quantity > 0)
    if (activeAllocations.length === 0) return

    backendError.value = ''
    try {
        if (props.isEdit && props.entryId != null) {
            await updateEntry({
                id: String(props.entryId),
                description,
                notes,
                date: toDateString(date),
                instrumentId,
                accountId: String(accountId),
                lotAllocations: activeAllocations
            })
            queryClient.invalidateQueries({ queryKey: ['portfolio-positions'] })
            queryClient.invalidateQueries({ queryKey: ['portfolio-lots'] })
        } else {
            const payload = {
                type: 'stockforfeit',
                description,
                notes,
                date: toDateString(date),
                instrumentId,
                accountId,
                lotAllocations: activeAllocations
            }
            await createMutation.mutateAsync(payload)
        }
        emit('update:visible', false)
    } catch (err) {
        backendError.value = getApiErrorMessage(err)
        console.error(props.isEdit ? 'Failed to update stock forfeit:' : 'Failed to create stock forfeit:', err)
    }
}

const dialogTitle = computed(() =>
    props.isEdit ? 'Edit forfeit' : 'Forfeit shares'
)
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="dialogTitle"
        class="entry-dialog entry-dialog--wide"
    >
        <Form
            :key="formKey"
            v-slot="$form"
            :resolver="zodResolver(z.object({}))"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="false"
        >
            <Message v-if="backendError" severity="error" :closable="false" class="mb-2">{{ backendError }}</Message>
            <div v-focustrap class="flex flex-column gap-3">
                <div>
                    <label for="instrumentId" class="form-label">Investment instrument</label>
                    <Select
                        id="instrumentId"
                        v-model="formValues.instrumentId"
                        name="instrumentId"
                        :options="instrumentOptions"
                        optionLabel="label"
                        optionValue="value"
                        placeholder="Select instrument"
                    />
                    <Message v-if="formErrors.instrumentId" severity="error" size="small">
                        {{ formErrors.instrumentId }}
                    </Message>
                </div>
                <div>
                    <label for="description" class="form-label">Description</label>
                    <InputText
                        id="description"
                        v-model="formValues.description"
                        name="description"
                        placeholder="e.g. Forfeit AAPL"
                    />
                    <Message v-if="formErrors.description" severity="error" size="small">
                        {{ formErrors.description }}
                    </Message>
                </div>
                <div>
                    <label for="date" class="form-label">Date</label>
                    <DatePicker
                        id="date"
                        name="date"
                        v-model="formValues.date"
                        :showIcon="true"
                        iconDisplay="input"
                        :dateFormat="pickerDateFormat"
                        :showButtonBar="true"
                    />
                    <Message v-if="formErrors.date" severity="error" size="small">
                        {{ formErrors.date }}
                    </Message>
                </div>
                <div>
                    <label for="AccountId" class="form-label">Unvested account</label>
                    <AccountSelector
                        v-model="formValues.AccountId"
                        name="AccountId"
                        placeholder="Select unvested account"
                        :accountTypes="['unvested']"
                    />
                    <Message v-if="formErrors.AccountId" severity="error" size="small">
                        {{ formErrors.AccountId }}
                    </Message>
                </div>

                <!-- Lot selection -->
                <div class="flex flex-column gap-3">
                    <div v-if="visibleLots.length > 0" class="lot-selection">
                        <div class="flex align-items-center justify-content-between mb-2">
                            <span class="font-semibold text-sm">Lot selection</span>
                            <div class="flex gap-2">
                                <Button type="button" label="All" size="small" severity="secondary" @click="applyAll" />
                                <Button type="button" label="Clear" size="small" severity="secondary" @click="clearAllocations" />
                            </div>
                        </div>
                        <table class="lot-table w-full">
                            <thead>
                                <tr>
                                    <th class="text-left">Open date</th>
                                    <th class="text-right">Available</th>
                                    <th class="text-right">Grant FMV</th>
                                    <th class="text-right">Forfeit qty</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="lot in visibleLots" :key="lot.id">
                                    <td>{{ lot.openDate?.split('T')[0] ?? lot.openDate }}</td>
                                    <td class="text-right">{{ getLotAvailable(lot) }}</td>
                                    <td class="text-right">{{ lot.costPerShare?.toFixed(4) }}</td>
                                    <td class="text-right">
                                        <InputNumber
                                            :modelValue="getLotQty(lot.id)"
                                            @update:modelValue="setLotQty(lot.id, $event)"
                                            :min="0"
                                            :max="getLotAvailable(lot)"
                                            :minFractionDigits="0"
                                            :maxFractionDigits="4"
                                            size="small"
                                            class="lot-qty-input"
                                        />
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                        <div class="flex justify-content-between mt-2">
                            <span class="lot-summary">
                                Total to forfeit: {{ allocatedTotal.toFixed(4).replace(/\.?0+$/, '') }}
                            </span>
                        </div>
                        <Message v-if="formErrors.lots" severity="error" size="small" :closable="false">
                            {{ formErrors.lots }}
                        </Message>
                    </div>
                    <div v-else>
                        <Message severity="warn" :closable="false">
                            No open lots found for the selected instrument and account.
                        </Message>
                    </div>
                </div>

                <!-- Notes Field -->
                <div>
                    <label for="notes" class="form-label">Notes</label>
                    <Textarea
                        id="notes"
                        v-model="formValues.notes"
                        name="notes"
                        rows="3"
                        autoResize
                        fluid
                    />
                </div>

                <!-- Action buttons -->
                <div class="flex justify-content-end gap-3 pt-3">
                    <Button
                        type="button"
                        label="Save"
                        icon="ti ti-check"
                        :loading="isSaving"
                        @click="doSave"
                    />
                    <Button
                        type="button"
                        label="Cancel"
                        icon="ti ti-x"
                        severity="secondary"
                        @click="$emit('update:visible', false)"
                    />
                </div>
            </div>
        </Form>
    </Dialog>
</template>

<style scoped>
.lot-table {
    border-collapse: collapse;
    font-size: 0.875rem;
}

.lot-table th,
.lot-table td {
    padding: 0.3rem 0.5rem;
    border-bottom: 1px solid var(--p-surface-border, #e2e8f0);
    white-space: nowrap;
}

.lot-table th {
    font-weight: 600;
    color: var(--p-text-muted-color, #6b7280);
}

.lot-qty-input {
    width: 7rem;
}

.lot-summary {
    font-size: 0.875rem;
    color: var(--p-text-muted-color, #6b7280);
}
</style>
