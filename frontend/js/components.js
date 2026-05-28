// ============================================
// LIMBO — UI Components
// ============================================

const Components = {

    // Official SVG Icons
    icons: {
        seerr: `<img src="assets/icons/seerr.svg" width="14" height="14" alt="Seerr">`,
        radarr: `<img src="assets/icons/radarr.svg" width="14" height="14" alt="Radarr">`,
        sonarr: `<img src="assets/icons/sonarr.svg" width="14" height="14" alt="Sonarr">`,
        triage: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>`,
        dot: `<span class="status-dot"></span>`,
        clock: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-tertiary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>`,
        movie: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"></rect><line x1="7" y1="2" x2="7" y2="22"></line><line x1="17" y1="2" x2="17" y2="22"></line><line x1="2" y1="12" x2="22" y2="12"></line><line x1="2" y1="7" x2="7" y2="7"></line><line x1="2" y1="17" x2="7" y2="17"></line><line x1="17" y1="17" x2="22" y2="17"></line><line x1="17" y1="7" x2="22" y2="7"></line></svg>`,
        tv: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#EC4899" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="15" rx="2" ry="2"></rect><polyline points="17 2 12 7 7 2"></polyline></svg>`,
        chevron: `<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>`,
        arrowLeft: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline></svg>`,
        arrowRight: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline></svg>`
    },

    /**
     * Render the stats bar.
     * @param {Object} stats - { pending, unavailable, waitingRelease, total }
     * @param {string|null} activeStatus - currently filtered status
     */
    renderStats(stats, activeStatus = null) {
        const container = document.getElementById('stats-bar');
        if (App.state.currentView !== 'dashboard') {
            if (container) container.classList.add('hidden');
            return;
        }
        if (container) {
            container.classList.remove('hidden');
        }

        const items = [
            { key: 'PENDING', count: stats.pending },
            { key: 'WAITING_RELEASE', count: stats.waitingRelease },
            { key: 'UNAVAILABLE', count: stats.unavailable },
            { key: 'COMPLETED', count: stats.completed }
        ];

        items.forEach(item => {
            const card = document.getElementById(`stat-${item.key}`);
            if (!card) return;

            // Populate the number
            const countEl = card.querySelector('.stat-count');
            if (countEl) {
                countEl.textContent = item.count !== undefined ? item.count : 0;
            }

            // Update active state class toggled by JS
            const isActive = activeStatus === item.key;
            card.classList.toggle('active', isActive);
        });
    },

    /**
     * Render request cards.
     * @param {Array} requests - enriched request objects
     * @param {Object} config - { seerrPublicUrl }
     */
    renderRequests(requests, config = {}) {
        const grid = document.getElementById('requests-grid');
        const skeleton = document.getElementById('skeleton-container');
        const empty = document.getElementById('empty-state');

        // Hide skeleton
        skeleton.classList.add('hidden');

        if (App.state.currentView !== 'dashboard') {
            if (grid) grid.classList.add('hidden');
            if (empty) empty.classList.add('hidden');
            return;
        }

        if (!requests || requests.length === 0) {
            grid.innerHTML = '';
            empty.classList.remove('hidden');
            return;
        }

        empty.classList.add('hidden');

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

        grid.innerHTML = requests.map((req, idx) => {
            const releaseStr = req.releaseDate
                ? `${releaseIcon(req.releaseSource || 'Unknown')} ${req.releaseSource || ''}: ${formatDate(req.releaseDate)}`
                : '';

            const fulfilledStr = req.status === 'COMPLETED' && req.fulfilledAt
                ? `✅ Fulfilled: ${formatDate(req.fulfilledAt)}`
                : '';

            const posterSrc = req.posterUrl || '';

            // Build Seerr link
            const seerrLink = req.seerrUrl || '#';

            return `
                <div class="request-card group flex flex-col p-0 rounded-xl bg-white/60 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700 hover:shadow-lg transition-all duration-200 animate-fade-in-up opacity-0" style="animation-delay: ${idx * 0.05}s" data-request-id="${req.seerrRequestId}">
                    <div class="flex gap-4 p-4 pb-2">
                        <div class="w-[60px] min-w-[60px] h-[90px] rounded overflow-hidden bg-slate-100 dark:bg-slate-900 shrink-0">
                            ${posterSrc
                    ? `<img src="${posterSrc}" alt="${req.title}" loading="lazy" class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105">`
                    : `<div class="w-full h-full flex items-center justify-center text-3xl text-slate-400 dark:text-slate-500 bg-gradient-to-br from-slate-100 to-slate-200 dark:from-slate-800 dark:to-slate-900">${req.mediaType === 'tv' ? Components.icons.tv : Components.icons.movie}</div>`
                }
                        </div>
                        <div class="flex-1 flex flex-col gap-1.5 min-w-0">
                            <div class="flex items-start justify-between gap-3">
                                <span class="text-[1.05rem] font-semibold text-slate-900 dark:text-slate-100 leading-snug truncate" title="${req.title}">${req.title || 'Unknown Title'}</span>
                                <span class="text-sm shrink-0">${req.mediaType === 'tv' ? Components.icons.tv : Components.icons.movie}</span>
                            </div>
                            <div class="text-[0.8rem] text-slate-500 dark:text-slate-400 flex items-center gap-2 flex-wrap">
                                <span class="flex items-center gap-1">${Components.icons.clock} ${timeAgo(req.createdAt)}</span>
                                ${releaseStr ? `<span class="flex items-center gap-1">${releaseStr}</span>` : ''}
                                ${fulfilledStr ? `<span class="flex items-center gap-1">${fulfilledStr}</span>` : ''}
                            </div>
                            <div class="status-badge ${req.status} inline-flex items-center gap-1.5 py-1 px-2.5 rounded-full text-[0.7rem] font-bold uppercase tracking-wider w-fit ${badgeColors[req.status] || ''}">
                                <span class="w-1.5 h-1.5 rounded-full animate-pulse ${badgeDotColors[req.status] || ''}"></span>
                                ${statusLabel(req.status)}
                            </div>
                        </div>
                    </div>
                    <div class="p-4 pt-0.5 mt-auto">
                        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 w-full">
                            ${Components._renderTriageAction(req)}
                            ${req.serviceUrl
                    ? `<a class="action-btn w-full sm:flex-1 flex items-center justify-center gap-1.5 py-1.5 px-2.5 rounded border border-slate-200 dark:border-slate-700 bg-white/60 dark:bg-slate-800/40 text-slate-600 dark:text-slate-400 text-xs font-semibold cursor-pointer transition-all duration-150 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-350 dark:hover:border-slate-600" href="${req.serviceUrl}" title="Open in Service">
                                     ${req.mediaType === 'tv' ? Components.icons.sonarr : Components.icons.radarr}
                                     <span>${req.mediaType === 'tv' ? 'Sonarr' : 'Radarr'}</span>
                                   </a>`
                    : ''
                }
                            <a class="action-btn w-full sm:flex-1 flex items-center justify-center gap-1.5 py-1.5 px-2.5 rounded border border-slate-200 dark:border-slate-700 bg-white/60 dark:bg-slate-800/40 text-slate-600 dark:text-slate-400 text-xs font-semibold cursor-pointer transition-all duration-150 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-350 dark:hover:border-slate-600" href="${seerrLink}" title="Seerr">
                                ${Components.icons.seerr}
                                <span>Seerr</span>
                            </a>
                        </div>
                    </div>
                </div>
            `;
        }).join('');
    },

    /**
     * Render triage dropdown menu.
     */
    _renderTriageAction(req) {
        if (req.status === 'COMPLETED') return '';

        return `
            <div class="triage-dropdown group/dropdown relative w-full sm:flex-1 flex" id="triage-dropdown-${req.seerrRequestId}">
                <button class="action-btn flex-1 flex items-center justify-center gap-1.5 py-1.5 px-2.5 rounded border border-slate-200 dark:border-slate-700 bg-white/60 dark:bg-slate-800/40 text-slate-600 dark:text-slate-400 text-xs font-semibold cursor-pointer transition-all duration-150 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-350 dark:hover:border-slate-600" onclick="Components.toggleTriageMenu(${req.seerrRequestId}, event)" title="Change Status">
                    ${Components.icons.triage}
                    <span>Status</span>
                    ${Components.icons.chevron}
                </button>
                <div class="triage-menu absolute bottom-[calc(100%+6px)] left-0 min-w-[180px] p-1.5 rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-xl z-50 opacity-0 invisible translate-y-2 transition-all duration-150 group-[.open]/dropdown:opacity-100 group-[.open]/dropdown:visible group-[.open]/dropdown:translate-y-0">
                    <button class="triage-option flex items-center gap-2 w-full py-2 px-3 border-none rounded bg-transparent text-slate-700 dark:text-slate-200 text-xs font-medium cursor-pointer transition-colors duration-150 hover:bg-slate-100 dark:hover:bg-slate-750" data-status="PENDING" onclick="App.setTriage(${req.seerrRequestId}, 'PENDING')">
                        <span class="w-2 h-2 rounded-full bg-amber-500"></span>
                        <span>Pending</span>
                    </button>
                    <button class="triage-option flex items-center gap-2 w-full py-2 px-3 border-none rounded bg-transparent text-slate-700 dark:text-slate-200 text-xs font-medium cursor-pointer transition-colors duration-150 hover:bg-slate-100 dark:hover:bg-slate-750" data-status="WAITING_RELEASE" onclick="App.setTriage(${req.seerrRequestId}, 'WAITING_RELEASE')">
                        <span class="w-2 h-2 rounded-full bg-sky-500"></span>
                        <span>Waiting Release</span>
                    </button>
                    <button class="triage-option flex items-center gap-2 w-full py-2 px-3 border-none rounded bg-transparent text-slate-700 dark:text-slate-200 text-xs font-medium cursor-pointer transition-colors duration-150 hover:bg-slate-100 dark:hover:bg-slate-750" data-status="UNAVAILABLE" onclick="App.setTriage(${req.seerrRequestId}, 'UNAVAILABLE')">
                        <span class="w-2 h-2 rounded-full bg-rose-500"></span>
                        <span>Unavailable</span>
                    </button>
                </div>
            </div>
        `;
    },

    /**
     * Toggle the triage dropdown visibility.
     */
    toggleTriageMenu(seerrRequestId, event) {
        event.stopPropagation();
        const el = document.getElementById(`triage-dropdown-${seerrRequestId}`);

        // Close all other open dropdowns
        document.querySelectorAll('.triage-dropdown.open').forEach(other => {
            if (other !== el) {
                other.classList.remove('open');
            }
        });

        if (el) {
            el.classList.toggle('open');
        }
    },

    /**
     * Show loading skeletons.
     */
    showSkeletons() {
        document.getElementById('skeleton-container').classList.remove('hidden');
        document.getElementById('requests-grid').innerHTML = '';
        document.getElementById('empty-state').classList.add('hidden');
    }
};

// Close triage dropdowns when clicking outside
document.addEventListener('click', (e) => {
    if (!e.target.closest('.triage-dropdown')) {
        document.querySelectorAll('.triage-dropdown.open').forEach(el => {
            el.classList.remove('open');
        });
    }
});
