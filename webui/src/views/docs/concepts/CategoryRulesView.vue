<script setup lang="ts">
import Card from 'primevue/card'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>Category Rules</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>Overview</h3>
                        <p>
                            Category rules automatically assign
                            <router-link to="/docs/concepts/categories">categories</router-link> to transactions
                            based on their description. This is especially useful when importing transactions from
                            CSV files — instead of categorizing each one by hand, rules do the work for you.
                        </p>
                    </section>

                    <section>
                        <h3>Rule groups</h3>
                        <p>
                            A rule group targets a specific category and contains one or more
                            <strong>patterns</strong>. If any pattern in the group matches a transaction's
                            description, that transaction is assigned the group's category.
                        </p>
                        <p>
                            For example, you could create a rule group targeting the <em>Groceries</em> category
                            with patterns like <em>whole foods</em>, <em>lidl</em>, and <em>aldi</em>. Any
                            imported transaction whose description contains one of those strings would be
                            automatically categorized as Groceries.
                        </p>
                    </section>

                    <section>
                        <h3>Patterns</h3>
                        <p>
                            Each pattern can be either a simple text match or a regular expression:
                        </p>
                        <ul>
                            <li><strong>Text match</strong> — A case-insensitive substring search. For example, the pattern <em>amazon</em> matches any description containing "amazon", "Amazon", or "AMAZON".</li>
                            <li><strong>Regex</strong> — A full regular expression for more advanced matching, such as matching multiple variations or enforcing word boundaries.</li>
                        </ul>
                        <p>
                            A rule group can contain as many patterns as needed. A transaction matches the group
                            if <strong>any</strong> of its patterns match.
                        </p>
                    </section>

                    <section>
                        <h3>Priority</h3>
                        <p>
                            Rule groups have a priority that controls the order they are evaluated. When multiple
                            groups could match a transaction, the group with the highest priority (lowest number)
                            wins.
                        </p>
                        <p>
                            This lets you create specific rules that take precedence over broader catch-all
                            patterns. For example, a rule for <em>Whole Foods</em> with priority 0 would win over
                            a general <em>*.com</em> rule with priority 10.
                        </p>
                    </section>

                    <section>
                        <h3>Re-applying rules</h3>
                        <p>
                            If you add or update rules after importing transactions, you can re-apply rules to
                            existing transactions. This re-runs all rule groups against your transaction history
                            and shows which categories would change, letting you review and apply the updates
                            selectively.
                        </p>
                        <p>
                            Rules can be managed under
                            <router-link to="/settings/category-rules">Settings &gt; Category Rules</router-link>.
                        </p>
                    </section>
                </div>
            </template>
        </Card>
    </div>
</template>

<style src="../doc-content.css"></style>
