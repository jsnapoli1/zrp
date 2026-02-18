window.module_audit = {
  render: async (container) => {
    const moduleColors = {
      device: 'bg-blue-100 text-blue-800',
      firmware: 'bg-purple-100 text-purple-800',
      po: 'bg-green-100 text-green-800',
      eco: 'bg-yellow-100 text-yellow-800',
      inventory: 'bg-teal-100 text-teal-800',
      ncr: 'bg-red-100 text-red-800',
      rma: 'bg-orange-100 text-orange-800',
      quote: 'bg-pink-100 text-pink-800',
      workorder: 'bg-indigo-100 text-indigo-800',
      doc: 'bg-gray-100 text-gray-800'
    };
    function moduleBadge(mod) {
      const cls = moduleColors[mod] || 'bg-gray-100 text-gray-700';
      return `<span class="badge ${cls}">${mod}</span>`;
    }

    let filterModule = '', filterUser = '', filterFrom = '', filterTo = '';

    async function load() {
      const params = new URLSearchParams({ limit: 200 });
      if (filterModule) params.set('module', filterModule);
      if (filterUser) params.set('user', filterUser);
      if (filterFrom) params.set('from', filterFrom);
      if (filterTo) params.set('to', filterTo);
      const res = await api('GET', 'audit?' + params);
      const items = res.data || [];

      // Get unique modules and users for filters
      const modules = [...new Set(items.map(i => i.module))].sort();
      const users = [...new Set(items.map(i => i.username))].sort();

      container.innerHTML = `<div class="card">
        <h2 class="text-lg font-semibold mb-4">Audit Log</h2>
        <div class="flex gap-3 mb-4 flex-wrap items-end">
          <div>
            <label class="label">Module</label>
            <select class="input w-40" id="audit-filter-module">
              <option value="">All</option>
              ${modules.map(m => `<option value="${m}" ${filterModule===m?'selected':''}>${m}</option>`).join('')}
            </select>
          </div>
          <div>
            <label class="label">User</label>
            <select class="input w-40" id="audit-filter-user">
              <option value="">All</option>
              ${users.map(u => `<option value="${u}" ${filterUser===u?'selected':''}>${u}</option>`).join('')}
            </select>
          </div>
          <div>
            <label class="label">From</label>
            <input type="date" class="input w-40" id="audit-filter-from" value="${filterFrom}">
          </div>
          <div>
            <label class="label">To</label>
            <input type="date" class="input w-40" id="audit-filter-to" value="${filterTo}">
          </div>
          <button class="btn btn-primary" id="audit-apply">Apply</button>
          <button class="btn btn-secondary" id="audit-clear">Clear</button>
        </div>
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">Timestamp</th><th class="pb-2">User</th><th class="pb-2">Module</th><th class="pb-2">Action</th><th class="pb-2">Record ID</th><th class="pb-2">Summary</th>
        </tr></thead><tbody>
          ${items.map(e => `<tr class="border-b border-gray-100">
            <td class="py-2 text-gray-500 text-xs whitespace-nowrap">${e.created_at}</td>
            <td class="py-2 font-medium">${e.username}</td>
            <td class="py-2">${moduleBadge(e.module)}</td>
            <td class="py-2">${e.action}</td>
            <td class="py-2 font-mono text-blue-600">${e.record_id}</td>
            <td class="py-2 text-gray-600 truncate max-w-[300px]">${e.summary||''}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No audit entries</p>':''}
        <div class="text-sm text-gray-400 mt-2">${items.length} entries</div>
      </div>`;

      document.getElementById('audit-apply').onclick = () => {
        filterModule = document.getElementById('audit-filter-module').value;
        filterUser = document.getElementById('audit-filter-user').value;
        filterFrom = document.getElementById('audit-filter-from').value;
        filterTo = document.getElementById('audit-filter-to').value;
        load();
      };
      document.getElementById('audit-clear').onclick = () => {
        filterModule = filterUser = filterFrom = filterTo = '';
        load();
      };
    }
    load();
  }
};
