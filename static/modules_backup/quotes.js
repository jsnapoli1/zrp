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
