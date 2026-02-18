window.module_workorders = {
  render: async (container) => {
    const bulk = setupBulkOps(container, 'workorders/bulk', [
      {action:'complete', label:'✓ Complete', class:'bg-green-600 hover:bg-green-700 text-white'},
      {action:'cancel', label:'✗ Cancel', class:'bg-yellow-600 hover:bg-yellow-700 text-white'},
      {action:'delete', label:'🗑 Delete', class:'bg-red-600 hover:bg-red-700 text-white'},
    ]);
    async function load() {
      const res = await api('GET', 'workorders');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Work Orders</h2>
          <button class="btn btn-primary" onclick="window._woCreate()">+ New Work Order</button>
        </div>
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2 w-8">${bulk.headerCheckbox()}</th>
          <th class="pb-2">WO #</th><th class="pb-2">Assembly</th><th class="pb-2">Qty</th><th class="pb-2">Status</th><th class="pb-2">Priority</th><th class="pb-2">Created</th>
        </tr></thead><tbody>
          ${items.map(w => `<tr class="table-row border-b border-gray-100" onclick="window._woEdit('${w.id}')">
            <td class="py-2">${bulk.checkbox(w.id)}</td>
            <td class="py-2 font-mono text-blue-600">${w.id}</td><td class="py-2">${w.assembly_ipn}</td>
            <td class="py-2">${w.qty}</td><td class="py-2">${badge(w.status)}</td>
            <td class="py-2">${badge(w.priority)}</td><td class="py-2 text-gray-500">${w.created_at?.substring(0,10)}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No work orders</p>':''}
      </div>`;
      bulk.init();
    }
    container.addEventListener('bulk-reload', load);
    const form = (w={}) => `<div class="space-y-3">
      <div><label class="label">Assembly IPN</label><input class="input" data-field="assembly_ipn" value="${w.assembly_ipn||''}"></div>
      <div class="grid grid-cols-3 gap-3">
        <div><label class="label">Quantity</label><input class="input" type="number" data-field="qty" value="${w.qty||1}"></div>
        <div><label class="label">Status</label><select class="input" data-field="status">
          ${['open','in_progress','completed','cancelled'].map(s=>`<option ${w.status===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
        <div><label class="label">Priority</label><select class="input" data-field="priority">
          ${['low','normal','high','critical'].map(s=>`<option ${w.priority===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
      </div>
      <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2">${w.notes||''}</textarea></div>
    </div>`;
    window._woCreate = () => showModal('New Work Order', form(), async (o) => {
      const v = getModalValues(o); v.qty = parseInt(v.qty)||1;
      try { await api('POST','workorders',v); toast('WO created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    window._woEdit = async (id) => {
      const w = (await api('GET','workorders/'+id)).data;
      const overlay = showModal('WO: '+id, form(w)+`<div class="flex gap-2 mt-3"><button class="btn btn-secondary" id="wo-bom">📋 View BOM</button><button class="btn btn-secondary" onclick="window.open('/api/v1/workorders/${id}/pdf','_blank')">🖨️ Print Traveler</button></div>` + attachmentsSection('workorder', id), async (o) => {
        const v = getModalValues(o); v.qty = parseInt(v.qty)||1;
        try { await api('PUT','workorders/'+id,v); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
      });
      initAttachments(overlay, 'workorder', id);
      document.getElementById('wo-bom')?.addEventListener('click', async () => {
        const bom = (await api('GET','workorders/'+id+'/bom')).data;
        const lines = bom.bom||[];
        const rowColor = (s) => s==='ok'?'bg-green-50':s==='low'?'bg-yellow-50':'bg-red-50';
        const statusIcon = (s) => s==='ok'?'✅':s==='low'?'⚠️':'❌';
        const hasShortages = lines.some(l => l.shortage > 0);
        showModal('BOM for '+bom.assembly_ipn+' (×'+bom.qty+')', `<table class="w-full text-sm"><thead><tr class="border-b text-gray-500">
          <th class="pb-1 text-left">IPN</th><th class="pb-1 text-left">Description</th><th class="pb-1 text-right">Required</th><th class="pb-1 text-right">On Hand</th><th class="pb-1 text-right">Shortage</th><th class="pb-1 text-center">Status</th>
        </tr></thead><tbody>
          ${lines.map(l=>`<tr class="border-b border-gray-100 ${rowColor(l.status)}">
            <td class="py-1 font-mono">${l.ipn}</td><td class="py-1 text-gray-600">${l.description||''}</td>
            <td class="py-1 text-right">${l.qty_required}</td><td class="py-1 text-right">${l.qty_on_hand}</td>
            <td class="py-1 text-right font-medium ${l.shortage>0?'text-red-600':''}">${l.shortage}</td>
            <td class="py-1 text-center">${statusIcon(l.status)} ${l.status}</td>
          </tr>`).join('')}
        </tbody></table>${lines.length===0?'<p class="text-gray-400 text-center py-2">No BOM data</p>':''}
        ${hasShortages?'<button class="btn btn-danger mt-3" id="wo-create-po">🛒 Create PO for Shortages</button>':''}`);
        document.getElementById('wo-create-po')?.addEventListener('click', async () => {
          try {
            const res = await api('POST', 'pos/generate-from-wo', { wo_id: id });
            toast('PO ' + res.data?.po_id + ' created with ' + res.data?.lines + ' lines');
            document.querySelector('.modal-overlay')?.remove();
          } catch(e) { toast(e.message, 'error'); }
        });
      });
    };
    load();
  }
};
