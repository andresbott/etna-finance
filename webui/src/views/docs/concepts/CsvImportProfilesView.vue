<script setup lang="ts">
import Card from 'primevue/card'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>CSV Import Profiles</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>Overview</h3>
                        <p>
                            A CSV import profile is a reusable configuration that tells Etna Finance how to read
                            a CSV file from a specific bank or institution. Each bank exports transactions in a
                            different format, so a profile captures the details needed to parse that format
                            correctly.
                        </p>
                        <p>
                            Profiles are linked to accounts — when you import a CSV into an account, the system
                            uses the account's linked profile to parse the file automatically.
                        </p>
                    </section>

                    <section>
                        <h3>Profile settings</h3>
                        <p>
                            A profile defines the following:
                        </p>
                        <ul>
                            <li><strong>Separator</strong> — The field delimiter used in the CSV file (e.g. comma, semicolon, or tab).</li>
                            <li><strong>Skip rows</strong> — The number of rows to skip before the header row, useful for files that start with a preamble.</li>
                            <li><strong>Date column</strong> — The name of the column containing transaction dates.</li>
                            <li><strong>Date format</strong> — How dates are formatted in the file (e.g. <em>2006-01-02</em> or <em>02/01/2006</em>).</li>
                            <li><strong>Description column</strong> — The name of the column containing transaction descriptions.</li>
                            <li><strong>Amount columns</strong> — Where to find the transaction amounts (see below).</li>
                        </ul>
                    </section>

                    <section>
                        <h3>Amount modes</h3>
                        <p>
                            CSV files represent transaction amounts in two common ways. The profile's
                            <strong>amount mode</strong> tells the parser which format to expect:
                        </p>
                        <ul>
                            <li><strong>Single column</strong> — One column contains signed amounts. Positive values are income, negative values are expenses.</li>
                            <li><strong>Split columns</strong> — Separate credit and debit columns. The credit column holds income amounts and the debit column holds expense amounts.</li>
                        </ul>
                    </section>

                    <section>
                        <h3>Linking a profile to an account</h3>
                        <p>
                            Each account can be linked to one import profile under
                            <router-link to="/settings/accounts">Settings &gt; Accounts</router-link>.
                            Once linked, importing a CSV into that account will use the profile to parse the file
                            without any additional configuration.
                        </p>
                    </section>

                    <section>
                        <h3>Import flow</h3>
                        <ol>
                            <li><strong>Upload</strong> — Select a CSV file for an account that has a linked profile.</li>
                            <li><strong>Preview</strong> — The system parses the file and shows a preview of detected transactions. <router-link to="/docs/concepts/category-rules">Category rules</router-link> are applied automatically during this step.</li>
                            <li><strong>Review</strong> — Check the parsed rows, adjust categories if needed, and exclude any rows you don't want to import. Duplicate transactions are flagged automatically.</li>
                            <li><strong>Confirm</strong> — Submit the selected rows to create the transactions in your account.</li>
                        </ol>
                    </section>

                    <section>
                        <h3>Managing profiles</h3>
                        <p>
                            Profiles can be created, edited, and tested under
                            <router-link to="/settings/csv-profiles">Settings &gt; CSV Profiles</router-link>.
                            You can upload a sample CSV file to preview how the profile parses it before using it
                            for a real import.
                        </p>
                    </section>
                </div>
            </template>
        </Card>
    </div>
</template>

<style src="../doc-content.css"></style>
