// ZRP Onboarding Tour — powered by Driver.js
// Loaded as a module; exposes window.startTour() and auto-starts for first-time users.

(function () {
  'use strict';

  /* ── Tour step definitions ─────────────────────────────────────── */
  const STEPS = [
    // 1 Dashboard
    { element: '[data-route="dashboard"]', title: 'Dashboard', description: 'Your command center. View KPI cards, activity feed, charts, low-stock alerts, and customize widgets to fit your workflow.', route: 'dashboard' },
    // 2 Calendar
    { element: '[data-route="calendar"]', title: 'Calendar', description: 'Monthly calendar view showing Work Order and PO due dates so nothing slips through the cracks.', route: 'calendar' },
    // 3 Parts & BOM
    { element: '[data-route="parts"]', title: 'Parts & BOM', description: 'Full PLM parts list with search, detail views, and an interactive BOM tree viewer showing parent-child relationships.', route: 'parts' },
    // 4 ECOs
    { element: '[data-route="ecos"]', title: 'Engineering Change Orders', description: 'Create, edit, and approve ECOs. Link affected IPNs and track the full change workflow from draft to implementation.', route: 'ecos' },
    // 5 Documents
    { element: '[data-route="docs"]', title: 'Documents', description: 'Manage engineering documents, specs, and drawings linked to parts and ECOs.', route: 'docs' },
    // 6 Inventory
    { element: '[data-route="inventory"]', title: 'Inventory', description: 'Real-time stock levels, full history, and quick stock-add. Color-coded alerts for low-stock items.', route: 'inventory' },
    // 7 Procurement / POs
    { element: '[data-route="procurement"]', title: 'Purchase Orders', description: 'Create POs with line items, auto-generate orders from BOM shortages, and track receiving.', route: 'procurement' },
    // 8 Vendors
    { element: '[data-route="vendors"]', title: 'Vendors & Suppliers', description: 'Manage your supply base — CRUD operations, price catalogs, historical price trends, and preferred-vendor flags.', route: 'vendors' },
    // 9 Work Orders
    { element: '[data-route="workorders"]', title: 'Work Orders', description: 'Create and manage manufacturing work orders. BOM shortage highlighting shows what\'s missing before you start.', route: 'workorders' },
    // 10 Testing
    { element: '[data-route="testing"]', title: 'Test Records', description: 'Log test results — pass/fail with attachments. Full traceability back to work orders and parts.', route: 'testing' },
    // 11 NCRs
    { element: '[data-route="ncr"]', title: 'Non-Conformance Reports', description: 'Track quality issues by severity. Document root cause analysis and auto-link NCRs to ECOs for corrective action.', route: 'ncr' },
    // 12 Devices
    { element: '[data-route="devices"]', title: 'Device Registry', description: 'Register deployed devices, view firmware history, and bulk-import from CSV. Track every unit in the field.', route: 'devices' },
    // 13 Firmware
    { element: '[data-route="firmware"]', title: 'Firmware Campaigns', description: 'Create OTA firmware campaigns, target specific device groups, and monitor rollout progress.', route: 'firmware' },
    // 14 RMAs
    { element: '[data-route="rma"]', title: 'Return Authorizations', description: 'Create RMAs, track status from receipt through diagnosis to resolution. Full field-service workflow.', route: 'rma' },
    // 15 Quotes
    { element: '[data-route="quotes"]', title: 'Quotes & Pricing', description: 'Build customer quotes with automatic margin analysis — compare BOM cost vs quoted price in real-time.', route: 'quotes' },
    // 16 Reports
    { element: '[data-route="reports"]', title: 'Reports', description: 'Generate on-demand reports across all modules. Export to CSV or HTML for stakeholder review.', route: 'reports' },
    // 17 Users
    { element: '[data-route="users"]', title: 'User Management', description: 'Manage team access with role-based permissions — Admin, User, and Read-Only roles.', route: 'users' },
    // 18 Audit Log
    { element: '[data-route="audit"]', title: 'Audit Log', description: 'Complete change history across every module. See who changed what, when, with before/after diffs.', route: 'audit' },
    // 19 API Keys
    { element: '[data-route="apikeys"]', title: 'API Keys', description: 'Generate and revoke API keys for integrations. Scoped access with expiration dates.', route: 'apikeys' },
    // 20 Global Search
    { element: '#global-search', title: 'Global Search', description: 'Search across parts, ECOs, devices, work orders, and more — all from one input. Results grouped by module.', side: 'bottom' },
    // 21 Dark Mode
    { element: '#dark-toggle', title: 'Dark Mode', description: 'Toggle between light and dark themes. Your preference is saved automatically.', side: 'bottom' },
    // 22 Notifications
    { element: '#notif-bell', title: 'Notifications', description: 'Real-time alerts for low stock, ECO approvals, RMA updates, and more. Never miss an important event.', side: 'bottom' },
    // 23 Keyboard Shortcuts
    { element: '#page-title', title: 'Keyboard Shortcuts & More', description: 'Press <kbd>?</kbd> to see all keyboard shortcuts. Use batch checkboxes for bulk operations, and drag-drop files onto ECOs and NCRs for attachments.', side: 'bottom' },
  ];

  /* ── CSS (injected once) ───────────────────────────────────────── */
  function injectCSS() {
    if (document.getElementById('zrp-tour-css')) return;
    const style = document.createElement('style');
    style.id = 'zrp-tour-css';
    style.textContent = `
      /* Overlay */
      .zt-overlay { position:fixed;inset:0;z-index:9998;pointer-events:auto;transition:opacity .3s; }
      .zt-overlay-bg { position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:9998;transition:opacity .3s; }
      .zt-highlight { position:fixed;z-index:9999;box-shadow:0 0 0 4000px rgba(0,0,0,.55);border-radius:8px;transition:all .35s cubic-bezier(.4,0,.2,1);pointer-events:none; }
      .zt-highlight::after { content:'';position:absolute;inset:-4px;border:2px solid #3b82f6;border-radius:10px;animation:zt-pulse 1.5s ease-in-out infinite; }
      @keyframes zt-pulse { 0%,100%{opacity:.6;transform:scale(1)} 50%{opacity:1;transform:scale(1.01)} }

      /* Popover */
      .zt-popover { position:fixed;z-index:10000;width:360px;max-width:90vw;background:#fff;border-radius:12px;box-shadow:0 20px 60px rgba(0,0,0,.25);padding:0;opacity:0;transform:translateY(12px);transition:opacity .3s,transform .3s;font-family:system-ui,-apple-system,sans-serif; }
      .zt-popover.visible { opacity:1;transform:translateY(0); }
      .dark .zt-popover { background:#1f2937;color:#e5e7eb;box-shadow:0 20px 60px rgba(0,0,0,.5); }

      .zt-pop-header { padding:16px 20px 0;display:flex;justify-content:space-between;align-items:center; }
      .zt-pop-step { font-size:11px;font-weight:600;color:#3b82f6;text-transform:uppercase;letter-spacing:.5px; }
      .zt-pop-close { cursor:pointer;color:#9ca3af;font-size:18px;line-height:1; }
      .zt-pop-close:hover { color:#ef4444; }
      .zt-pop-title { padding:8px 20px 0;font-size:16px;font-weight:700;color:#111827; }
      .dark .zt-pop-title { color:#f3f4f6; }
      .zt-pop-desc { padding:8px 20px 0;font-size:13px;line-height:1.6;color:#6b7280; }
      .dark .zt-pop-desc { color:#9ca3af; }
      .zt-pop-desc kbd { background:#e5e7eb;padding:1px 6px;border-radius:4px;font-size:12px;font-family:monospace; }
      .dark .zt-pop-desc kbd { background:#374151; }

      /* Progress bar */
      .zt-progress { margin:16px 20px 0;height:3px;background:#e5e7eb;border-radius:2px;overflow:hidden; }
      .dark .zt-progress { background:#374151; }
      .zt-progress-fill { height:100%;background:linear-gradient(90deg,#3b82f6,#6366f1);border-radius:2px;transition:width .35s ease; }

      /* Buttons */
      .zt-pop-btns { padding:12px 20px 16px;display:flex;justify-content:space-between;align-items:center;gap:8px; }
      .zt-btn { padding:7px 16px;border-radius:8px;font-size:13px;font-weight:600;cursor:pointer;border:none;transition:all .15s; }
      .zt-btn-prev { background:#f3f4f6;color:#374151; }
      .zt-btn-prev:hover { background:#e5e7eb; }
      .dark .zt-btn-prev { background:#374151;color:#d1d5db; }
      .zt-btn-next { background:#3b82f6;color:#fff; }
      .zt-btn-next:hover { background:#2563eb; }
      .zt-btn-skip { background:transparent;color:#9ca3af;font-size:12px; }
      .zt-btn-skip:hover { color:#ef4444; }

      /* Start Tour button in header */
      #tour-start-btn { cursor:pointer;display:inline-flex;align-items:center;gap:4px;padding:4px 10px;border-radius:6px;font-size:12px;font-weight:600;color:#3b82f6;background:transparent;border:1px solid #3b82f6;transition:all .15s; }
      #tour-start-btn:hover { background:#3b82f6;color:#fff; }
      .dark #tour-start-btn { color:#60a5fa;border-color:#60a5fa; }
      .dark #tour-start-btn:hover { background:#60a5fa;color:#111827; }
    `;
    document.head.appendChild(style);
  }

  /* ── Tour engine ────────────────────────────────────────────────── */
  let current = -1;
  let overlayBg = null;
  let highlightEl = null;
  let popoverEl = null;
  let autoAdvanceTimer = null;
  let autoPlayMode = false;

  function createOverlay() {
    overlayBg = document.createElement('div');
    overlayBg.className = 'zt-overlay-bg';
    overlayBg.onclick = () => endTour();
    document.body.appendChild(overlayBg);

    highlightEl = document.createElement('div');
    highlightEl.className = 'zt-highlight';
    document.body.appendChild(highlightEl);

    popoverEl = document.createElement('div');
    popoverEl.className = 'zt-popover';
    document.body.appendChild(popoverEl);
  }

  function removeOverlay() {
    [overlayBg, highlightEl, popoverEl].forEach(el => el && el.remove());
    overlayBg = highlightEl = popoverEl = null;
  }

  function positionPopover(targetRect, side) {
    if (!popoverEl) return;
    const pad = 12;
    const pw = popoverEl.offsetWidth || 360;
    const ph = popoverEl.offsetHeight || 200;
    let top, left;

    side = side || 'right';

    // Try right
    if (side === 'right' || side === 'auto') {
      left = targetRect.right + pad;
      top = targetRect.top;
      if (left + pw > window.innerWidth) { side = 'bottom'; }
    }
    if (side === 'bottom') {
      top = targetRect.bottom + pad;
      left = targetRect.left;
      if (top + ph > window.innerHeight) { side = 'top'; }
    }
    if (side === 'top') {
      top = targetRect.top - ph - pad;
      left = targetRect.left;
    }
    if (side === 'left') {
      left = targetRect.left - pw - pad;
      top = targetRect.top;
    }
    if (side === 'right') {
      left = targetRect.right + pad;
      top = targetRect.top;
    }

    // Clamp
    left = Math.max(8, Math.min(left, window.innerWidth - pw - 8));
    top = Math.max(8, Math.min(top, window.innerHeight - ph - 8));

    popoverEl.style.left = left + 'px';
    popoverEl.style.top = top + 'px';
  }

  async function showStep(idx) {
    if (idx < 0 || idx >= STEPS.length) { endTour(); return; }
    current = idx;
    const step = STEPS[idx];

    // Navigate if needed
    if (step.route && typeof window.navigate === 'function') {
      window.navigate(step.route);
      await new Promise(r => setTimeout(r, 600));
    }

    const target = document.querySelector(step.element);
    if (!target) {
      // Skip missing element
      if (idx < STEPS.length - 1) { showStep(idx + 1); return; }
      endTour(); return;
    }

    // Scroll into view
    target.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    await new Promise(r => setTimeout(r, 200));

    const rect = target.getBoundingClientRect();

    // Position highlight
    if (highlightEl) {
      highlightEl.style.left = (rect.left - 4) + 'px';
      highlightEl.style.top = (rect.top - 4) + 'px';
      highlightEl.style.width = (rect.width + 8) + 'px';
      highlightEl.style.height = (rect.height + 8) + 'px';
    }

    // Build popover
    const pct = ((idx + 1) / STEPS.length * 100).toFixed(0);
    if (popoverEl) {
      popoverEl.classList.remove('visible');
      popoverEl.innerHTML = `
        <div class="zt-pop-header">
          <span class="zt-pop-step">Step ${idx + 1} of ${STEPS.length}</span>
          <span class="zt-pop-close" id="zt-close">&times;</span>
        </div>
        <div class="zt-pop-title">${step.title}</div>
        <div class="zt-pop-desc">${step.description}</div>
        <div class="zt-progress"><div class="zt-progress-fill" style="width:${pct}%"></div></div>
        <div class="zt-pop-btns">
          <button class="zt-btn zt-btn-skip" id="zt-skip">Skip tour</button>
          <div style="display:flex;gap:6px">
            ${idx > 0 ? '<button class="zt-btn zt-btn-prev" id="zt-prev">← Back</button>' : ''}
            <button class="zt-btn zt-btn-next" id="zt-next">${idx === STEPS.length - 1 ? 'Finish ✓' : 'Next →'}</button>
          </div>
        </div>`;

      positionPopover(rect, step.side || 'auto');

      requestAnimationFrame(() => {
        popoverEl.classList.add('visible');
        positionPopover(rect, step.side || 'auto');
      });

      // Bind buttons
      const nextBtn = document.getElementById('zt-next');
      const prevBtn = document.getElementById('zt-prev');
      const skipBtn = document.getElementById('zt-skip');
      const closeBtn = document.getElementById('zt-close');
      if (nextBtn) nextBtn.onclick = () => showStep(current + 1);
      if (prevBtn) prevBtn.onclick = () => showStep(current - 1);
      if (skipBtn) skipBtn.onclick = () => endTour();
      if (closeBtn) closeBtn.onclick = () => endTour();
    }

    // Auto-advance in autoplay mode
    if (autoPlayMode) {
      clearTimeout(autoAdvanceTimer);
      autoAdvanceTimer = setTimeout(() => {
        if (current < STEPS.length - 1) showStep(current + 1);
        else endTour();
      }, 3500);
    }
  }

  function startTour(autoplay) {
    // If tour is already active, don't restart
    if (current >= 0) return;
    autoPlayMode = !!autoplay;
    localStorage.setItem('zrp-tour-seen', 'true');
    injectCSS();
    createOverlay();
    showStep(0);
    // Dispatch event for testing
    window.dispatchEvent(new CustomEvent('zrp-tour-start'));
  }

  function endTour() {
    clearTimeout(autoAdvanceTimer);
    autoPlayMode = false;
    current = -1;
    removeOverlay();
    window.dispatchEvent(new CustomEvent('zrp-tour-end'));
  }

  /* ── Inject "Start Tour" button into header ─────────────────────── */
  function injectButton() {
    if (document.getElementById('tour-start-btn')) return;
    // Insert near the dark mode toggle
    const darkBtn = document.getElementById('dark-toggle');
    if (!darkBtn) return;
    const btn = document.createElement('button');
    btn.id = 'tour-start-btn';
    btn.title = 'Start guided tour';
    btn.innerHTML = '? Tour';
    btn.onclick = () => startTour(false);
    darkBtn.parentElement.insertBefore(btn, darkBtn);
  }

  /* ── Auto-start for first-time users ────────────────────────────── */
  function maybeAutoStart() {
    if (!localStorage.getItem('zrp-tour-seen')) {
      setTimeout(() => startTour(false), 1500);
    }
  }

  /* ── Init on app visible ────────────────────────────────────────── */
  function init() {
    const app = document.getElementById('app');
    if (!app) return;
    const obs = new MutationObserver(() => {
      if (!app.classList.contains('hidden')) {
        obs.disconnect();
        injectCSS();
        injectButton();
        maybeAutoStart();
      }
    });
    obs.observe(app, { attributes: true, attributeFilter: ['class'] });
    // Also check immediately
    if (!app.classList.contains('hidden')) {
      obs.disconnect();
      injectCSS();
      injectButton();
      maybeAutoStart();
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose globally
  window.startTour = startTour;
  window.endTour = endTour;
  window.getTourState = () => ({ current, total: STEPS.length, active: current >= 0 });
})();
