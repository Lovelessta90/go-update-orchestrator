import type { Device } from './types.js';

async function loadDevices(): Promise<void> {
    try {
        const resp = await fetch('/api/devices');
        const devices: Device[] = await resp.json();

        const tbody = document.querySelector('#devices-table tbody');
        if (!tbody) return;

        if (devices.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">No devices found</td></tr>';
            return;
        }

        tbody.innerHTML = devices.map(device => `
            <tr>
                <td><code>${escapeHtml(device.ID)}</code></td>
                <td>${escapeHtml(device.Name || '-')}</td>
                <td>${escapeHtml(device.Address)}</td>
                <td><span class="status-badge status-${device.Status}">${device.Status}</span></td>
                <td>${escapeHtml(device.FirmwareVersion || '-')}</td>
                <td>${escapeHtml(device.Location || '-')}</td>
            </tr>
        `).join('');
    } catch (err) {
        console.error('Failed to load devices:', err);
        const tbody = document.querySelector('#devices-table tbody');
        if (tbody) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">Error loading devices</td></tr>';
        }
    }
}

function escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Load devices on page load
document.addEventListener('DOMContentLoaded', () => {
    loadDevices();
    // Refresh every 10 seconds
    setInterval(loadDevices, 10000);
});
