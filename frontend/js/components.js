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
        const items = [
            { key: 'PENDING', label: 'Pending', count: stats.pending, dotClass: 'pending' },
            { key: 'WAITING_RELEASE', label: 'Waiting', count: stats.waitingRelease, dotClass: 'waiting' },
            { key: 'UNAVAILABLE', label: 'Unavailable', count: stats.unavailable, dotClass: 'unavailable' },
            { key: 'COMPLETED', label: 'Done', count: stats.completed, dotClass: 'completed' }
        ];

        container.innerHTML = items.map(item => `
            <div class="stat-card ${activeStatus === item.key ? 'active' : ''}" 
                 data-status="${item.key}" 
                 id="stat-${item.key}"
                 onclick="App.filterByStatus('${item.key}')">
                <div class="stat-count">${item.count || 0}</div>
                <div class="stat-label">
                    <span class="stat-dot ${item.dotClass}"></span>
                    ${item.label}
                </div>
            </div>
        `).join('');
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

        grid.innerHTML = requests.map((req, idx) => {
            const typeEmoji = req.mediaType === 'tv' ? '📺' : '🎬';
            const releaseStr = req.releaseDate
                ? `${releaseIcon(req.releaseSource || 'Unknown')} ${req.releaseSource || ''}: ${formatDate(req.releaseDate)}`
                : '';
            
            const fulfilledStr = req.status === 'COMPLETED' && req.fulfilledAt
                ? `✅ Fulfilled: ${formatDate(req.fulfilledAt)}`
                : '';
            
            const posterSrc = req.posterUrl || '';

            // Build Seerr link
            const seerrLink = req.seerrUrl || '#';

            // Determine arr link based on type


            return `
                <div class="request-card" style="animation-delay: ${idx * 0.05}s" data-request-id="${req.seerrRequestId}">
                    <div class="card-body">
                        <div class="request-poster">
                            ${posterSrc 
                                ? `<img src="${posterSrc}" alt="${req.title}" loading="lazy" class="poster-img">`
                                : `<div class="poster-placeholder">${req.mediaType === 'tv' ? Components.icons.tv : Components.icons.movie}</div>`
                            }
                        </div>
                        <div class="request-content">
                            <div class="request-header">
                                <span class="request-title" title="${req.title}">${req.title || 'Unknown Title'}</span>
                                <span class="request-type-badge">${req.mediaType === 'tv' ? Components.icons.tv : Components.icons.movie}</span>
                            </div>
                            <div class="request-meta">
                                <span>${Components.icons.clock} ${timeAgo(req.createdAt)}</span>
                                ${releaseStr ? `<span class="request-release">${releaseStr}</span>` : ''}
                                ${fulfilledStr ? `<span class="request-fulfilled">${fulfilledStr}</span>` : ''}
                            </div>
                            <div class="status-badge ${req.status}">
                                <span class="status-dot"></span>
                                ${statusLabel(req.status)}
                            </div>
                        </div>
                    </div>
                    <div class="request-footer">
                        <div class="request-actions">
                            ${Components._renderTriageAction(req)}
                            ${req.serviceUrl 
                                ? `<a class="action-btn action-search" href="${req.serviceUrl}" title="Open in Service">
                                     ${req.mediaType === 'tv' ? Components.icons.sonarr : Components.icons.radarr}
                                     <span>${req.mediaType === 'tv' ? 'Sonarr' : 'Radarr'}</span>
                                   </a>`
                                : ''
                            }
                            <a class="action-btn" href="${seerrLink}" title="Seerr">
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
            <div class="triage-dropdown" id="triage-dropdown-${req.seerrRequestId}">
                <button class="action-btn" onclick="Components.toggleTriageMenu(${req.seerrRequestId}, event)" title="Change Status">
                    ${Components.icons.triage}
                    <span>Status</span>
                    ${Components.icons.chevron}
                </button>
                <div class="triage-menu">
                    <button class="triage-option" data-status="PENDING" onclick="App.setTriage(${req.seerrRequestId}, 'PENDING')">
                        <span class="option-dot"></span>
                        <span>Pending</span>
                    </button>
                    <button class="triage-option" data-status="WAITING_RELEASE" onclick="App.setTriage(${req.seerrRequestId}, 'WAITING_RELEASE')">
                        <span class="option-dot"></span>
                        <span>Waiting Release</span>
                    </button>
                    <button class="triage-option" data-status="UNAVAILABLE" onclick="App.setTriage(${req.seerrRequestId}, 'UNAVAILABLE')">
                        <span class="option-dot"></span>
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
