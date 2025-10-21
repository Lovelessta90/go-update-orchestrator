async function loadDashboardStats() {
    try {
        // Fetch devices
        const devicesResp = await fetch('/api/devices');
        const devices = await devicesResp.json();
        // Update total devices
        const totalEl = document.getElementById('total-devices');
        if (totalEl) {
            totalEl.textContent = devices.length.toString();
        }
        // Count online devices
        const onlineCount = devices.filter(d => d.Status === 'online').length;
        const onlineEl = document.getElementById('online-devices');
        if (onlineEl) {
            onlineEl.textContent = onlineCount.toString();
        }
        // Fetch updates
        const updatesResp = await fetch('/api/updates');
        const updates = await updatesResp.json();
        // Count active updates
        const activeCount = updates.filter(u => u.Status === 'in_progress').length;
        const activeEl = document.getElementById('active-updates');
        if (activeEl) {
            activeEl.textContent = activeCount.toString();
        }
        // Display recent updates
        const recentContainer = document.getElementById('recent-updates');
        if (recentContainer) {
            if (updates.length === 0) {
                recentContainer.innerHTML = '<div class="loading">No updates scheduled</div>';
            }
            else {
                const recentHTML = updates.slice(0, 5).map(update => `
                    <div class="update-item">
                        <div>
                            <strong>${escapeHtml(update.UpdateID)}</strong>
                            <div style="font-size: 0.875rem; color: var(--text-secondary);">
                                ${update.Completed} / ${update.TotalDevices} devices completed
                            </div>
                        </div>
                        <span class="status-badge status-${update.Status}">${update.Status}</span>
                    </div>
                `).join('');
                recentContainer.innerHTML = recentHTML;
            }
        }
    }
    catch (err) {
        console.error('Failed to load dashboard stats:', err);
    }
}
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
// Load stats on page load
document.addEventListener('DOMContentLoaded', () => {
    loadDashboardStats();
    // Refresh every 5 seconds
    setInterval(loadDashboardStats, 5000);
});
export {};
