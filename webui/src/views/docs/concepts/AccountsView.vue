<script setup lang="ts">
import Card from 'primevue/card'
import Message from 'primevue/message'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>Accounts</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>Overview</h3>
                        <p>
                            Accounts are the core building blocks of Etna Finance. Every financial transaction involves
                            moving money into or out of an account. Each account belongs to a
                            <strong>provider</strong> (the institution that holds it, such as a bank or brokerage)
                            and has a specific <strong>type</strong> that determines what kind of money or assets it tracks.
                        </p>
                        <p>
                            All accounts require a <strong>currency</strong>, which determines the denomination used
                            for balances and transactions within that account.
                        </p>
                    </section>

                    <section>
                        <h3>Providers</h3>
                        <p>
                            Every account must belong to a provider. A provider represents the institution that
                            holds the account — for example, your bank, brokerage, or employer stock plan administrator.
                            You can have multiple accounts under the same provider (e.g. a checking and a savings
                            account at the same bank).
                        </p>
                        <p>
                            Providers can be managed under
                            <router-link to="/settings/accounts">Settings &gt; Accounts</router-link>.
                        </p>
                    </section>

                    <section>
                        <h3>Account types</h3>
                        <p>
                            Etna Finance supports six account types, each designed for a different financial purpose:
                        </p>

                        <h4>Cash accounts: Cash, Checking, Savings, and Lent</h4>
                        <p>
                            These four account types all behave the same way — they hold money in a given currency
                            and support the same transactions: income, expenses, and transfers between accounts.
                            The distinction between them is purely organizational, helping you label the purpose of
                            each account:
                        </p>
                        <ul>
                            <li><strong>Cash</strong> — Physical money or cash equivalents, such as a wallet or petty cash fund.</li>
                            <li><strong>Checking</strong> — A bank account for day-to-day use, typically where your salary is deposited and everyday expenses are paid.</li>
                            <li><strong>Savings</strong> — A bank account where money is set aside, usually earning interest over time. Use this for emergency funds or goal-based saving.</li>
                            <li><strong>Lent</strong> — Money you have lent to others. This money is still yours, but it is not held in any of your bank or brokerage accounts. Use this to track outstanding loans and repayments.</li>
                        </ul>

                        <h4>Investment accounts: Investment and Unvested</h4>
                        <p>
                            These two account types track investment assets such as stocks, ETFs, or mutual funds.
                            Both work together with
                            <router-link to="/settings/instruments">investment instruments</router-link> to track
                            quantities, cost basis, and market value of your holdings.
                        </p>
                        <ul>
                            <li><strong>Investment</strong> — Vested assets you own and can sell at any time.</li>
                            <li><strong>Unvested</strong> — Assets promised to you but not yet accessible, such as unvested RSUs (Restricted Stock Units) from an employer grant. Once they vest, they are typically transferred to an Investment account.</li>
                        </ul>
                        <p>
                            See the <router-link to="/docs/guides/handling-rsus">Handling RSUs</router-link> guide
                            for a detailed walkthrough of the unvested-to-investment workflow.
                        </p>
                        <Message severity="info" :closable="false" icon="ti ti-info-circle">
                            Unvested assets are not included in financial reports. Only
                            once they are transferred to an Investment account do they count towards your portfolio
                            value and performance.
                        </Message>

                    </section>
                </div>
            </template>
        </Card>
    </div>
</template>

<style src="../doc-content.css"></style>
