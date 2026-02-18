window.module_procurement = {
  render: async (container) => {
    async function load() {
      const res = await api('GET', 'pos');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Purchase Orders</h2>
          <div class="flex gap-2">
            <button class="btn btn-secondary" onclick="window._poFromWO()">🔄 Generate from WO Shortage</button>
            <button class="btn btn-primary" onclick="window._poCreate()">+ New PO</button>
          </div>
        </div>
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 100 4 2 2 0 000-4z"/></svg>
          <p class="text-gray-500 font-medium">No purchase orders yet</p>
          <p class="text-gray-400 text-sm mt-1">Create your first PO to get started</p>
          <button class="btn btn-primary mt-4" onclick="window._poCreate()">+ New PO</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">PO #</th><th class="pb-2">Vendor</th><th class="pb-2">Status</th><th class="pb-2">Expected</th><th class="pb-2">Created</th><th class="pb-2 w-8"></th>
        </tr></thead><tbody>
          ${items.map(p => `<tr class="table-row border-b border-gray-100" onclick="window._poEdit('${p.id}')">
            <td class="py-2 font-mono text-blue-600">${p.id}</td>
            <td class="py-2">${p.vendor_id||''}</td>
            <td class="py-2">${badge(p.status)}</td>
            <td class="py-2">${p.expected_date||''}</td>
            <td class="py-2 text-gray-500">${p.created_at?.substring(0,10)}</td>
            <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
          </tr>`).join('')}
        </tbody></table>
        </div>`}
      </div>`;
    }

    window._poFromWO = async () => {
      const [woRes, vRes] = await Promise.all([api('GET', 'workorders'), api('GET', 'vendors')]);
      const wos = woRes.data || [];
      const vendors = vRes.data || [];
      showModal('Generate PO from WO Shortages', `<div class="space-y-3">
        <div><label class="label">Work Order</label><select class="input" data-field="wo_id">
          <option value="">Select work order...</option>
          ${wos.map(w => `<option value="${w.id}">${w.id} — ${w.assembly_ipn} (×${w.qty}) [${w.status}]</option>`).join('')}
        </select></div>
        <div><label class="label">Vendor (optional)</label><select class="input" data-field="vendor_id">
          <option value="">No vendor</option>
          ${vendors.map(v => `<option value="${v.id}">${v.name}</option>`).join('')}
        </select></div>
      </div>`, async (o) => {
        const v = getModalValues(o);
        if (!v.wo_id) { toast('Select a work order', 'error'); return; }
        try {
          const res = await api('POST', 'pos/generate-from-wo', { wo_id: v.wo_id, vendor_id: v.vendor_id });
          const poId = res.data?.po_id;
          toast('PO ' + poId + ' created with ' + res.data?.lines + ' lines');
          o.remove();
          load();
          if (poId) setTimeout(() => window._poEdit(poId), 300);
        } catch(e) { toast(e.message, 'error'); }
      });
    };

    window._poCreate = async () => {
      const vRes = await api('GET', 'vendors');
      const vendors = vRes.data || [];
      showModal('New Purchase Order', `<div class="space-y-3">
        <div><label class="label">Vendor</label><select class="input" data-field="vendor_id">
          <option value="">Select vendor...</option>
          ${vendors.map(v => `<option value="${v.id}">${v.name}</option>`).join('')}
        </select></div>
        <div><label class="label">Expected Date</label><input class="input" type="date" data-field="expected_date"></div>
        <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2"></textarea></div>
      </div>`, async (o) => {
        const v = getModalValues(o);
        if (!v.vendor_id) { toast('Please select a vendor', 'error'); return; }
        const btn = o.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('POST', 'pos', v); toast('PO created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
    };
    window._poEdit = async (id) => {
      const res = await api('GET', 'pos/' + id);
      const p = res.data;
      showModal('PO: ' + id, `
        <div class="space-y-2 mb-4">
          <div class="flex justify-between"><span class="text-gray-500">Status</span>${badge(p.status)}</div>
          <div class="flex justify-between"><span class="text-gray-500">Vendor</span><span>${p.vendor_id}</span></div>
          <div class="flex justify-between"><span class="text-gray-500">Expected</span><span>${p.expected_date||'N/A'}</span></div>
        </div>
        <h3 class="font-semibold mb-2">Lines</h3>
        <table class="w-full text-sm"><thead><tr class="border-b text-gray-500">
          <th class="pb-1 text-left">IPN</th><th class="pb-1 text-left">MPN</th><th class="pb-1">Ordered</th><th class="pb-1">Received</th><th class="pb-1">Price</th>
        </tr></thead><tbody>
          ${(p.lines||[]).map(l => `<tr class="border-b border-gray-100">
            <td class="py-1 font-mono">${l.ipn}</td><td class="py-1">${l.mpn||''}</td>
            <td class="py-1 text-center">${l.qty_ordered}</td><td class="py-1 text-center">${l.qty_received}</td>
            <td class="py-1 text-right">$${(l.unit_price||0).toFixed(2)}</td>
          </tr>`).join('')}
        </tbody></table>
        ${p.status!=='received'&&p.status!=='cancelled'?`<button class="btn btn-success mt-3" id="po-receive">📥 Receive All</button>`:''}
      `);
      document.getElementById('po-receive')?.addEventListener('click', async () => {
        const lines = (p.lines||[]).map(l => ({id:l.id, qty:l.qty_ordered-l.qty_received})).filter(l=>l.qty>0);
        if(lines.length) { await api('POST','pos/'+id+'/receive',{lines}); toast('Items received'); document.querySelector('.modal-overlay')?.remove(); load(); }
      });
    };
    load();
  }
};
