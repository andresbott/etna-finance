<script setup>
import Card from 'primevue/card'
import Password from 'primevue/password'
import InputGroup from 'primevue/inputgroup'
import InputText from 'primevue/inputtext'
import ToggleSwitch from 'primevue/toggleswitch'
import InputGroupAddon from 'primevue/inputgroupaddon'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { ref } from 'vue'
import { useUserStore } from '@/lib/user/userstore.js'
import { Form } from '@primevue/forms'
import router from '@/router/index.js'
import LoadingScreen from '@/components/common/loadingScreen.vue'
const user = useUserStore()
import { FormField } from '@primevue/forms'

//  TODO: if I access the login page while logged-in i should be redirected
// TODO: make the redirection path a property in the router
const omFormSubmit = (e) => {
    if (e.valid) {
        user.login(
            e.states.username.value,
            e.states.password.value,
            e.states.loggedin.value,
            function () {
                router.push('/')
            }
        )
    }
}

const initialValues = ref({
    username: '',
    password: '',
    loggedin: false
})

const resolver = ({ values }) => {
    const errors = {}

    if (!values.username) {
        errors.username = [{ message: 'Username is required.' }]
    }

    if (!values.password) {
        errors.password = [{ message: 'Password is required.' }]
    }

    return {
        errors
    }
}
</script>
<template>
    <Card>
        <template #title>Log in</template>
        <template #content>
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
                    <!--      USERNAME            -->
                    <InputGroup>
                        <InputGroupAddon>
                            <i class="pi pi-user"></i>
                        </InputGroupAddon>
                        <InputText
                            id="username"
                            name="username"
                            type="text"
                            autocomplete="username"
                            required
                            placeholder="Username"
                        />
                    </InputGroup>
                    <Message v-if="$form.username?.invalid" severity="error" size="small">{{
                        $form.username.error?.message
                    }}</Message>

                    <!--      PASSWORD            -->

                    <InputGroup>
                        <InputGroupAddon>
                            <i class="pi pi-lock"></i>
                        </InputGroupAddon>
                        <Password
                            id="password"
                            name="password"
                            :feedback="false"
                            placeholder="Password"
                            toggleMask
                            :inputProps="{ autocomplete: 'current-password', required: true }"
                        />
                    </InputGroup>
                    <Message v-if="$form.password?.invalid" severity="error" size="small">{{
                        $form.password.error?.message
                    }}</Message>

                    <!--      KEEP ME LOGGED IN            -->

                    <InputGroup>
                        <FormField v-slot="$field" name="loggedin" initialValue="">
                            <ToggleSwitch />
                            <label class="ml-4" for="loggedin">Keep me signed in</label>
                        </FormField>
                    </InputGroup>

                    <Message v-if="user.wrongPwErr" severity="error" closable
                        >Wrong username or password</Message
                    >

                    <Button id="login-submit"  type="submit" label="Log in" class="w-full" />
                </div>
            </Form>
        </template>
    </Card>
    <loadingScreen v-if="user.isLoading" />
</template>
