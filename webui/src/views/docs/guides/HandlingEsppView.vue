<script setup lang="ts">
import Card from 'primevue/card'
import Message from 'primevue/message'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>Handling ESPP</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>What is an ESPP?</h3>
                        <p>
                            An Employee Stock Purchase Plan (ESPP) is a benefit that lets you buy company stock
                            at a discount, typically 15% below market price. Money is deducted from your paycheck
                            over an offering period (usually 6 months), and at the end of the period the accumulated
                            funds are used to purchase shares at the discounted price.
                        </p>
                        <p>
                            A typical ESPP lifecycle looks like this:
                        </p>
                        <ol>
                            <li><strong>Enrollment</strong> — You enroll and choose a contribution percentage from your salary.</li>
                            <li><strong>Contribution period</strong> — Money is deducted from each paycheck over the offering period.</li>
                            <li><strong>Purchase</strong> — At the end of the period, shares are bought at a discount using your accumulated contributions.</li>
                        </ol>
                    </section>

                    <section>
                        <h3>Account setup</h3>
                        <p>
                            To track ESPP you need two accounts:
                        </p>
                        <ul>
                            <li>
                                <strong>Checking</strong> (or Cash) account — Holds the contributions that will be
                                used to purchase shares. This account must be in the same currency as the ESPP stock.
                            </li>
                            <li>
                                <strong>Investment</strong> account — Holds the purchased shares.
                            </li>
                        </ul>
                        <p>
                            Make sure the company stock is configured under
                            <router-link to="/settings/instruments">Investment Products</router-link>.
                        </p>
                    </section>

                    <section>
                        <h3>Step 1: Record the contributions</h3>
                        <p>
                            Each paycheck, a portion of your salary is set aside for the ESPP. Record these
                            as <strong>Income</strong> transactions in your ESPP checking account — for example,
                            if the money is deducted directly from your paycheck you can capture it as salary income.
                            This way the income is still categorized correctly in your reports (e.g. under your salary category),
                            while the balance builds up in the ESPP account ready for the purchase at the end of the
                            offering period.
                        </p>
                    </section>

                    <section>
                        <h3>Step 2: Record the purchase</h3>
                        <p>
                            At the end of the offering period, your accumulated contributions are used to buy
                            shares at the discounted price. Record a <strong>Buy stock</strong> transaction in your
                            <strong>Investment</strong> account, selecting the ESPP checking account as the cash source
                            and using the discounted purchase price as the price per share.
                        </p>
                        <p>
                            The discount is reflected in the lower cost basis — you don't need to record it
                            separately. The system will use this purchase price when calculating gains or losses later.
                        </p>
                        <Message severity="info" :closable="false" icon="ti ti-info-circle">
                            <strong>Note:</strong> Any benefit or loss is calculated at sell time, based on the difference between
                            the sale price and the discounted purchase price.
                        </Message>
                    </section>
                </div>
            </template>
        </Card>
    </div>
</template>

<style src="../doc-content.css"></style>
