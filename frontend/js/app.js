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

        // Parse view and filters from URL query parameters
        this.parseURLParams();

        // Initialize Theme Settings
        this.initTheme();

        // Bind event listeners
        this.bindEvents();

        // Register Service Worker for PWA installation
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js')
                .then(reg => {
                    console.log('✅ ServiceWorker registered on scope:', reg.scope);
                    this.initNotifications(reg);
                    this.trackSWUpdates(reg);
                })
                .catch(err => console.error('❌ ServiceWorker registration failed:', err));
        }

        // Sync filter UI elements with state
        const searchInput = document.getElementById('search-input');
        if (searchInput) searchInput.value = this.state.filters.search;

        const filterType = document.getElementById('filter-type');
        if (filterType) filterType.value = this.state.filters.type;

        const filterStatus = document.getElementById('filter-status');
        if (filterStatus) filterStatus.value = this.state.filters.status;

        const filterSort = document.getElementById('filter-sort');
        if (filterSort) filterSort.value = this.state.filters.sort;

        // Apply saved view without pushing/replacing URL again
        this.switchView(this.state.currentView, false);

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
            this.updateURLParams();
            this.loadRequests();
        }, 300));

        // Filter selects
        document.getElementById('filter-type').addEventListener('change', (e) => {
            this.state.filters.type = e.target.value;
            this.updateURLParams();
            this.loadRequests();
        });

        document.getElementById('filter-status').addEventListener('change', (e) => {
            this.state.filters.status = e.target.value;
            this.updateURLParams();
            // Update stat card active states
            document.querySelectorAll('.stat-card').forEach(card => {
                card.classList.toggle('active', card.dataset.status === e.target.value);
            });
            this.loadRequests();
        });

        document.getElementById('filter-sort').addEventListener('change', (e) => {
            this.state.filters.sort = e.target.value;
            this.updateURLParams();
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
                    this.updateURLParams();
                    search.blur();
                    this.loadRequests();
                }
            }
        });

        // Listen for history back/forward navigation
        window.addEventListener('popstate', () => {
            this.parseURLParams();

            // Sync filter UI elements with state
            const searchInput = document.getElementById('search-input');
            if (searchInput) searchInput.value = this.state.filters.search;

            const filterType = document.getElementById('filter-type');
            if (filterType) filterType.value = this.state.filters.type;

            const filterStatus = document.getElementById('filter-status');
            if (filterStatus) filterStatus.value = this.state.filters.status;

            const filterSort = document.getElementById('filter-sort');
            if (filterSort) filterSort.value = this.state.filters.sort;

            // Apply view without pushing/replacing URL params again
            this.switchView(this.state.currentView, false);

            // Reload requests
            this.loadRequests();
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
        const grid = document.getElementById('requests-grid');

        // Delay the loading opacity to avoid flickering/stuttering on fast connections (e.g. localhost)
        const loadingTimeout = setTimeout(() => {
            if (grid) {
                grid.classList.add('opacity-40', 'pointer-events-none');
            }
        }, 150);

        try {
            this.state.requests = await API.requests(this.state.filters);
            Components.renderRequests(this.state.requests);
        } catch (err) {
            console.error('Failed to load requests:', err);
            showToast('Failed to load requests', 'error');
        } finally {
            clearTimeout(loadingTimeout);
            if (grid) {
                grid.classList.remove('opacity-40', 'pointer-events-none');
            }
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

        this.updateURLParams();

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
        // 0. Prevent updating if the status is already the same
        const request = this.state.requests.find(r => r.seerrRequestId === seerrRequestId);
        if (request && request.status === status) {
            const dropdown = document.getElementById(`triage-dropdown-${seerrRequestId}`);
            if (dropdown) dropdown.classList.remove('open');
            return;
        }

        // 1. Immediately close the dropdown so UI feels responsive
        const dropdown = document.getElementById(`triage-dropdown-${seerrRequestId}`);
        if (dropdown) dropdown.classList.remove('open');

        // 2. Locate the request card in the DOM
        const card = document.querySelector(`.request-card[data-request-id="${seerrRequestId}"]`);

        // 3. If we are currently filtering by status, and the new status doesn't match the current status filter,
        // we can fade out the card and remove it from our local state.
        const currentFilterStatus = this.state.filters.status;
        const matchesFilter = !currentFilterStatus || currentFilterStatus === status;

        if (card && !matchesFilter) {
            // Apply a nice fade-out transition class
            card.style.transition = 'all 0.3s ease';
            card.style.opacity = '0';
            card.style.transform = 'translateY(10px)';
            setTimeout(() => {
                card.remove();
                // Check if grid is empty now to show empty state
                const grid = document.getElementById('requests-grid');
                if (grid && grid.children.length === 0) {
                    const empty = document.getElementById('empty-state');
                    if (empty) empty.classList.remove('hidden');
                    if (grid) grid.classList.add('hidden');
                }
            }, 300);
        } else if (card) {
            // If it still matches the filter, we can just update its status badge immediately in the UI!
            const badge = card.querySelector('.status-badge');
            if (badge) {
                // Update badge text and classes
                const badgeColors = {
                    'PENDING': 'bg-amber-500/10 text-amber-500 border border-amber-500/20',
                    'WAITING_RELEASE': 'bg-sky-500/10 text-sky-500 border border-sky-500/20',
                    'UNAVAILABLE': 'bg-rose-500/10 text-rose-500 border border-rose-500/20',
                    'COMPLETED': 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20'
                };
                const badgeDotColors = {
                    'PENDING': 'bg-amber-500',
                    'WAITING_RELEASE': 'bg-sky-500',
                    'UNAVAILABLE': 'bg-rose-500',
                    'COMPLETED': 'bg-emerald-500'
                };
                badge.className = `status-badge ${status} inline-flex items-center gap-1.5 py-1 px-2.5 rounded-full text-[0.7rem] font-bold uppercase tracking-wider w-fit ${badgeColors[status] || ''}`;
                badge.innerHTML = `
                    <span class="w-1.5 h-1.5 rounded-full animate-pulse ${badgeDotColors[status] || ''}"></span>
                    ${statusLabel(status)}
                `;
            }

            // Also hide the triage dropdown action button if it's completed (completed cards don't have triage buttons)
            if (status === 'COMPLETED') {
                const triageBtn = card.querySelector('.triage-dropdown');
                if (triageBtn) triageBtn.remove();
            }
        }

        try {
            await API.setTriage({ seerrRequestId, status });
            showToast(`Status set to ${statusLabel(status)}`, 'success');

            // Refresh background data to keep everything completely in sync (stats bar etc.)
            await this.refresh();
        } catch (err) {
            showToast('Failed to update triage', 'error');
            // Revert state by reloading in case of failure
            await this.refresh();
        }
    },

    /**
     * Switch view between dashboard and maintenance.
     */
    switchView(view, updateURL = true) {
        this.state.currentView = view;
        if (updateURL) {
            this.updateURLParams();
        }
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
     * Initialize PWA Push Notifications.
     */
    async initNotifications(reg) {
        if (!('PushManager' in window)) {
            console.log('PWA: Push messaging is not supported in this browser');
            return;
        }

        try {
            // Request notification permission if default
            if (Notification.permission === 'default') {
                const permission = await Notification.requestPermission();
                if (permission !== 'granted') {
                    console.log('PWA: Notification permission was denied.');
                    return;
                }
            } else if (Notification.permission !== 'granted') {
                return;
            }

            // Get VAPID public key from backend
            const { publicKey } = await API.getNotificationsConfig();
            if (!publicKey) {
                console.warn('PWA: VAPID public key not configured on server.');
                return;
            }

            // Check if already subscribed
            let subscription = await reg.pushManager.getSubscription();
            if (subscription) {
                // Verify/update subscription on server
                await API.subscribeNotifications(subscription);
                console.log('✅ PWA: Already subscribed to push notifications');
                return;
            }

            // Subscribe
            subscription = await reg.pushManager.subscribe({
                userVisibleOnly: true,
                applicationServerKey: urlB64ToUint8Array(publicKey)
            });

            await API.subscribeNotifications(subscription);
            console.log('✅ PWA: Subscribed to push notifications successfully');
        } catch (err) {
            console.error('PWA: Failed to subscribe to Web Push:', err);
        }
    },

    /**
     * Monitor Service Worker updates and prompt user to reload when a new version is ready.
     */
    trackSWUpdates(reg) {
        // If there's already a waiting worker (e.g. from a previous session), show the update prompt immediately
        if (reg.waiting) {
            this.showUpdatePrompt(reg.waiting);
            return;
        }

        // Listen for new worker installing
        reg.addEventListener('updatefound', () => {
            const newWorker = reg.installing;
            newWorker.addEventListener('statechange', () => {
                // Once installed, check if we had an existing active controller (i.e. it is an update, not the first load)
                if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                    this.showUpdatePrompt(newWorker);
                }
            });
        });

        // Reload the page once the new service worker takes control (after skipWaiting)
        let refreshing = false;
        navigator.serviceWorker.addEventListener('controllerchange', () => {
            if (refreshing) return;
            refreshing = true;
            window.location.reload();
        });
    },

    /**
     * Show a non-intrusive toast requesting the user to update the app.
     */
    showUpdatePrompt(worker) {
        const container = document.getElementById('toast-container');
        if (!container) return;

        const toast = document.createElement('div');
        // Styled beautifully following tailwind classes
        toast.className = 'flex items-center justify-between gap-4 py-3.5 px-4 rounded-xl border shadow-2xl text-xs font-semibold bg-slate-900/95 text-white border-indigo-500/30 max-w-sm w-full transition-all duration-300 transform translate-y-0 opacity-100';

        toast.innerHTML = `
            <div class="flex items-center gap-2">
                <span class="text-sm font-bold text-indigo-400">✨</span>
                <span>New version installed.</span>
            </div>
            <button class="bg-indigo-600 hover:bg-indigo-500 text-white px-3 py-1.5 rounded-lg text-[0.75rem] font-bold cursor-pointer transition-all duration-150 hover:-translate-y-px active:translate-y-0 shrink-0" id="btn-sw-update">
                Reload
            </button>
        `;

        container.appendChild(toast);

        const btn = toast.querySelector('#btn-sw-update');
        btn.addEventListener('click', () => {
            worker.postMessage({ action: 'skipWaiting' });
            toast.remove();
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
    },

    /**
     * Parse current view and filters from URL query parameters.
     */
    parseURLParams() {
        const params = new URLSearchParams(window.location.search);
        let urlChanged = false;

        // 1. View parameter
        const view = params.get('view');
        if (view !== null) {
            const validViews = ['dashboard', 'maintenance'];
            if (validViews.includes(view.toLowerCase())) {
                this.state.currentView = view.toLowerCase();
            } else {
                this.state.currentView = 'dashboard'; // default view
                params.delete('view');
                urlChanged = true;
            }
        } else {
            this.state.currentView = 'dashboard';
        }

        // 2. Status parameter
        const status = params.get('status');
        if (status === null) {
            this.state.filters.status = 'PENDING'; // Default status is Pending when missing
        } else {
            const validStatuses = ['PENDING', 'WAITING_RELEASE', 'UNAVAILABLE', 'COMPLETED'];
            if (validStatuses.includes(status.toUpperCase())) {
                this.state.filters.status = status.toUpperCase();
            } else {
                this.state.filters.status = ''; // Default to all if the status parameter is invalid, wrong, or 'all'
                params.delete('status');
                urlChanged = true;
            }
        }

        // 3. Type parameter
        const type = params.get('type');
        if (type !== null) {
            const validTypes = ['movie', 'tv'];
            if (validTypes.includes(type.toLowerCase())) {
                this.state.filters.type = type.toLowerCase();
            } else {
                this.state.filters.type = ''; // Default to all types
                params.delete('type');
                urlChanged = true;
            }
        } else {
            this.state.filters.type = '';
        }

        // 4. Search parameter
        const search = params.get('search');
        this.state.filters.search = search !== null ? search : '';

        // 5. Sort parameter
        const sort = params.get('sort');
        if (sort !== null) {
            const validSorts = ['newest', 'oldest', 'title', 'release', 'release_desc', 'release_asc'];
            if (validSorts.includes(sort.toLowerCase())) {
                const normSort = sort.toLowerCase();
                this.state.filters.sort = normSort === 'release_asc' ? 'release' : normSort;
            } else {
                this.state.filters.sort = 'newest'; // Default sort
                params.delete('sort');
                urlChanged = true;
            }
        } else {
            this.state.filters.sort = 'newest';
        }

        // Apply URL clean-up if any invalid parameter was removed
        if (urlChanged) {
            const queryString = params.toString();
            const newURL = window.location.pathname + (queryString ? `?${queryString}` : '');
            window.history.replaceState(null, '', newURL);
        }
    },

    /**
     * Sync the current view and active filters into the URL query parameters.
     */
    updateURLParams() {
        const params = new URLSearchParams();

        if (this.state.currentView && this.state.currentView !== 'dashboard') {
            params.set('view', this.state.currentView);
        }

        if (this.state.filters.status && this.state.filters.status !== 'PENDING') {
            params.set('status', this.state.filters.status.toLowerCase());
        }

        if (this.state.filters.type) {
            params.set('type', this.state.filters.type);
        }

        if (this.state.filters.search) {
            params.set('search', this.state.filters.search);
        }

        if (this.state.filters.sort !== 'newest') {
            params.set('sort', this.state.filters.sort);
        }

        const queryString = params.toString();
        const newURL = window.location.pathname + (queryString ? `?${queryString}` : '');
        window.history.replaceState(null, '', newURL);
    }
};

// --- Boot ---
document.addEventListener('DOMContentLoaded', () => App.init());
