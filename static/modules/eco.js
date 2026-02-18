window.module_ecos = {
  render: async (container) => {
    const bulk = setupBulkOps(container, 'ecos/bulk', [
      {action:'approve', label:'✓ Approve', class:'bg-green-600 hover:bg-green-700 text-white'},
      {action:'reject', label:'✗ Reject', class:'bg-yellow-600 hover:bg-yellow-700 text-white'},
      {action:'implement', label:'🚀 Implement', class:'bg-blue-600 hover:bg-blue-700 text-white'},
      {action:'delete', label:'🗑 Delete', class:'bg-red-600 hover:bg-red-700 text-white'},
    ]);
    async function load() {
      const res = await api('GET', 'ecos');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Engineering Change Orders</h2>
          <button class="btn btn-primary" onclick="window._ecoCreate()">+ New ECO</button>
        </div>
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
          <p class="text-gray-500 font-medium">No ECOs yet</p>
          <p class="text-gray-400 text-sm mt-1">Create your first Engineering Change Order to get started</p>
          <button class="btn btn-primary mt-4" onclick="window._ecoCreate()">+ New ECO</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2 w-8">${bulk.headerCheckbox()}</th>
          <th class="pb-2">ID</th><th class="pb-2">Title</th><th class="pb-2">Status</th><th class="pb-2">Priority</th><th class="pb-2">Created</th><th class="pb-2 w-8"></th>
        </tr></thead><tbody>
          ${items.map(e => `<tr class="table-row border-b border-gray-100" onclick="window._ecoEdit('${e.id}')">
            <td class="py-2">${bulk.checkbox(e.id)}</td>
            <td class="py-2 font-mono text-blue-600">${e.id}</td>
            <td class="py-2">${e.title}${e.ncr_id ? ' <span class="badge bg-purple-100 text-purple-800">From '+e.ncr_id+'</span>' : ''}</td>
            <td class="py-2">${badge(e.status)}</td>
            <td class="py-2">${badge(e.priority)}</td>
            <td class="py-2 text-gray-500">${e.created_at?.substring(0,10)}</td>
            <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
          </tr>`).join('')}
        </tbody></table>
        </div>`}
      </div>`;
      bulk.init();
    }
    container.addEventListener('bulk-reload', load);
    const formHTML = (e={}) => `
      <div class="space-y-3">
        <div><label class="label">Title</label><input class="input" data-field="title" value="${e.title||''}"></div>
        <div><label class="label">Description</label><textarea class="input" data-field="description" rows="3">${e.description||''}</textarea></div>
        <div class="grid grid-cols-2 gap-3">
          <div><label class="label">Status</label><select class="input" data-field="status">
            ${['draft','review','approved','implemented','rejected'].map(s=>`<option ${e.status===s?'selected':''}>${s}</option>`).join('')}
          </select></div>
          <div><label class="label">Priority</label><select class="input" data-field="priority">
            ${['low','normal','high','critical'].map(s=>`<option ${e.priority===s?'selected':''}>${s}</option>`).join('')}
          </select></div>
        </div>
        <div><label class="label">Affected IPNs (comma-separated)</label><input class="input" data-field="affected_ipns" value="${e.affected_ipns||''}"></div>
      </div>`;
    window._ecoCreate = () => {
      showModal('New ECO', formHTML(), async (overlay) => {
        const v = getModalValues(overlay);
        if (!v.title?.trim()) { toast('Title is required', 'error'); return; }
        const btn = overlay.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('POST', 'ecos', v); toast('ECO created'); overlay.remove(); load(); } catch(e) { toast(e.message, 'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
    };
    window._ecoEdit = async (id) => {
      const res = await api('GET', 'ecos/' + id);
      const e = res.data;
      const ncrBadge = e.ncr_id ? `<div class="mb-3"><span class="badge bg-purple-100 text-purple-800 cursor-pointer" onclick="navigate('ncr')">From ${e.ncr_id}</span></div>` : '';
      const overlay = showModal('ECO: ' + e.id + ' — ' + (e.title||'').substring(0,40), ncrBadge + formHTML(e) + `
        <div class="flex gap-2 mt-4">
          ${e.status==='review'||e.status==='draft'?`<button class="btn btn-success" id="eco-approve">✓ Approve</button>`:''}
          ${e.status==='approved'?`<button class="btn btn-primary" id="eco-implement">🚀 Implement</button>`:''}
        </div>`, async (overlay) => {
        const v = getModalValues(overlay);
        if (!v.title?.trim()) { toast('Title is required', 'error'); return; }
        const btn = overlay.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('PUT', 'ecos/' + id, v); toast('ECO updated'); overlay.remove(); load(); } catch(e) { toast(e.message, 'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
      overlay.querySelector('#eco-approve')?.addEventListener('click', async () => {
        await api('POST', 'ecos/' + id + '/approve'); toast('ECO approved'); overlay.remove(); load();
      });
      overlay.querySelector('#eco-implement')?.addEventListener('click', async () => {
        await api('POST', 'ecos/' + id + '/implement'); toast('ECO implemented'); overlay.remove(); load();
      });
    };
    load();
  }
};
