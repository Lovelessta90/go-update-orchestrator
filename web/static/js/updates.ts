import type { UpdateStatus } from './types.js';

async function loadUpdates(): Promise<void> {
    try {
        const resp = await fetch('/api/updates');
        const updates: UpdateStatus[] = await resp.json();

        const tbody = document.querySelector('#updates-table tbody');
        if (!tbody) return;

        if (updates.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">No updates scheduled</td></tr>';
            return;
        }

        tbody.innerHTML = updates.map(update => {
            const progress = update.TotalDevices > 0
                ? Math.round((update.Completed / update.TotalDevices) * 100)
                : 0;

            return `
                <tr>
                    <td><code>${escapeHtml(update.UpdateID)}</code></td>
                    <td><span class="status-badge status-${update.Status}">${update.Status}</span></td>
                    <td>${update.TotalDevices}</td>
                    <td>${update.Completed}</td>
                    <td>${update.Failed}</td>
                    <td>
                        <div class="progress-bar">
                            <div class="progress-fill" style="width: ${progress}%"></div>
                        </div>
                        <div class="progress-text">${progress}%</div>
                    </td>
                </tr>
            `;
        }).join('');
    } catch (err) {
        console.error('Failed to load updates:', err);
        const tbody = document.querySelector('#updates-table tbody');
        if (tbody) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">Error loading updates</td></tr>';
        }
    }
}

function escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Load updates on page load
document.addEventListener('DOMContentLoaded', () => {
    loadUpdates();
    // Refresh every 2 seconds for real-time updates
    setInterval(loadUpdates, 2000);
});
