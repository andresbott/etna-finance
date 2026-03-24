<script setup lang="ts">
import Card from 'primevue/card'
import Message from 'primevue/message'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>Handling RSUs</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>What are RSUs?</h3>
                        <p>
                            Restricted Stock Units (RSUs) are a form of compensation where your employer promises to give you
                            shares of company stock on a future date, known as the <strong>vesting date</strong>. Until they vest,
                            you don't actually own the shares — they are "restricted" because you can't sell or transfer them.
                        </p>
                        <p>
                            A typical RSU lifecycle looks like this:
                        </p>
                        <ol>
                            <li><strong>Grant</strong> — Your employer awards you a number of RSUs (e.g. 100 shares vesting over 4 years).</li>
                            <li><strong>Vesting</strong> — On each vesting date, a portion of the RSUs convert into actual shares you own.</li>
                            <li><strong>Sell</strong> — You sell the vested shares on the market. Any profit or loss is realized at this point.</li>
                        </ol>
                    </section>

                    <section>
                        <h3>Account setup</h3>
                        <p>
                            To track RSUs you need two accounts under the same provider (e.g. your brokerage):
                        </p>
                        <ul>
                            <li>
                                <strong>Restricted stocks</strong> account — Holds the granted shares that haven't vested yet.
                                Create an account with type
                                <router-link to="/settings/accounts">Restricted stocks</router-link>.
                            </li>
                            <li>
                                <strong>Investment</strong> account — Holds the vested shares you actually own.
                                Create an account with type
                                <router-link to="/settings/accounts">Investment</router-link>.
                            </li>
                        </ul>
                        <p>
                            Both accounts must be linked to the same investment product (the company stock).
                            Make sure the stock is configured under
                            <router-link to="/settings/instruments">Investment Products</router-link>.
                        </p>
                    </section>

                    <section>
                        <h3>Step 1: Record the grant</h3>
                        <p>
                            When your employer grants you RSUs, record a <strong>Grant instrument</strong> transaction
                            in your <strong>Restricted stocks</strong> account. This captures the number of shares
                            granted and the fair market value at grant time.
                        </p>
                        <p>
                            No money changes hands at this point — the grant simply records that you have been promised
                            shares that will vest in the future.
                        </p>
                    </section>

                    <section>
                        <h3>Step 2: Record the vesting</h3>
                        <p>
                            When a batch of RSUs vests, the shares move from being "promised" to being "owned".
                            Record a <strong>Transfer instrument</strong> transaction to move the vested shares
                            from your <strong>Restricted stocks</strong> account to your <strong>Investment</strong> account.
                        </p>
                        <p>
                            This reflects that those shares are now yours — they sit in your investment account
                            and their value will track the current market price.
                        </p>
                        <Message severity="info" :closable="false" icon="ti ti-info-circle">
                            <strong>Note:</strong> Any benefit or loss is calculated at sell time, based on the difference between
                            the sale price and the fair market value at vesting.
                        </Message>
                    </section>


                </div>
            </template>
        </Card>
    </div>
</template>

<style src="../doc-content.css"></style>
