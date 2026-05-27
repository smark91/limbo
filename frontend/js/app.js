// ============================================
// LIMBO — Main Application
// ============================================

const App = {
    // State
    state: {
        stats: null,
        requests: [],
        filters: {
            status: 'PENDING',
            type: '',
            search: '',
            sort: 'newest'
        },
        lastSync: null,
        syncing: false,
        currentView: 'dashboard'
    },

    /**
     * Initialize the application.
     */
    async init() {
        console.log('🌀 Limbo initializing...');

        // Initialize Theme Settings
        this.initTheme();

        // Bind event listeners
        this.bindEvents();

        // Register Service Worker for PWA installation
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js')
                .then(reg => console.log('✅ ServiceWorker registered on scope:', reg.scope))
                .catch(err => console.error('❌ ServiceWorker registration failed:', err));
        }

        // Initialize default date value
        const dateInput = document.getElementById('clean-date');
        if (dateInput) {
            const d = new Date();
            d.setDate(d.getDate() - 30);
            dateInput.value = d.toISOString().split('T')[0];
        }

        // Show skeletons
        Components.showSkeletons();

        // Load initial data
        await Promise.all([
            this.loadStats(),
            this.loadRequests()
        ]);

        // Auto-refresh disabled by user request

        console.log('✅ Limbo ready');
    },

    /**
     * Bind DOM event listeners.
     */
    bindEvents() {
        // Theme button click: cycle theme
        const btnTheme = document.getElementById('btn-theme');
        if (btnTheme) {
            btnTheme.addEventListener('click', (e) => {
                e.stopPropagation();
                this.cycleTheme();
            });
        }

        // Sync button
        document.getElementById('btn-sync').addEventListener('click', () => this.triggerSync());

        // Search input (debounced)
        document.getElementById('search-input').addEventListener('input', debounce((e) => {
            this.state.filters.search = e.target.value;
            this.loadRequests();
        }, 300));

        // Filter selects
        document.getElementById('filter-type').addEventListener('change', (e) => {
            this.state.filters.type = e.target.value;
            this.loadRequests();
        });

        document.getElementById('filter-status').addEventListener('change', (e) => {
            this.state.filters.status = e.target.value;
            // Update stat card active states
            document.querySelectorAll('.stat-card').forEach(card => {
                card.classList.toggle('active', card.dataset.status === e.target.value);
            });
            this.loadRequests();
        });

        document.getElementById('filter-sort').addEventListener('change', (e) => {
            this.state.filters.sort = e.target.value;
            this.loadRequests();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + K to focus search
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                document.getElementById('search-input').focus();
            }
            // Escape to clear search
            if (e.key === 'Escape') {
                const search = document.getElementById('search-input');
                if (document.activeElement === search) {
                    search.value = '';
                    this.state.filters.search = '';
                    search.blur();
                    this.loadRequests();
                }
            }
        });
    },

    /**
     * Load stats from API.
     */
    async loadStats() {
        try {
            this.state.stats = await API.stats();
            this.state.lastSync = this.state.stats.lastScan;
            
            if (this.state.lastSync && this.state.lastSync !== "0001-01-01T00:00:00Z") {
                document.getElementById('last-sync').textContent = `Last sync: ${timeAgo(this.state.lastSync)}`;
            }

            if (this.state.stats.version) {
                document.getElementById('logo-version').textContent = this.state.stats.version;
            }

            Components.renderStats(this.state.stats, this.state.filters.status);
        } catch (err) {
            console.error('Failed to load stats:', err);
        }
    },

    /**
     * Load requests from API with current filters.
     */
    async loadRequests() {
        try {
            this.state.requests = await API.requests(this.state.filters);
            Components.renderRequests(this.state.requests);
        } catch (err) {
            console.error('Failed to load requests:', err);
            showToast('Failed to load requests', 'error');
        }
    },

    /**
     * Refresh both stats and requests.
     */
    async refresh() {
        await Promise.all([
            this.loadStats(),
            this.loadRequests()
        ]);
    },

    /**
     * Filter by clicking a stat card.
     */
    filterByStatus(status) {
        const select = document.getElementById('filter-status');

        // Toggle: if clicking the same status, clear the filter
        if (this.state.filters.status === status) {
            this.state.filters.status = '';
            select.value = '';
        } else {
            this.state.filters.status = status;
            select.value = status;
        }

        // Update stat card active states
        document.querySelectorAll('.stat-card').forEach(card => {
            card.classList.toggle('active', card.dataset.status === this.state.filters.status);
        });

        this.loadRequests();
    },

    /**
     * Trigger a background sync.
     */
    async triggerSync() {
        if (this.state.syncing) return;

        this.state.syncing = true;
        const btn = document.getElementById('btn-sync');
        btn.classList.add('syncing');

        try {
            await API.sync();
            showToast('Sync complete', 'success');
            await this.refresh();
        } catch (err) {
            showToast('Sync failed', 'error');
        } finally {
            this.state.syncing = false;
            btn.classList.remove('syncing');
        }
    },

	async setTriage(seerrRequestId, status) {
		try {
			await API.setTriage({ seerrRequestId, status });
			showToast(`Status set to ${statusLabel(status)}`, 'success');

			// Close dropdown
			const dropdown = document.getElementById(`triage-dropdown-${seerrRequestId}`);
			if (dropdown) dropdown.classList.remove('open');

			// Refresh
			await this.refresh();
		} catch (err) {
			showToast('Failed to update triage', 'error');
		}
	},

    /**
     * Switch view between dashboard and maintenance.
     */
    switchView(view) {
        this.state.currentView = view;
        const dashboardElements = [
            document.getElementById('stats-bar'),
            document.getElementById('filter-bar'),
            document.getElementById('requests-grid'),
            document.getElementById('empty-state'),
            document.getElementById('skeleton-container')
        ];
        const maintenanceView = document.getElementById('maintenance-view');
        const navDashboard = document.getElementById('nav-dashboard');
        const navMaintenance = document.getElementById('nav-maintenance');

        if (view === 'dashboard') {
            if (navDashboard) navDashboard.classList.add('active');
            if (navMaintenance) navMaintenance.classList.remove('active');
            if (maintenanceView) maintenanceView.classList.add('hidden');

            // Show dashboard elements
            const showGrid = this.state.requests && this.state.requests.length > 0;
            dashboardElements.forEach(el => {
                if (el) {
                    if (el.id === 'empty-state') {
                        el.classList.toggle('hidden', showGrid);
                    } else if (el.id === 'skeleton-container') {
                        el.classList.add('hidden');
                    } else if (el.id === 'requests-grid') {
                        el.classList.toggle('hidden', !showGrid);
                    } else {
                        el.classList.remove('hidden');
                    }
                }
            });
        } else {
            if (navDashboard) navDashboard.classList.remove('active');
            if (navMaintenance) navMaintenance.classList.add('active');
            if (maintenanceView) maintenanceView.classList.remove('hidden');

            // Hide dashboard elements
            dashboardElements.forEach(el => {
                if (el) el.classList.add('hidden');
            });

            // Set default date to 30 days ago
            const dateInput = document.getElementById('clean-date');
            if (dateInput && !dateInput.value) {
                const d = new Date();
                d.setDate(d.getDate() - 30);
                dateInput.value = d.toISOString().split('T')[0];
            }

            // Load cache metrics
            this.loadCacheInfo();
        }
    },

    /**
     * Perform mass cleanup of older requests.
     */
    async cleanOlderRequests() {
        const dateInput = document.getElementById('clean-date');
        if (!dateInput || !dateInput.value) {
            showToast('Please select a date', 'error');
            return;
        }

        const dateVal = dateInput.value;
        const statuses = [];
        if (document.getElementById('clean-status-pending').checked) statuses.push('PENDING');
        if (document.getElementById('clean-status-waiting').checked) statuses.push('WAITING_RELEASE');
        if (document.getElementById('clean-status-unavailable').checked) statuses.push('UNAVAILABLE');

        if (statuses.length === 0) {
            showToast('Please select at least one status to remove', 'error');
            return;
        }

        const confirmMsg = `Are you sure you want to permanently delete unfulfilled requests older than ${formatDate(dateVal)} for status(es): ${statuses.map(statusLabel).join(', ')}?\n\nThis will remove them from Seerr and the local database. This action cannot be undone.`;
        if (!confirm(confirmMsg)) {
            return;
        }

        const btn = document.getElementById('btn-clean-requests');
        const origText = btn.textContent;
        btn.disabled = true;
        btn.textContent = 'Cleaning...';

        try {
            // Append timezone info to make it a full RFC3339 string
            const olderThan = new Date(dateVal + 'T00:00:00Z').toISOString();
            const res = await API.cleanOlder(olderThan, statuses);
            showToast(`Successfully deleted ${res.deletedCount} request(s)`, 'success');
            
            // Force immediate sync to reconcile state with Seerr
            showToast('Forcing sync...', 'info');
            await API.sync();
            
            await this.refresh();
        } catch (err) {
            showToast(err.message || 'Failed to clean requests', 'error');
        } finally {
            btn.disabled = false;
            btn.textContent = origText;
        }
    },

    /**
     * Force refresh poster and release date cache.
     */
    async refreshPosterCache() {
        const confirmMsg = 'Are you sure you want to force refresh poster paths and release dates for all active requests?\n\nThis will reload metadata from Seerr/TMDB. It may take some time.';
        if (!confirm(confirmMsg)) {
            return;
        }

        const btn = document.getElementById('btn-refresh-cache');
        const origText = btn.textContent;
        btn.disabled = true;
        btn.textContent = 'Refreshing...';

        try {
            const res = await API.refreshCache();
            showToast(`Successfully refreshed cache for ${res.refreshedCount} request(s)`, 'success');
            await this.refresh();
            await this.loadCacheInfo();
        } catch (err) {
            showToast(err.message || 'Failed to refresh cache', 'error');
        } finally {
            btn.disabled = false;
            btn.textContent = origText;
        }
    },

    /**
     * Fetch cache stats and update the maintenance page.
     */
    async loadCacheInfo() {
        try {
            const info = await API.getCacheInfo();
            const activeEl = document.getElementById('cache-active-count');
            const totalEl = document.getElementById('cache-total-count');
            const sizeEl = document.getElementById('cache-size');

            if (activeEl) activeEl.textContent = info.activeCount;
            if (totalEl) totalEl.textContent = info.totalCount;
            if (sizeEl) sizeEl.textContent = formatBytes(info.size);
        } catch (err) {
            console.error('Failed to load cache info:', err);
        }
    },

    /**
     * Trigger a test notification.
     * @param {'system'|'movie'|'tv'} type
     */
    async testNotification(type) {
        const btn = document.getElementById(`btn-test-notification-${type}`);
        if (!btn) return;
        const origText = btn.textContent;
        btn.disabled = true;
        btn.textContent = 'Testing...';

        try {
            const res = await API.testNotification(type);
            showToast(res.message || 'Test notification sent successfully', 'success');
        } catch (err) {
            showToast(err.message || 'Failed to send test notification', 'error');
        } finally {
            btn.disabled = false;
            btn.textContent = origText;
        }
    },

    /**
     * Initialize theme settings and register listeners.
     */
    initTheme() {
        const theme = localStorage.getItem('limbo-theme') || 'system';
        this.applyTheme(theme);

        // Watch for system theme changes if set to system
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
            if (localStorage.getItem('limbo-theme') === 'system') {
                this.applyTheme('system');
            }
        });
    },

    /**
     * Set the application theme (light, dark, or system).
     */
    setTheme(theme, event) {
        if (event) event.stopPropagation();
        localStorage.setItem('limbo-theme', theme);
        this.applyTheme(theme);
        showToast(`Theme set to ${theme}`, 'success');
    },

    /**
     * Cycle to the next theme (system -> light -> dark -> system).
     */
    cycleTheme() {
        const current = localStorage.getItem('limbo-theme') || 'system';
        let next = 'system';
        if (current === 'system') {
            next = 'light';
        } else if (current === 'light') {
            next = 'dark';
        } else {
            next = 'system';
        }
        this.setTheme(next);
    },

    /**
     * Apply theme class helper.
     */
    applyTheme(theme) {
        const isDark = theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
        
        if (isDark) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }

        // Update active icon class in button
        const icons = {
            light: document.getElementById('theme-icon-light'),
            dark: document.getElementById('theme-icon-dark'),
            system: document.getElementById('theme-icon-system')
        };

        Object.keys(icons).forEach(k => {
            if (icons[k]) {
                icons[k].classList.toggle('active', k === theme);
                icons[k].classList.toggle('hidden', k !== theme);
            }
        });
    }
};

// --- Boot ---
document.addEventListener('DOMContentLoaded', () => App.init());
