// ============================================
// LIMBO — UI Components
// ============================================

const Components = {

    statusDotColors: {
        'PENDING': 'bg-amber-500',
        'WAITING_RELEASE': 'bg-sky-500',
        'UNAVAILABLE': 'bg-rose-500',
        'COMPLETED': 'bg-emerald-500'
    },

    // Official SVG Icons
    icons: {
        seerr: `<svg class="fill-current" width="14" height="14" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg"><path d="M256 0C114.6 0 0 114.6 0 256s114.6 256 256 256 256-114.6 256-256S397.4 0 256 0M64 256c0 11.8-9.6 21.3-21.3 21.3-11.8 0-21.3-9.6-21.3-21.3-.1-129.6 105-234.7 234.6-234.7 11.8 0 21.3 9.6 21.3 21.3 0 11.8-9.5 21.4-21.3 21.4-106 0-192 86-192 192m224.1 191.9c-88.4 0-160-71.6-160-160 0-1.3 0-2.7.1-4-.1-2.2-.2-4.4-.2-6.6 0-15.3 2.3-30.1 6.6-44 11.7 25.9 37.8 44 68.1 44 41.2 0 74.7-33.4 74.7-74.7 0-30.3-18-56.4-44-68.1 13.9-4.3 28.7-6.6 44-6.6 2.1 0 4.3.1 6.4.2-.4 0-.7 0-1.1-.1 1.8-.1 3.6-.1 5.4-.1 88.4 0 160 71.6 160 160s-71.6 160-160 160" style="fill-rule:evenodd;clip-rule:evenodd"/></svg>`,
        radarr: `<svg class="fill-current" width="14" height="14" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg"><path d="m80.3 80.8 3.9 372.4c-31.4 3.9-54.9-11.8-54.9-43.1l-3.9-309.7c0-98 90.2-121.5 145.1-82.3l278.3 160.7c39.2 27.4 47 78.4 27.4 113.7-3.9-27.4-15.7-43.1-39.2-58.8L123.4 57.2C99.9 41.6 80.3 45.5 80.3 80.8m-23.5 392c23.5 7.8 47 3.9 66.6-7.8l321.5-188.2c19.6 27.4 15.7 54.9-7.8 70.6L166.5 504.2c-39.2 19.6-90.1 0-109.7-31.4M150.9 363 343 253.3 154.8 147.4z"/></svg>`,
        sonarr: `<svg class="fill-current" width="14" height="14" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg"><path d="M144.2 103.3c30.7 30.7 70 38.6 112.4 38.6 43.6 0 82.8-8.4 114.7-40.4 14.7-14.7 44.4-44.3 45.5-45.4C371.1 18.7 317.6 0 256.2 0c-60.8 0-114 18.5-159.8 55.5zM373 258.4c0 42.3 6.7 81.2 38.2 112.7 22.9 22.9 44.7 44.5 45 44.8 37-45.8 55.6-99.1 55.6-159.9 0-58.9-17.4-110.8-52.3-155.9L406.6 153c-30.9 31-33.6 57.9-33.6 105.4m-271.1 113c32.7-32.7 38-70.6 38-113.1 0-41.3-6.8-79.9-36.8-110-20.1-20-47.6-47.2-49.7-49.4-31.8 40.3-49.2 86.4-52.3 138.3-.3.6-.5 1.1-.5 1.7C.3 244.3.2 250 .2 256c0 5.7.2 11.3.4 17 .5 10.2 1.7 20.3 3.4 30.2 7.3 42.1 24.8 80 52.7 113.6.1-.2 23.2-23.4 45.2-45.4m269.6 46c-36.8-36.8-66.1-40.4-114.7-40.4-46.7 0-78.4 4.3-112.6 38.5-20.2 20.3-43.4 43.6-43.8 43.9 2.2 1.7 4.4 3.3 6.6 4.9 43 31.8 92.7 47.7 149.3 47.7q84.75 0 149.4-47.7c2.5-1.7 4.9-3.5 7.3-5.4zM186 269.1c-.5-2.8-.8-5.5-.9-8.4-.1-1.6-.1-3.1-.1-4.7 0-1.7 0-3.2.1-4.7 0-.2 0-.3.1-.5 1-17.4 7.9-32.4 20.5-45.1 13.9-13.8 30.6-20.7 50.2-20.7s36.3 6.9 50.2 20.7c13.8 14 20.7 30.8 20.7 50.3s-6.9 36.2-20.7 50.2c-.5.5-1 1.1-1.5 1.5q-3.45 3.3-7.2 6-18 13.2-41.4 13.2c-23.4 0-29.4-4.4-41.3-13.2-3.1-2.2-6.1-4.7-8.9-7.6-10.8-10.6-17.3-22.9-19.8-37" style="fill-rule:evenodd;clip-rule:evenodd"/><path d="m375.2 143.5-1.6-1.6v-.1L440 77.2l-1.4-1.4-66.4 64.6.7.7-.7-.7h-.1l-1.9-1.9-40 40.6 5 5zm-238.3 2.1 40.6 40.5 5-5-40.6-40.5-1.7 1.7-66.4-66.1-1.4 1.4 66.4 66.1zm234.9 223.9-42.6-42.4-5 5 42.6 42.4 1.8-1.8 65.6 67.8 1.4-1.4-65.5-67.9zm-233.3 2.1 1.9 1.9-64.3 64.4 1.4 1.4 64.4-64.5 1.6 1.6 39.5-41.1-5-4.8z"/></svg>`,
        triage: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"></path></svg>`,
        clock: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>`,
        movie: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#3B82F6" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"></rect><line x1="7" y1="2" x2="7" y2="22"></line><line x1="17" y1="2" x2="17" y2="22"></line><line x1="2" y1="12" x2="22" y2="12"></line><line x1="2" y1="7" x2="7" y2="7"></line><line x1="2" y1="17" x2="7" y2="17"></line><line x1="17" y1="17" x2="22" y2="17"></line><line x1="17" y1="7" x2="22" y2="7"></line></svg>`,
        tv: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#EC4899" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="15" rx="2" ry="2"></rect><polyline points="17 2 12 7 7 2"></polyline></svg>`,
        chevron: `<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>`,
        arrowLeft: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline></svg>`,
        arrowRight: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline></svg>`,
        ban: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="4.93" y1="4.93" x2="19.07" y2="19.07"></line></svg>`,
        digital: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polygon points="10 8 16 12 10 16 10 8"></polygon></svg>`,
        physical: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><circle cx="12" cy="12" r="3"></circle></svg>`,
        theatrical: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"></path><path d="M13 5v14"></path></svg>`,
        airdate: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4.9 19.1C1 15.2 1 8.8 4.9 4.9"></path><path d="M7.8 16.2c-2.3-2.3-2.3-6.1 0-8.5"></path><circle cx="12" cy="12" r="2"></circle><path d="M16.2 7.8c2.3 2.3 2.3 6.1 0 8.5"></path><path d="M19.1 4.9C23 8.8 23 15.2 19.1 19.1"></path></svg>`,
        unknown: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>`,
        warning: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a1 1 0 0 0 .86 1.5h18.64a1 1 0 0 0 .86-1.5L13.71 3.86a1 1 0 0 0-1.72 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>`,
        calendar: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>`,
        check: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>`
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
            if (grid) grid.classList.add('hidden');
            empty.classList.remove('hidden');
            return;
        }

        if (grid) grid.classList.remove('hidden');
        empty.classList.add('hidden');

        const badgeColors = {
            'PENDING': 'bg-amber-500/10 text-amber-500 border border-amber-500/20',
            'WAITING_RELEASE': 'bg-sky-500/10 text-sky-500 border border-sky-500/20',
            'UNAVAILABLE': 'bg-rose-500/10 text-rose-500 border border-rose-500/20',
            'COMPLETED': 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20'
        };

        grid.innerHTML = requests.map((req, idx) => {
            const releaseSourceStr = req.releaseSource || 'Unknown';
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
                                <div class="flex items-center gap-1.5 shrink-0">
                                    ${req.is4k ? `
                                    <span class="text-[0.75rem] font-black text-amber-500 dark:text-amber-400 leading-none select-none cursor-default shrink-0" title="4K Request">4K</span>
                                    ` : ''}
                                    <span class="text-sm" title="${req.mediaType === 'tv' ? 'TV Show' : 'Movie'}">${req.mediaType === 'tv' ? Components.icons.tv : Components.icons.movie}</span>
                                </div>
                            </div>
                            <div class="text-[0.8rem] text-slate-500 dark:text-slate-400 flex flex-col gap-1">
                                <span class="flex items-center gap-1.5" title="Request Date: ${formatDate(req.createdAt)}">
                                    <span class="flex items-center" title="Request Date: ${formatDate(req.createdAt)}">${Components.icons.clock}</span> ${timeAgo(req.createdAt)}
                                </span>
                                ${req.releaseDate && req.status !== 'COMPLETED' ? `
                                <div class="flex items-center gap-1.5" title="Release Date: ${formatDate(req.releaseDate)} (${releaseSourceStr})">
                                    <span class="flex items-center" title="Release Date: ${formatDate(req.releaseDate)} (${releaseSourceStr})">${releaseIcon(releaseSourceStr)}</span>
                                    <span>${formatDate(req.releaseDate)}</span>
                                </div>` : (!req.releaseDate && req.status !== 'COMPLETED' ? `
                                <div class="flex items-center gap-1.5 text-red-500" title="No release date">
                                    ${Components.icons.warning}<span>No release date</span>
                                </div>` : '')}
                                ${req.status === 'COMPLETED' && req.fulfilledAt ? `
                                <span class="flex items-center gap-1.5 title="Fulfillment Date: ${formatDate(req.fulfilledAt)}">
                                    <span class="flex items-center" title="Fulfillment Date: ${formatDate(req.fulfilledAt)}">${Components.icons.check}</span> ${formatDate(req.fulfilledAt)}
                                </span>` : ''}
                            </div>
                            <div class="flex flex-wrap gap-2 items-center justify-between">
                                <div class="status-badge ${req.status} inline-flex items-center gap-1.5 py-1 px-2.5 rounded-full text-[0.7rem] font-bold uppercase tracking-wider w-fit ${badgeColors[req.status] || ''}">
                                    <span class="w-1.5 h-1.5 rounded-full animate-pulse ${Components.statusDotColors[req.status] || ''}"></span>
                                    ${statusLabel(req.status)}
                                </div>
                                <div class="flex items-center gap-1.5">
                                    ${req.requestedSeasons ? `
                                    <div class="inline-flex items-center gap-1 py-1 px-2.5 rounded-full text-[0.7rem] font-bold uppercase tracking-wider w-fit bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700">
                                        <span>${req.requestedSeasons}</span>
                                    </div>
                                    ` : ''}
                                </div>
                            </div>
                        </div>
                    </div>
                    ${req.reason ? `
                    <div class="px-4 pb-3 text-[0.8rem]">
                        <div class="flex items-center gap-2 bg-rose-500/5 dark:bg-rose-500/5 p-2 rounded-lg border border-rose-500/10 text-rose-600 dark:text-rose-450">
                            <span class="shrink-0 font-semibold flex items-center justify-center" title="Reason for unavailable">${Components.icons.ban}</span>
                            <span class="break-words w-full" id="reason-text-${req.seerrRequestId}">${req.reason}</span>
                        </div>
                    </div>` : ''}
                    <div class="p-4 pt-0.5 mt-auto">
                        <div class="flex flex-row items-center gap-2 w-full">
                            ${Components._renderTriageAction(req)}
                            ${req.serviceUrl
                    ? `<a class="action-btn flex-1 flex items-center justify-center gap-1.5 py-2.5 px-2.5 rounded border border-slate-200 dark:border-slate-700 bg-white/60 dark:bg-slate-800/40 text-slate-600 dark:text-slate-400 text-xs font-semibold cursor-pointer transition-all duration-150 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-350 dark:hover:border-slate-600" href="${req.serviceUrl}" title="Open in Service">
                                     ${req.mediaType === 'tv' ? Components.icons.sonarr : Components.icons.radarr}
                                     <span>${req.mediaType === 'tv' ? 'Sonarr' : 'Radarr'}</span>
                                   </a>`
                    : ''
                }
                            <a class="action-btn flex-1 flex items-center justify-center gap-1.5 py-2.5 px-2.5 rounded border border-slate-200 dark:border-slate-700 bg-white/60 dark:bg-slate-800/40 text-slate-600 dark:text-slate-400 text-xs font-semibold cursor-pointer transition-all duration-150 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-350 dark:hover:border-slate-600" href="${seerrLink}" title="Seerr">
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
            <div class="triage-dropdown group/dropdown relative flex-1 flex" id="triage-dropdown-${req.seerrRequestId}">
                <button class="action-btn flex-1 flex items-center justify-center gap-1.5 py-2.5 px-2.5 rounded border border-slate-200 dark:border-slate-700 bg-white/60 dark:bg-slate-800/40 text-slate-600 dark:text-slate-400 text-xs font-semibold cursor-pointer transition-all duration-150 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-350 dark:hover:border-slate-600" onclick="Components.toggleTriageMenu(${req.seerrRequestId}, event)" title="Change Status">
                    ${Components.icons.triage}
                    <span>Status</span>
                    ${Components.icons.chevron}
                </button>
                <div class="triage-menu absolute bottom-[calc(100%+6px)] left-0 min-w-[180px] p-1.5 rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-xl z-50 opacity-0 invisible translate-y-2 transition-all duration-150 group-[.open]/dropdown:opacity-100 group-[.open]/dropdown:visible group-[.open]/dropdown:translate-y-0">
                    ${req.status !== 'PENDING' ? `
                    <button class="triage-option flex items-center gap-2 w-full py-2 px-3 border-none rounded bg-transparent text-slate-700 dark:text-slate-200 text-xs font-medium cursor-pointer transition-colors duration-150 hover:bg-slate-100 dark:hover:bg-slate-700" data-status="PENDING" onclick="App.setTriage(${req.seerrRequestId}, 'PENDING')">
                        <span class="w-2 h-2 rounded-full bg-amber-500"></span>
                        <span>Pending</span>
                    </button>` : ''}
                    ${req.status !== 'WAITING_RELEASE' ? `
                    <button class="triage-option flex items-center gap-2 w-full py-2 px-3 border-none rounded bg-transparent text-slate-700 dark:text-slate-200 text-xs font-medium cursor-pointer transition-colors duration-150 hover:bg-slate-100 dark:hover:bg-slate-700" data-status="WAITING_RELEASE" onclick="App.setTriage(${req.seerrRequestId}, 'WAITING_RELEASE')">
                        <span class="w-2 h-2 rounded-full bg-sky-500"></span>
                        <span>Waiting Release</span>
                    </button>` : ''}
                    ${req.status !== 'UNAVAILABLE' ? `
                    <button class="triage-option flex items-center gap-2 w-full py-2 px-3 border-none rounded bg-transparent text-slate-700 dark:text-slate-200 text-xs font-medium cursor-pointer transition-colors duration-150 hover:bg-slate-100 dark:hover:bg-slate-700" data-status="UNAVAILABLE" onclick="App.setTriage(${req.seerrRequestId}, 'UNAVAILABLE')">
                        <span class="w-2 h-2 rounded-full bg-rose-500"></span>
                        <span>Unavailable</span>
                    </button>` : ''}
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
