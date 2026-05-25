// ============================================
// LIMBO — API Client
// ============================================

const API = {
    /**
     * Fetch wrapper with error handling.
     */
    async _fetch(url, options = {}) {
        try {
            const resp = await fetch(url, {
                headers: { 'Content-Type': 'application/json', ...options.headers },
                ...options
            });
            if (!resp.ok) {
                const text = await resp.text();
                throw new Error(`HTTP ${resp.status}: ${text}`);
            }
            return resp.json();
        } catch (err) {
            console.error(`[API] ${options.method || 'GET'} ${url} failed:`, err);
            throw err;
        }
    },

    /**
     * GET /api/health
     */
    health() {
        return this._fetch('/api/health');
    },

    /**
     * GET /api/stats
     */
    stats() {
        return this._fetch('/api/stats');
    },

    /**
     * GET /api/requests with optional filters.
     * @param {Object} params - { status, type, search, sort }
     */
    requests(params = {}) {
        const query = new URLSearchParams();
        if (params.status) query.set('status', params.status);
        if (params.type) query.set('type', params.type);
        if (params.search) query.set('search', params.search);
        if (params.sort) query.set('sort', params.sort);
        const qs = query.toString();
        return this._fetch(`/api/requests${qs ? '?' + qs : ''}`);
    },

    /**
     * GET /api/triage/:seerrRequestId
     */
    getTriage(seerrRequestId) {
        return this._fetch(`/api/triage/${seerrRequestId}`);
    },

    /**
     * POST /api/triage
     * @param {Object} data - { seerrRequestId, status, notes, reason }
     */
    setTriage(data) {
        return this._fetch('/api/triage', {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    /**
     * POST /api/sync
     */
    sync() {
        return this._fetch('/api/sync', { method: 'POST' });
    },

    /**
     * POST /api/maintenance/clean-older
     */
    cleanOlder(olderThan, statuses) {
        return this._fetch('/api/maintenance/clean-older', {
            method: 'POST',
            body: JSON.stringify({ olderThan, statuses })
        });
    },

    /**
     * POST /api/maintenance/refresh-cache
     */
    refreshCache() {
        return this._fetch('/api/maintenance/refresh-cache', {
            method: 'POST'
        });
    }
};
