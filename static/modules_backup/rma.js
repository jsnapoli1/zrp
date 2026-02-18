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
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2 w-8">${bulk.headerCheckbox()}</th>
          <th class="pb-2">ID</th><th class="pb-2">Serial</th><th class="pb-2">Customer</th><th class="pb-2">Reason</th><th class="pb-2">Status</th>
        </tr></thead><tbody>
          ${items.map(r => `<tr class="table-row border-b border-gray-100" onclick="window._rmaEdit('${r.id}')">
            <td class="py-2">${bulk.checkbox(r.id)}</td>
            <td class="py-2 font-mono text-blue-600">${r.id}</td><td class="py-2 font-mono">${r.serial_number}</td>
            <td class="py-2">${r.customer||''}</td><td class="py-2">${r.reason||''}</td><td class="py-2">${badge(r.status)}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No RMAs</p>':''}
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
      try { await api('POST','rmas',getModalValues(o)); toast('RMA created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    window._rmaEdit = async (id) => {
      const r = (await api('GET','rmas/'+id)).data;
      const o = showModal('RMA: '+id, form(r) + attachmentsSection('rma', id), async (o) => {
        try { await api('PUT','rmas/'+id,getModalValues(o)); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
      });
      initAttachments(o, 'rma', id);
    };
    load();
  }
};
