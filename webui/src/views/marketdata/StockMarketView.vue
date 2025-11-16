<template>
    <div class="stock-market-view">
        <div class="view-header">
            <h1 class="view-title">
                <i class="pi pi-chart-line"></i>
                Stock Market
            </h1>
            <p class="view-subtitle">Monitor stock market data and investment performance</p>
        </div>

        <div class="content-container">
            <div class="coming-soon">
                <i class="pi pi-chart-line icon-large"></i>
                <h2>Stock Market Data</h2>
                <p>Stock market tracking and analysis coming soon</p>
            </div>
        </div>
    </div>
</template>

<script setup>
// Stock market view logic will be implemented here
</script>

<style scoped>
.stock-market-view {
    padding: 2rem;
}

.view-header {
    margin-bottom: 2rem;
}

.view-title {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 2rem;
    font-weight: 700;
    color: var(--c-primary-900);
    margin: 0 0 0.5rem 0;
}

.view-title i {
    font-size: 1.75rem;
    color: var(--c-primary-600);
}

.view-subtitle {
    font-size: 1.125rem;
    color: var(--c-primary-600);
    margin: 0;
}

.content-container {
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 3rem;
}

.coming-soon {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 4rem 2rem;
    text-align: center;
}

.icon-large {
    font-size: 5rem;
    color: var(--c-primary-300);
    margin-bottom: 1.5rem;
}

.coming-soon h2 {
    font-size: 1.75rem;
    font-weight: 600;
    color: var(--c-primary-800);
    margin: 0 0 1rem 0;
}

.coming-soon p {
    font-size: 1.125rem;
    color: var(--c-primary-600);
    margin: 0;
}
</style>


