window.module_reports = {
  render: async (container) => {
    const reports = [
      { id: 'inventory-valuation', name: 'Inventory Valuation', icon: '📦', desc: 'Qty × unit price from latest PO, grouped by category' },
      { id: 'open-ecos', name: 'Open ECOs by Priority', icon: '🔄', desc: 'Draft/review ECOs sorted by priority with age' },
      { id: 'wo-throughput', name: 'WO Throughput', icon: '⚙️', desc: 'Work orders completed with cycle time analysis' },
      { id: 'low-stock', name: 'Low Stock Report', icon: '⚠️', desc: 'Items below reorder point with suggested orders' },
      { id: 'ncr-summary', name: 'NCR Summary', icon: '📋', desc: 'Open NCRs by severity and defect type' }
    ];

    let currentReport = null;
    let woDays = 30;

    function renderMenu() {
      container.innerHTML = `<div class="card mb-4">
        <h2 class="text-lg font-semibold mb-4">Reports</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          ${reports.map(r => `<div class="border rounded-lg p-4 hover:bg-blue-50 dark:hover:bg-gray-700 cursor-pointer transition-colors" onclick="window._reportRun('${r.id}')">
            <div class="text-2xl mb-2">${r.icon}</div>
            <div class="font-semibold">${r.name}</div>
            <div class="text-sm text-gray-500">${r.desc}</div>
          </div>`).join('')}
        </div>
      </div>`;
    }

    async function runReport(id) {
      currentReport = id;
      const params = id === 'wo-throughput' ? `?days=${woDays}` : '';
      container.innerHTML = '<div class="card"><p class="text-gray-400">Loading report...</p></div>';
      try {
        const res = await api('GET', 'reports/' + id + params);
        const data = res.data;
        let html = `<div class="card">
          <div class="flex justify-between items-center mb-4">
            <div class="flex items-center gap-3">
              <button class="btn btn-secondary" onclick="window._reportBack()">← Back</button>
              <h2 class="text-lg font-semibold">${reports.find(r=>r.id===id)?.name || id}</h2>
            </div>
            <button class="btn btn-secondary" onclick="window._reportExport('${id}')">📥 Export CSV</button>
          </div>`;

        if (id === 'inventory-valuation') {
          html += renderInventoryValuation(data);
        } else if (id === 'open-ecos') {
          html += renderOpenECOs(data);
        } else if (id === 'wo-throughput') {
          html += renderWOThroughput(data);
        } else if (id === 'low-stock') {
          html += renderLowStock(data);
        } else if (id === 'ncr-summary') {
          html += renderNCRSummary(data);
        }
        html += '</div>';
        container.innerHTML = html;
      } catch(e) {
        container.innerHTML = `<div class="card"><p class="text-red-500">${e.message}</p><button class="btn btn-secondary mt-2" onclick="window._reportBack()">← Back</button></div>`;
      }
    }

    function renderInventoryValuation(data) {
      let html = '';
      for (const g of (data.groups || [])) {
        html += `<h3 class="font-semibold mt-4 mb-2">${g.category} <span class="text-sm text-gray-500">(Subtotal: $${g.subtotal.toFixed(2)})</span></h3>`;
        html += `<div class="overflow-x-auto"><table class="w-full text-sm mb-2"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-1">IPN</th><th class="pb-1">Description</th><th class="pb-1 text-right">Qty</th><th class="pb-1 text-right">Unit Price</th><th class="pb-1 text-right">Subtotal</th>
        </tr></thead><tbody>`;
        for (const it of g.items) {
          html += `<tr class="border-b border-gray-100"><td class="py-1 font-mono">${it.ipn}</td><td class="py-1">${it.description}</td>
            <td class="py-1 text-right">${it.qty_on_hand}</td><td class="py-1 text-right">$${it.unit_price.toFixed(4)}</td>
            <td class="py-1 text-right font-medium">$${it.subtotal.toFixed(2)}</td></tr>`;
        }
        html += '</tbody></table></div>';
      }
      html += `<div class="mt-4 pt-4 border-t text-lg font-bold text-right">Grand Total: $${(data.grand_total||0).toFixed(2)}</div>`;
      return html;
    }

    function renderOpenECOs(data) {
      let html = `<div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
        <th class="pb-2">ID</th><th class="pb-2">Title</th><th class="pb-2">Status</th><th class="pb-2">Priority</th><th class="pb-2">Created By</th><th class="pb-2 text-right">Age (Days)</th>
      </tr></thead><tbody>`;
      for (const e of data) {
        html += `<tr class="border-b border-gray-100"><td class="py-2 font-mono text-blue-600">${e.id}</td><td class="py-2">${e.title}</td>
          <td class="py-2">${badge(e.status)}</td><td class="py-2">${badge(e.priority)}</td>
          <td class="py-2">${e.created_by}</td><td class="py-2 text-right">${e.age_days}</td></tr>`;
      }
      html += '</tbody></table></div>';
      if (data.length === 0) html = '<p class="text-gray-400 text-center py-4">No open ECOs</p>';
      return html;
    }

    function renderWOThroughput(data) {
      let html = `<div class="flex gap-2 mb-4">
        ${[30,60,90].map(d => `<button class="btn ${woDays===d?'btn-primary':'btn-secondary'}" onclick="window._reportWODays(${d})">${d} Days</button>`).join('')}
      </div>`;
      html += `<div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-4">
        <div class="bg-blue-50 dark:bg-gray-700 rounded-lg p-4"><div class="text-2xl font-bold">${data.total_completed}</div><div class="text-sm text-gray-500">Completed</div></div>
        <div class="bg-green-50 dark:bg-gray-700 rounded-lg p-4"><div class="text-2xl font-bold">${data.avg_cycle_time_days}</div><div class="text-sm text-gray-500">Avg Cycle (Days)</div></div>
      </div>`;
      if (Object.keys(data.count_by_status||{}).length > 0) {
        html += `<h3 class="font-semibold mb-2">By Status</h3><table class="w-full text-sm"><tbody>`;
        for (const [s,c] of Object.entries(data.count_by_status)) {
          html += `<tr class="border-b border-gray-100"><td class="py-1">${badge(s)}</td><td class="py-1 text-right font-medium">${c}</td></tr>`;
        }
        html += '</tbody></table>';
      }
      return html;
    }

    function renderLowStock(data) {
      let html = `<div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
        <th class="pb-2">IPN</th><th class="pb-2">Description</th><th class="pb-2 text-right">On Hand</th><th class="pb-2 text-right">Reorder Point</th><th class="pb-2 text-right">Suggested Order</th><th class="pb-2"></th>
      </tr></thead><tbody>`;
      for (const it of data) {
        html += `<tr class="border-b border-gray-100"><td class="py-2 font-mono">${it.ipn}</td><td class="py-2">${it.description}</td>
          <td class="py-2 text-right text-red-600 font-medium">${it.qty_on_hand}</td><td class="py-2 text-right">${it.reorder_point}</td>
          <td class="py-2 text-right font-medium">${it.suggested_order}</td>
          <td class="py-2"><button class="text-blue-600 text-xs hover:underline" onclick="navigate('procurement')">Create PO →</button></td></tr>`;
      }
      html += '</tbody></table></div>';
      if (data.length === 0) html = '<p class="text-green-600 text-center py-4">✅ All items above reorder point</p>';
      return html;
    }

    function renderNCRSummary(data) {
      let html = `<div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-4">
        <div class="bg-red-50 dark:bg-gray-700 rounded-lg p-4"><div class="text-2xl font-bold">${data.total_open}</div><div class="text-sm text-gray-500">Open NCRs</div></div>
        <div class="bg-blue-50 dark:bg-gray-700 rounded-lg p-4"><div class="text-2xl font-bold">${data.avg_resolve_days}</div><div class="text-sm text-gray-500">Avg Resolve (Days)</div></div>
      </div>`;
      if (Object.keys(data.by_severity||{}).length > 0) {
        html += `<h3 class="font-semibold mb-2">By Severity</h3><table class="w-full text-sm mb-4"><tbody>`;
        for (const [s,c] of Object.entries(data.by_severity)) {
          html += `<tr class="border-b border-gray-100"><td class="py-1">${badge(s)}</td><td class="py-1 text-right font-medium">${c}</td></tr>`;
        }
        html += '</tbody></table>';
      }
      if (Object.keys(data.by_defect_type||{}).length > 0) {
        html += `<h3 class="font-semibold mb-2">By Defect Type</h3><table class="w-full text-sm"><tbody>`;
        for (const [d,c] of Object.entries(data.by_defect_type)) {
          html += `<tr class="border-b border-gray-100"><td class="py-1">${d}</td><td class="py-1 text-right font-medium">${c}</td></tr>`;
        }
        html += '</tbody></table>';
      }
      return html;
    }

    window._reportRun = (id) => runReport(id);
    window._reportBack = () => renderMenu();
    window._reportExport = (id) => {
      const params = id === 'wo-throughput' ? `&days=${woDays}` : '';
      window.open('/api/v1/reports/' + id + '?format=csv' + params, '_blank');
    };
    window._reportWODays = (d) => { woDays = d; runReport('wo-throughput'); };

    renderMenu();
  }
};
