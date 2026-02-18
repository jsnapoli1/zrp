window.module_quotes = {
  render: async (container) => {
    async function load() {
      const res = await api('GET', 'quotes');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Quotes & Sales Orders</h2>
          <button class="btn btn-primary" onclick="window._quoteCreate()">+ New Quote</button>
        </div>
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Customer</th><th class="pb-2">Status</th><th class="pb-2">Valid Until</th><th class="pb-2">Created</th>
        </tr></thead><tbody>
          ${items.map(q => `<tr class="table-row border-b border-gray-100" onclick="window._quoteEdit('${q.id}')">
            <td class="py-2 font-mono text-blue-600">${q.id}</td><td class="py-2">${q.customer}</td>
            <td class="py-2">${badge(q.status)}</td><td class="py-2">${q.valid_until||''}</td>
            <td class="py-2 text-gray-500">${q.created_at?.substring(0,10)}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No quotes</p>':''}
      </div>`;
    }
    window._quoteCreate = () => showModal('New Quote', `<div class="space-y-3">
      <div><label class="label">Customer</label><input class="input" data-field="customer"></div>
      <div><label class="label">Valid Until</label><input class="input" type="date" data-field="valid_until"></div>
      <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2"></textarea></div>
    </div>`, async (o) => {
      try { await api('POST','quotes',getModalValues(o)); toast('Quote created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    window._quoteEdit = async (id) => {
      const q = (await api('GET','quotes/'+id)).data;
      const lines = q.lines||[];
      showModal('Quote: '+id, `
        <div class="space-y-2 mb-4">
          <div class="flex justify-between"><span class="text-gray-500">Customer</span><span>${q.customer}</span></div>
          <div class="flex justify-between"><span class="text-gray-500">Status</span>${badge(q.status)}</div>
          <div class="flex justify-between"><span class="text-gray-500">Valid Until</span><span>${q.valid_until||'N/A'}</span></div>
        </div>
        <h3 class="font-semibold mb-2">Line Items</h3>
        <table class="w-full text-sm"><thead><tr class="border-b text-gray-500">
          <th class="pb-1 text-left">IPN</th><th class="pb-1 text-left">Description</th><th class="pb-1">Qty</th><th class="pb-1">Unit Price</th><th class="pb-1">Total</th>
        </tr></thead><tbody>
          ${lines.map(l=>`<tr class="border-b border-gray-100">
            <td class="py-1 font-mono">${l.ipn}</td><td class="py-1">${l.description||''}</td>
            <td class="py-1 text-center">${l.qty}</td><td class="py-1 text-right">$${(l.unit_price||0).toFixed(2)}</td>
            <td class="py-1 text-right font-medium">$${(l.qty*(l.unit_price||0)).toFixed(2)}</td>
          </tr>`).join('')}
          ${lines.length?`<tr class="font-bold"><td colspan="4" class="py-2 text-right">Total:</td><td class="py-2 text-right">$${lines.reduce((s,l)=>s+l.qty*(l.unit_price||0),0).toFixed(2)}</td></tr>`:''}
        </tbody></table>
        <div class="flex gap-2 mt-3"><button class="btn btn-secondary" id="quote-cost">💰 View Cost Rollup</button><button class="btn btn-secondary" onclick="window.open('/api/v1/quotes/${id}/pdf','_blank')">🖨️ Print Quote</button></div>
      `);
      document.getElementById('quote-cost')?.addEventListener('click', async () => {
        const c = (await api('GET','quotes/'+id+'/cost')).data;
        showModal('Cost Rollup: '+id, `<table class="w-full text-sm"><thead><tr class="border-b text-gray-500">
          <th class="pb-1 text-left">IPN</th><th class="pb-1">Qty</th><th class="pb-1">Unit</th><th class="pb-1">Total</th>
        </tr></thead><tbody>
          ${(c.lines||[]).map(l=>`<tr class="border-b border-gray-100"><td class="py-1">${l.ipn}</td><td class="py-1 text-center">${l.qty}</td><td class="py-1 text-right">$${l.unit_price.toFixed(2)}</td><td class="py-1 text-right">$${l.line_total.toFixed(2)}</td></tr>`).join('')}
          <tr class="font-bold"><td colspan="3" class="py-2 text-right">Total:</td><td class="py-2 text-right">$${(c.total||0).toFixed(2)}</td></tr>
        </tbody></table>`);
      });
    };
    load();
  }
};
