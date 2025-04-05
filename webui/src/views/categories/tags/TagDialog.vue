<script setup>
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import InputGroupAddon from 'primevue/inputgroupaddon'
import InputGroup from 'primevue/inputgroup'
import Message from 'primevue/message'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useTagStore } from '@/stores/tags.js'

const tagStore = useTagStore()

const props = defineProps({
    parentId: {
        type: Number,
        default: 0
    },
    isEdit: {
        type: Boolean,
        default: false
    },
    itemId: {
        type: Number,
        default: 0
    },
    name: {
        type: String,
        default: ''
    }
})

const omFormSubmit = (e) => {
    if (e.valid) {
        if (props.isEdit) {
            tagStore.Update({
                id: e.values.itemId.value,
                name: e.values.name.value
                // parent: e.values.parentId.value
            })
        } else {
            tagStore.Add(e.values.name.value, e.values.parentId.value)
        }
        // bkmStore.Add(e.values.url.value, e.values.name.value, e.values.description.value)
        visible.value = false
    }
}

const initialValues = ref({
    itemId: props.itemId,
    parentId: props.parentId,
    name: props.name
})

const resolver = ref(
    zodResolver(
        z.object({
            name: z.string().min(1, { message: 'Tag name is required' })
        })
    )
)

const visible = ref(false)
</script>

<template>
    <div>
        <Button
            v-if="!props.isEdit"
            label=""
            severity="secondary"
            variant="text"
            icon="pi pi-plus"
            @click="visible = true"
        />
        <Button
            v-if="props.isEdit"
            label=""
            severity="secondary"
            variant="text"
            icon="pi pi-pencil"
            @click="visible = true"
        />
        <Dialog v-model:visible="visible" :draggable="false" modal header="Add New Tag">
            <Form
                v-slot="$form"
                :resolver
                :initialValues
                :validateOnValueUpdate="false"
                :validateOnBlur="true"
                class=""
                @submit="omFormSubmit"
            >
                <div v-focustrap class="flex flex-column items-center gap-4">
                    <InputGroup>
                        <InputText type="hidden" name="itemId" />
                        <InputText type="hidden" name="parentId" />
                    </InputGroup>

                    <InputGroup>
                        <InputGroupAddon>
                            <i class="pi pi-tag"></i>
                        </InputGroupAddon>
                        <InputText name="name" placeholder="Name" />
                    </InputGroup>
                    <Message v-if="$form.name?.invalid" severity="error" size="small">{{
                        $form.name.error?.message
                    }}</Message>

                    <!--                    <iconSelect />-->

                    <div class="flex justify-content-end gap-3">
                        <Button type="submit" label="Save" icon="pi pi-check"></Button>
                        <Button
                            type="button"
                            label="Cancel"
                            icon="pi pi-times"
                            severity="secondary"
                            @click="visible = false"
                        ></Button>
                    </div>
                </div>
            </Form>
        </Dialog>
    </div>
</template>

<style></style>
