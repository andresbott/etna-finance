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
            <div class="cards-layout">
                <!-- Machine Card -->
                <div class="tracking-card">
                    <h2 class="card-title">
                        <i class="pi pi-chart-bar"></i>
                        Machine
                    </h2>
                    
                    <div class="panels-grid">
                        <div v-for="panel in panels" :key="panel.id" class="panel" @click="selectPanel(panel)">
                            <div class="panel-header">
                                <h3 class="panel-title">{{ panel.title }}</h3>
                            </div>
                            <div class="panel-content">
                                <div class="panel-date">
                                    <i class="pi pi-calendar"></i>
                                    {{ panel.date }}
                                </div>
                                <div class="stats">
                                    <span class="stat-value">{{ panel.weight }} KG</span>
                                    <span class="stat-separator">•</span>
                                    <span class="stat-value">{{ panel.reps }} reps</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Detail View Card -->
                <div class="detail-card" v-if="selectedPanel">
                    <div class="detail-header">
                        <h2 class="detail-title">{{ selectedPanel.title }}</h2>
                        <button class="edit-button">
                            <i class="pi pi-pencil"></i>
                        </button>
                    </div>
                    
                    <div class="detail-description">
                        <p>{{ selectedPanel.description }}</p>
                    </div>

                    <div class="entries-table-container">
                        <table class="entries-table">
                            <thead>
                                <tr>
                                    <th>Date</th>
                                    <th>Weight (KG)</th>
                                    <th>Reps</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="entry in selectedPanel.entries" :key="entry.id">
                                    <td>{{ entry.date }}</td>
                                    <td>{{ entry.weight }}</td>
                                    <td>{{ entry.reps }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref } from 'vue';

// Sample data for panels
const panels = ref([
    {
        id: 1,
        title: 'Bench Press',
        weight: 85,
        reps: 12,
        date: '2025-11-20',
        description: 'Upper body compound exercise focusing on chest, shoulders, and triceps development.',
        entries: [
            { id: 1, date: '2025-11-20', weight: 85, reps: 12 },
            { id: 2, date: '2025-11-18', weight: 82, reps: 12 },
            { id: 3, date: '2025-11-15', weight: 80, reps: 10 },
            { id: 4, date: '2025-11-13', weight: 80, reps: 12 },
            { id: 5, date: '2025-11-10', weight: 77, reps: 12 }
        ]
    },
    {
        id: 2,
        title: 'Squat',
        weight: 120,
        reps: 10,
        date: '2025-11-21',
        description: 'Fundamental lower body exercise targeting quadriceps, hamstrings, and glutes.',
        entries: [
            { id: 1, date: '2025-11-21', weight: 120, reps: 10 },
            { id: 2, date: '2025-11-19', weight: 115, reps: 10 },
            { id: 3, date: '2025-11-16', weight: 115, reps: 8 },
            { id: 4, date: '2025-11-14', weight: 110, reps: 10 },
            { id: 5, date: '2025-11-11', weight: 110, reps: 12 }
        ]
    },
    {
        id: 3,
        title: 'Deadlift',
        weight: 140,
        reps: 8,
        date: '2025-11-22',
        description: 'Complete posterior chain exercise engaging back, glutes, and hamstrings.',
        entries: [
            { id: 1, date: '2025-11-22', weight: 140, reps: 8 },
            { id: 2, date: '2025-11-20', weight: 135, reps: 8 },
            { id: 3, date: '2025-11-17', weight: 135, reps: 6 },
            { id: 4, date: '2025-11-15', weight: 130, reps: 8 },
            { id: 5, date: '2025-11-12', weight: 125, reps: 8 }
        ]
    },
    {
        id: 4,
        title: 'Overhead Press',
        weight: 55,
        reps: 15,
        date: '2025-11-23',
        description: 'Shoulder pressing movement for deltoid strength and upper body stability.',
        entries: [
            { id: 1, date: '2025-11-23', weight: 55, reps: 15 },
            { id: 2, date: '2025-11-21', weight: 52, reps: 15 },
            { id: 3, date: '2025-11-18', weight: 52, reps: 12 },
            { id: 4, date: '2025-11-16', weight: 50, reps: 15 },
            { id: 5, date: '2025-11-13', weight: 50, reps: 12 }
        ]
    },
    {
        id: 5,
        title: 'Barbell Row',
        weight: 75,
        reps: 12,
        date: '2025-11-24',
        description: 'Horizontal pulling exercise for back thickness and overall back development.',
        entries: [
            { id: 1, date: '2025-11-24', weight: 75, reps: 12 },
            { id: 2, date: '2025-11-22', weight: 72, reps: 12 },
            { id: 3, date: '2025-11-19', weight: 70, reps: 10 },
            { id: 4, date: '2025-11-17', weight: 70, reps: 12 },
            { id: 5, date: '2025-11-14', weight: 68, reps: 12 }
        ]
    }
]);

