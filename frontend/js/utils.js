// ============================================
// LIMBO — Utility Functions
// ============================================

/**
 * Format a relative time string from a date.
 * @param {string|Date} date
 * @returns {string}
 */
function timeAgo(date) {
    if (!date) return '—';
    const now = new Date();
    const then = new Date(date);
    const diffMs = now - then;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays === 1) return 'yesterday';
    if (diffDays < 30) return `${diffDays}d ago`;
    return then.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

/**
 * Format a date for display.
 * @param {string|Date} date
 * @returns {string}
 */
function formatDate(date) {
    if (!date) return '—';
    const d = new Date(date);
    return d.toLocaleDateString('en-US', { day: '2-digit', month: 'short', year: 'numeric' });
}

/**
 * Get the release icon emoji based on release source.
 * @param {string} source
 * @returns {string}
 */
function releaseIcon(source) {
    const icons = {
        'Digital': '📀',
        'Physical': '💿',
        'Theatrical': '🎬',
        'Air Date': '📡',
        'Unknown': '❓'
    };
    return icons[source] || '❓';
}

/**
 * Get the status display label.
 * @param {string} status
 * @returns {string}
 */
function statusLabel(status) {
    const labels = {
        'PENDING': 'Pending',
        'WAITING_RELEASE': 'Waiting Release',
        'NOT_AVAILABLE': 'Not Available',
        'COMPLETED': 'Completed'
    };
    return labels[status] || status;
}

/**
 * Debounce a function.
 * @param {Function} fn
 * @param {number} delay
 * @returns {Function}
 */
function debounce(fn, delay = 300) {
    let timer;
    return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => fn(...args), delay);
    };
}

/**
 * Show a toast notification.
 * @param {string} message
 * @param {'success'|'error'|'info'} type
 */
function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;

    const icons = { success: '✓', error: '✕', info: 'ℹ' };
    toast.innerHTML = `<span>${icons[type] || ''}</span><span>${message}</span>`;

    container.appendChild(toast);

    setTimeout(() => {
        toast.classList.add('removing');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}


