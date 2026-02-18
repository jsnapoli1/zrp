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
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6v12m-3-2.818l.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
          <p class="text-gray-500 font-medium">No quotes yet</p>
          <p class="text-gray-400 text-sm mt-1">Create a quote to start a sales order</p>
          <button class="btn btn-primary mt-4" onclick="window._quoteCreate()">+ New Quote</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Customer</th><th class="pb-2">Status</th><th class="pb-2">Valid Until</th><th class="pb-2">Created</th><th class="pb-2 w-8"></th>
        </tr></thead><tbody>
          ${items.map(q => `<tr class="table-row border-b border-gray-100" onclick="window._quoteEdit('${q.id}')">
            <td class="py-2 font-mono text-blue-600">${q.id}</td><td class="py-2">${q.customer}</td>
            <td class="py-2">${badge(q.status)}</td><td class="py-2">${q.valid_until||''}</td>
            <td class="py-2 text-gray-500">${q.created_at?.substring(0,10)}</td>
            <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
          </tr>`).join('')}
        </tbody></table></div>`}
      </div>`;
    }
    window._quoteCreate = () => showModal('New Quote', `<div class="space-y-3">
      <div><label class="label">Customer</label><input class="input" data-field="customer"></div>
      <div><label class="label">Valid Until</label><input class="input" type="date" data-field="valid_until"></div>
      <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2"></textarea></div>
    </div>`, async (o) => {
      const v = getModalValues(o);
      if (!v.customer?.trim()) { toast('Customer is required', 'error'); return; }
      const btn = o.querySelector('#modal-save');
      btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
      try { await api('POST','quotes',v); toast('Quote created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
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
        <div class="flex gap-2 mt-3">
          <button class="btn btn-secondary" id="quote-tab-lines" onclick="document.getElementById('quote-lines-panel').style.display='block';document.getElementById('quote-margin-panel').style.display='none';">Line Items</button>
          <button class="btn btn-secondary" id="quote-tab-margin" onclick="window._quoteLoadMargin('${id}')">📊 Margin Analysis</button>
          <button class="btn btn-secondary" onclick="window.open('/api/v1/quotes/${id}/pdf','_blank')">🖨️ Print Quote</button>
        </div>
        <div id="quote-lines-panel"></div>
        <div id="quote-margin-panel" style="display:none"><p class="text-gray-400 py-2">Loading margin data...</p></div>
      `);
      window._quoteLoadMargin = async (qid) => {
        document.getElementById('quote-lines-panel').style.display='none';
        const panel = document.getElementById('quote-margin-panel');
        panel.style.display='block';
        try {
          const c = (await api('GET','quotes/'+qid+'/cost')).data;
          const marginColor = (pct) => { if(pct==null) return ''; if(pct>50) return 'text-green-600'; if(pct>=20) return 'text-yellow-600'; return 'text-red-600'; };
          let html = `<table class="w-full text-sm mt-3"><thead><tr class="border-b text-gray-500">
            <th class="pb-1 text-left">IPN</th><th class="pb-1 text-right">Qty</th><th class="pb-1 text-right">Quoted</th><th class="pb-1 text-right">BOM Cost</th><th class="pb-1 text-right">Margin/Unit</th><th class="pb-1 text-right">Margin %</th>
          </tr></thead><tbody>`;
          for (const l of (c.lines||[])) {
            const hasCost = l.bom_cost != null;
            const mc = marginColor(l.margin_pct);
            html += `<tr class="border-b border-gray-100">
              <td class="py-1 font-mono">${l.ipn}</td><td class="py-1 text-right">${l.qty}</td>
              <td class="py-1 text-right">$${l.unit_price_quoted.toFixed(2)}</td>
              <td class="py-1 text-right">${hasCost?'$'+l.bom_cost.toFixed(2):'<span class=\"text-gray-400\">—</span>'}</td>
              <td class="py-1 text-right ${mc}">${hasCost?'$'+l.margin_per_unit.toFixed(2):'—'}</td>
              <td class="py-1 text-right font-medium ${mc}">${hasCost?l.margin_pct.toFixed(1)+'%':'<span class=\"text-xs text-gray-400\">No cost data</span>'}</td>
            </tr>`;
          }
          html += '</tbody></table>';
          html += `<div class="mt-4 pt-3 border-t grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
            <div><span class="text-gray-500">Total Quoted</span><div class="font-bold">$${(c.total_quoted||0).toFixed(2)}</div></div>
            <div><span class="text-gray-500">Total BOM Cost</span><div class="font-bold">${c.total_bom_cost!=null?'$'+c.total_bom_cost.toFixed(2):'—'}</div></div>
            <div><span class="text-gray-500">Total Margin</span><div class="font-bold ${marginColor(c.total_margin_pct)}">${c.total_margin!=null?'$'+c.total_margin.toFixed(2):'—'}</div></div>
            <div><span class="text-gray-500">Margin %</span><div class="font-bold ${marginColor(c.total_margin_pct)}">${c.total_margin_pct!=null?c.total_margin_pct.toFixed(1)+'%':'—'}</div></div>
          </div>`;
          panel.innerHTML = html;
        } catch(e) { panel.innerHTML = '<p class="text-red-500">'+e.message+'</p>'; }
      };
    };
    load();
  }
};
