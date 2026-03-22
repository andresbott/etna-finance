<script setup lang="ts">
import Card from 'primevue/card'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>Categories</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>Overview</h3>
                        <p>
                            Categories let you classify income and expense transactions so you can see where your
                            money comes from and where it goes. They are optional — a transaction can exist without
                            a category — but assigning categories enables meaningful breakdowns in reports.
                        </p>
                        <p>
                            There are two separate category types:
                        </p>
                        <ul>
                            <li><strong>Income categories</strong> — classify money coming in (e.g. Salary, Freelance, Dividends).</li>
                            <li><strong>Expense categories</strong> — classify money going out (e.g. Groceries, Rent, Utilities).</li>
                        </ul>
                        <p>
                            Income and expense categories are kept strictly separate — an income transaction can only
                            use an income category, and an expense transaction can only use an expense category.
                        </p>
                    </section>

                    <section>
                        <h3>Hierarchy</h3>
                        <p>
                            Categories can be organized into a tree structure with parent and child categories.
                            For example, you might have a top-level <strong>Food</strong> expense category with
                            children like <strong>Groceries</strong>, <strong>Restaurants</strong>, and
                            <strong>Coffee</strong>.
                        </p>
                        <p>
                            A child category must be the same type as its parent — you cannot nest an expense
                            category under an income category. There is no limit to how deep the hierarchy can go.
                        </p>
                        <p>
                            Categories can be managed under
                            <router-link to="/settings/categories">Settings &gt; Categories</router-link>.
                        </p>
                    </section>

                    <section>
                        <h3>Category rules</h3>
                        <p>
                            Categories can be automatically assigned to transactions based on their description
                            using <router-link to="/docs/concepts/category-rules">category rules</router-link>.
                            This is especially useful when importing transactions from CSV files.
                        </p>
                    </section>
                </div>
            </template>
        </Card>
    </div>
</template>

<style src="../doc-content.css"></style>
