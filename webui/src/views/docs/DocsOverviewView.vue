<script setup lang="ts">
import Card from 'primevue/card'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>Overview</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>What is Etna Finance?</h3>
                        <p>
                            Etna Finance is a personal finance application for tracking your money across bank accounts,
                            savings, investments, and more. It gives you a clear picture of your income, expenses, and
                            net worth over time.
                        </p>
                    </section>

                    <section>
                        <h3>Key concepts</h3>
                        <ul>
                            <li>
                                <router-link to="/docs/concepts/accounts"><strong>Accounts</strong></router-link>
                                — Where your money lives. Each account belongs to a provider (e.g. a bank or brokerage)
                                and has a type like Checking, Savings, or Investment.
                            </li>
                            <li>
                                <router-link to="/docs/concepts/categories"><strong>Categories</strong></router-link>
                                — Labels for your income and expense transactions (e.g. Groceries, Salary, Rent)
                                that let you see where your money goes.
                            </li>
                            <li>
                                <router-link to="/docs/concepts/category-rules"><strong>Category Rules</strong></router-link>
                                — Pattern-based rules that automatically assign categories to transactions when
                                importing from CSV files.
                            </li>
                            <li>
                                <router-link to="/docs/concepts/csv-import-profiles"><strong>CSV Import Profiles</strong></router-link>
                                — Reusable configurations that tell Etna Finance how to read CSV exports from your
                                bank or institution.
                            </li>
                        </ul>
                    </section>

                    <section>
                        <h3>Getting started</h3>
                        <ol>
                            <li>
                                <strong>Create a provider</strong> — Add the bank or institution that holds your accounts
                                under <router-link to="/settings/accounts">Settings &gt; Accounts</router-link>.
                            </li>
                            <li>
                                <strong>Create accounts</strong> — Add your checking, savings, or investment accounts
                                under the provider you just created.
                            </li>
                            <li>
                                <strong>Set up categories</strong> — Define income and expense categories under
                                <router-link to="/settings/categories">Settings &gt; Categories</router-link>
                                to organize your transactions.
                            </li>
                            <li>
                                <strong>Import transactions</strong> — Create a
                                <router-link to="/docs/concepts/csv-import-profiles">CSV import profile</router-link>
                                for your bank's export format, link it to an account, and import your first CSV file.
                            </li>
                            <li>
                                <strong>Review reports</strong> — Check the
                                <router-link to="/reports/overview">dashboard</router-link> and
                                <router-link to="/reports/income-expense">income &amp; expense report</router-link>
                                to see your financial overview.
                            </li>
                        </ol>
                    </section>
                </div>
            </template>
        </Card>
    </div>
</template>

<style src="./doc-content.css"></style>
