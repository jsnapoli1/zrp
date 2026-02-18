window.module_rma = {
  render: async (container) => {
    const bulk = setupBulkOps(container, 'rmas/bulk', [
      {action:'close', label:'Close', class:'bg-gray-600 hover:bg-gray-700 text-white'},
      {action:'delete', label:'🗑 Delete', class:'bg-red-600 hover:bg-red-700 text-white'},
    ]);
    async function load() {
      const res = await api('GET', 'rmas');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">RMAs</h2>
          <button class="btn btn-primary" onclick="window._rmaCreate()">+ New RMA</button>
        </div>
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M11.42 15.17l-5.384 3.03A.75.75 0 015.25 17.527V6.473a.75.75 0 01.786-.673l5.384 3.03a.75.75 0 010 1.34zm6.06-3.03l-3.384 1.903a.75.75 0 000 1.34l3.384 1.903"/></svg>
          <p class="text-gray-500 font-medium">No RMAs yet</p>
          <p class="text-gray-400 text-sm mt-1">Create an RMA to track product returns</p>
          <button class="btn btn-primary mt-4" onclick="window._rmaCreate()">+ New RMA</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2 w-8">${bulk.headerCheckbox()}</th>
          <th class="pb-2">ID</th><th class="pb-2">Serial</th><th class="pb-2">Customer</th><th class="pb-2">Reason</th><th class="pb-2">Status</th><th class="pb-2 w-8"></th>
        </tr></thead><tbody>
          ${items.map(r => `<tr class="table-row border-b border-gray-100" onclick="window._rmaEdit('${r.id}')">
            <td class="py-2">${bulk.checkbox(r.id)}</td>
            <td class="py-2 font-mono text-blue-600">${r.id}</td><td class="py-2 font-mono">${r.serial_number}</td>
            <td class="py-2">${r.customer||''}</td><td class="py-2">${r.reason||''}</td><td class="py-2">${badge(r.status)}</td>
            <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
          </tr>`).join('')}
        </tbody></table></div>`}
      </div>`;
      bulk.init();
    }
    container.addEventListener('bulk-reload', load);
    const form = (r={}) => `<div class="space-y-3">
      <div><label class="label">Serial Number</label><input class="input" data-field="serial_number" value="${r.serial_number||''}"></div>
      <div><label class="label">Customer</label><input class="input" data-field="customer" value="${r.customer||''}"></div>
      <div><label class="label">Reason</label><input class="input" data-field="reason" value="${r.reason||''}"></div>
      <div><label class="label">Status</label><select class="input" data-field="status">
        ${['open','received','diagnosing','repaired','shipped','closed'].map(s=>`<option ${r.status===s?'selected':''}>${s}</option>`).join('')}
      </select></div>
      <div><label class="label">Defect Description</label><textarea class="input" data-field="defect_description" rows="2">${r.defect_description||''}</textarea></div>
      <div><label class="label">Resolution</label><textarea class="input" data-field="resolution" rows="2">${r.resolution||''}</textarea></div>
    </div>`;
    window._rmaCreate = () => showModal('New RMA', form(), async (o) => {
      const v = getModalValues(o);
      if (!v.serial_number?.trim()) { toast('Serial number is required', 'error'); return; }
      const btn = o.querySelector('#modal-save');
      btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
      try { await api('POST','rmas',v); toast('RMA created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
    });
    window._rmaEdit = async (id) => {
      const r = (await api('GET','rmas/'+id)).data;
      const o = showModal('RMA: '+r.id+' — '+(r.serial_number||''), form(r) + attachmentsSection('rma', id), async (o) => {
        const v = getModalValues(o);
        const btn = o.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('PUT','rmas/'+id,v); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
      initAttachments(o, 'rma', id);
    };
    load();
  }
};