const selectedPanel = ref(panels.value[0]);

const selectPanel = (panel) => {
    selectedPanel.value = panel;
};
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

.cards-layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
}

/* Tracking Card Styles */
.tracking-card {
    padding: 2rem;
    background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
    border-radius: 12px;
    border: 1px solid #e9ecef;
}

.card-title {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--c-primary-800);
    margin: 0 0 1.5rem 0;
}

.card-title i {
    color: var(--c-primary-600);
}

.panels-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 1.5rem;
}

.panel {
    background: white;
    border-radius: 10px;
    padding: 1rem;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    transition: all 0.3s ease;
    border: 1px solid #e9ecef;
    cursor: pointer;
}

.panel:hover {
    transform: translateY(-4px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
    border-color: var(--c-primary-400);
}

.panel-header {
    margin-bottom: 0.75rem;
    padding-bottom: 0.5rem;
    border-bottom: 2px solid var(--c-primary-200);
}

.panel-title {
    font-size: 1rem;
    font-weight: 600;
    color: var(--c-primary-900);
    margin: 0;
}

.panel-content {
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
}

.panel-date {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.875rem;
    color: var(--c-primary-600);
    background: #f8f9fa;
    padding: 0.4rem 0.75rem;
    border-radius: 6px;
    white-space: nowrap;
}

.stats {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0;
}

.stat-value {
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--c-primary-700);
}

.stat-separator {
    font-size: 0.875rem;
    color: var(--c-primary-400);
}

.panel-date i {
    font-size: 0.875rem;
}

/* Detail Card Styles */
.detail-card {
    padding: 2rem;
    background: white;
    border-radius: 12px;
    border: 1px solid #e9ecef;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.detail-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
    padding-bottom: 1rem;
    border-bottom: 2px solid var(--c-primary-200);
}

.detail-title {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--c-primary-900);
    margin: 0;
}

.edit-button {
    background: var(--c-primary-100);
    border: 1px solid var(--c-primary-300);
    border-radius: 6px;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.edit-button:hover {
    background: var(--c-primary-200);
    border-color: var(--c-primary-400);
}

.edit-button i {
    font-size: 0.875rem;
    color: var(--c-primary-700);
}

.detail-description {
    margin-bottom: 1.5rem;
}

.detail-description p {
    color: var(--c-primary-700);
    line-height: 1.6;
    margin: 0;
    font-size: 0.95rem;
}

.entries-table-container {
    overflow-x: auto;
}

.entries-table {
    width: 100%;
    border-collapse: collapse;
}

.entries-table thead {
    background: var(--c-primary-50);
}

.entries-table th {
    text-align: left;
    padding: 0.75rem 1rem;
    font-weight: 600;
    color: var(--c-primary-900);
    font-size: 0.875rem;
    border-bottom: 2px solid var(--c-primary-200);
}

.entries-table td {
    padding: 0.75rem 1rem;
    color: var(--c-primary-700);
    font-size: 0.9rem;
    border-bottom: 1px solid #e9ecef;
}

.entries-table tbody tr:hover {
    background: var(--c-primary-50);
}

.entries-table tbody tr:last-child td {
    border-bottom: none;
}

/* Responsive Design */
@media (max-width: 1200px) {
    .cards-layout {
        grid-template-columns: 1fr;
    }
}

@media (max-width: 768px) {
    .panels-grid {
        grid-template-columns: 1fr;
    }
    
    .tracking-card, .detail-card {
        padding: 1.5rem;
    }
    
    .stock-market-view {
        padding: 1rem;
    }
    
    .content-container {
        padding: 1.5rem;
    }
}
</style>




