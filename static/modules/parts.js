window.module_parts = {
  render: async (container) => {
    let currentCat = '';
    let searchQ = '';
    let page = 1;

    async function load() {
      const params = new URLSearchParams({ page, limit: 50 });
      if (currentCat) params.set('category', currentCat);
      if (searchQ) params.set('q', searchQ);
      const res = await api('GET', 'parts?' + params);
      const parts = res.data || [];
      const meta = res.meta || {};
      
      // Load categories
      const catRes = await api('GET', 'categories');
      const cats = catRes.data || [];

      container.innerHTML = `
        <div class="card mb-4">
          <div class="flex items-center justify-between mb-4">
            <div class="flex gap-2 flex-wrap">
              <button class="btn ${!currentCat ? 'btn-primary' : 'btn-secondary'}" onclick="window._partsSetCat('')">All</button>
              ${cats.map(c => `<button class="btn ${currentCat === c.id ? 'btn-primary' : 'btn-secondary'}" onclick="window._partsSetCat('${c.id}')">${c.name} (${c.count})</button>`).join('')}
            </div>
            <div class="flex gap-2">
              <input type="text" class="input w-48" placeholder="Search parts..." value="${searchQ}" onkeyup="if(event.key==='Enter'){window._partsSearch(this.value)}">
              <button class="btn btn-secondary" onclick="window._partsExport()">📥 CSV</button>
            </div>
          </div>
          ${parts.length === 0 ? `<div class="text-center py-12">
            <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
            <p class="text-gray-500 font-medium">No parts found</p>
            <p class="text-gray-400 text-sm mt-1">Configure --pmDir to load gitplm CSVs</p>
          </div>` : `
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead><tr class="border-b text-left text-gray-500">
                <th class="pb-2 pr-4">IPN</th>
                <th class="pb-2 pr-4">Category</th>
                ${parts[0] && parts[0].fields ? Object.keys(parts[0].fields).filter(k=>k!=='_category'&&k!=='ipn'&&k!=='IPN'&&k!=='pn'&&k!=='PN').slice(0,5).map(k=>`<th class="pb-2 pr-4">${k}</th>`).join('') : ''}
              </tr></thead>
              <tbody>
                ${parts.map(p => `<tr class="table-row border-b border-gray-100" onclick="window._partsDetail('${p.ipn}')">
                  <td class="py-2 pr-4 font-mono text-blue-600">${p.ipn} <a href="${window._gitplmURL||'http://localhost:8888'}/#/parts?ipn=${encodeURIComponent(p.ipn)}" target="_blank" onclick="event.stopPropagation()" class="text-gray-400 hover:text-blue-500" title="Open in gitplm-ui">↗</a></td>
                  <td class="py-2 pr-4">${p.fields?._category || ''}</td>
                  ${p.fields ? Object.entries(p.fields).filter(([k])=>k!=='_category'&&k!=='ipn'&&k!=='IPN'&&k!=='pn'&&k!=='PN').slice(0,5).map(([k,v])=>`<td class="py-2 pr-4 truncate max-w-[200px]">${v}</td>`).join('') : ''}
                </tr>`).join('')}
              </tbody>
            </table>
          </div>
          <div class="flex justify-between items-center mt-4 text-sm text-gray-500">
            <span>Showing ${parts.length} of ${meta.total || parts.length}</span>
            <div class="flex gap-2">
              ${page > 1 ? `<button class="btn btn-secondary" onclick="window._partsPage(${page-1})">← Prev</button>` : ''}
              ${(meta.total || 0) > page * 50 ? `<button class="btn btn-secondary" onclick="window._partsPage(${page+1})">Next →</button>` : ''}
            </div>
          </div>`}
        </div>
      `;
    }

    window._partsSetCat = (c) => { currentCat = c; page = 1; load(); };
    window._partsSearch = (q) => { searchQ = q; page = 1; load(); };
    window._partsPage = (p) => { page = p; load(); };
    window._partsDetail = async (ipn) => {
      const res = await api('GET', 'parts/' + ipn);
      const p = res.data;
      const fields = Object.entries(p.fields || {}).map(([k,v]) => `<div class="mb-2"><span class="label">${k}</span><div class="text-sm">${v}</div></div>`).join('');
      const isAssembly = ipn.toUpperCase().startsWith('PCA-') || ipn.toUpperCase().startsWith('ASY-');

      // Cost section
      let costHTML = '<div id="parts-cost-panel" class="mt-4 border-t pt-4"><p class="text-gray-400 text-sm">Loading cost...</p></div>';

      // Pricing tab
      let pricingTabBtn = `<button class="btn btn-secondary btn-sm" onclick="window._partsLoadPricing('${ipn}')">💲 Pricing</button>`;
      let supplierPriceTabBtn = `<button class="btn btn-secondary btn-sm" onclick="window._partsLoadSupplierPrices('${ipn}')">📊 Price Quotes</button>`;

      let bomHTML = '';
      if (isAssembly) {
        bomHTML = `<div class="mt-4 border-t pt-4">
          <div class="flex gap-2 mb-3">
            <button class="btn btn-secondary btn-sm" id="parts-tab-details" onclick="document.getElementById('parts-details-panel').style.display='block';document.getElementById('parts-bom-panel').style.display='none';document.getElementById('parts-pricing-panel')&&(document.getElementById('parts-pricing-panel').style.display='none');document.getElementById('parts-supplier-prices-panel')&&(document.getElementById('parts-supplier-prices-panel').style.display='none');">Details</button>
            <button class="btn btn-secondary btn-sm" id="parts-tab-bom" onclick="window._partsLoadBOM('${ipn}')">📋 BOM</button>
            ${pricingTabBtn}
            ${supplierPriceTabBtn}
          </div>
          <div id="parts-details-panel">${fields}</div>
          <div id="parts-bom-panel" style="display:none"><p class="text-gray-400 text-center py-2">Loading BOM...</p></div>
          <div id="parts-pricing-panel" style="display:none"><p class="text-gray-400 text-center py-2">Loading pricing...</p></div>
          <div id="parts-supplier-prices-panel" style="display:none"><p class="text-gray-400 text-center py-2">Loading supplier prices...</p></div>
        </div>`;
      } else {
        bomHTML = `<div class="mt-4 border-t pt-4">
          <div class="flex gap-2 mb-3">
            ${pricingTabBtn}
            ${supplierPriceTabBtn}
          </div>
          <div id="parts-details-panel">${fields}</div>
          <div id="parts-pricing-panel" style="display:none"><p class="text-gray-400 text-center py-2">Loading pricing...</p></div>
          <div id="parts-supplier-prices-panel" style="display:none"><p class="text-gray-400 text-center py-2">Loading supplier prices...</p></div>
        </div>`;
      }
      const gitplmLink = `<a href="${window._gitplmURL||'http://localhost:8888'}/#/parts?ipn=${encodeURIComponent(ipn)}" target="_blank" class="text-sm text-blue-500 hover:underline ml-2">Open in gitplm-ui →</a>`;
      showModal('Part: ' + ipn, `<div class="font-mono text-lg text-blue-600 mb-4">${ipn}${gitplmLink}</div>${bomHTML}${costHTML}`);

      // Load cost asynchronously
      try {
        const costRes = await api('GET', 'parts/' + encodeURIComponent(ipn) + '/cost');
        const cost = costRes.data;
        const panel = document.getElementById('parts-cost-panel');
        if (panel) {
          let html = '<h3 class="font-semibold text-sm mb-2">💰 Cost</h3>';
          if (cost.last_unit_price !== undefined) {
            html += `<div class="text-sm"><span class="text-gray-500">Last Unit Price:</span> <span class="font-semibold">$${Number(cost.last_unit_price).toFixed(4)}</span></div>`;
            html += `<div class="text-sm text-gray-500">PO: ${cost.po_id||'—'} · ${cost.last_ordered||'—'}</div>`;
          } else {
            html += '<p class="text-sm text-gray-400">No purchase history</p>';
          }
          if (cost.bom_cost !== undefined) {
            html += `<div class="text-sm mt-2"><span class="text-gray-500">BOM Cost Estimate:</span> <span class="font-semibold">$${Number(cost.bom_cost).toFixed(4)}</span></div>`;
          }
          panel.innerHTML = html;
        }
      } catch(e) {
        const panel = document.getElementById('parts-cost-panel');
        if (panel) panel.innerHTML = '<p class="text-sm text-gray-400">Cost data unavailable</p>';
      }
    };
    window._partsLoadBOM = async (ipn) => {
      document.getElementById('parts-details-panel').style.display = 'none';
      const panel = document.getElementById('parts-bom-panel');
      panel.style.display = 'block';
      try {
        const res = await api('GET', 'parts/' + encodeURIComponent(ipn) + '/bom');
        const tree = res.data;
        if (!tree || !tree.children || tree.children.length === 0) {
          panel.innerHTML = '<p class="text-gray-400 text-center py-4">No BOM data found. BOM CSV file not available for this assembly.</p>';
          return;
        }
        let uniqueParts = new Set();
        let totalLines = 0;
        function renderTree(node, depth) {
          let html = '';
          for (const child of (node.children || [])) {
            totalLines++;
            uniqueParts.add(child.ipn);
            const indent = depth * 24;
            const isAsm = child.ipn.toUpperCase().startsWith('PCA-') || child.ipn.toUpperCase().startsWith('ASY-');
            html += `<tr class="border-b border-gray-100 ${isAsm?'bg-blue-50':''}">
              <td class="py-1" style="padding-left:${indent}px">
                <span class="font-mono text-blue-600 cursor-pointer hover:underline" onclick="window._partsDetail('${child.ipn}')">${isAsm?'📦 ':''}${child.ipn}</span>
              </td>
              <td class="py-1 text-gray-600">${child.description||''}</td>
              <td class="py-1 text-center">${child.qty||''}</td>
              <td class="py-1 text-gray-500 text-xs">${child.ref||''}</td>
            </tr>`;
            if (child.children && child.children.length > 0) {
              html += renderTree(child, depth + 1);
            }
          }
          return html;
        }
        const rows = renderTree(tree, 0);
        panel.innerHTML = `<table class="w-full text-sm"><thead><tr class="border-b text-gray-500">
          <th class="pb-1 text-left">IPN</th><th class="pb-1 text-left">Description</th><th class="pb-1 text-center">Qty</th><th class="pb-1 text-left">Ref Des</th>
        </tr></thead><tbody>${rows}</tbody></table>
        <div class="flex gap-4 mt-3 text-sm text-gray-500">
          <span>📊 ${totalLines} total lines</span>
          <span>🔧 ${uniqueParts.size} unique parts</span>
        </div>`;
      } catch(e) {
        panel.innerHTML = '<p class="text-red-500 text-center py-4">' + (e.message||'Failed to load BOM') + '</p>';
      }
    };
    window._partsLoadSupplierPrices = async (ipn) => {
      if (document.getElementById('parts-details-panel')) document.getElementById('parts-details-panel').style.display = 'none';
      if (document.getElementById('parts-bom-panel')) document.getElementById('parts-bom-panel').style.display = 'none';
      if (document.getElementById('parts-pricing-panel')) document.getElementById('parts-pricing-panel').style.display = 'none';
      const panel = document.getElementById('parts-supplier-prices-panel');
      if (!panel) return;
      panel.style.display = 'block';
      panel.innerHTML = '<p class="text-gray-400 text-center py-2">Loading supplier prices...</p>';
      try {
        const [pricesRes, trendRes] = await Promise.all([
          api('GET', 'supplier-prices?ipn=' + encodeURIComponent(ipn) + '&limit=500'),
          api('GET', 'supplier-prices/trend?ipn=' + encodeURIComponent(ipn))
        ]);
        const prices = pricesRes.data || [];
        const trend = trendRes.data || [];
        let html = '';
        // Chart
        if (trend.length > 1) {
          const vendors = [...new Set(trend.map(t => t.vendor))];
          const colors = ['#3b82f6','#ef4444','#10b981','#f59e0b','#8b5cf6','#ec4899'];
          const w=500,h=200,pad=35;
          const allPrices = trend.map(t=>t.price);
          const minP=Math.min(...allPrices)*0.95, maxP=Math.max(...allPrices)*1.05, range=maxP-minP||1;
          const allDates=[...new Set(trend.map(t=>t.date))].sort();
          let paths='',dots='';
          vendors.forEach((v,vi)=>{
            const c=colors[vi%colors.length];
            const pts=trend.filter(t=>t.vendor===v).map(t=>({
              x:pad+((allDates.indexOf(t.date))/Math.max(allDates.length-1,1))*(w-2*pad),
              y:h-pad-((t.price-minP)/range)*(h-2*pad),t
            }));
            if(pts.length>1) paths+=`<polyline points="${pts.map(p=>p.x+','+p.y).join(' ')}" fill="none" stroke="${c}" stroke-width="2"/>`;
            pts.forEach(p=>{dots+=`<circle cx="${p.x}" cy="${p.y}" r="3" fill="${c}"><title>$${p.t.price} — ${p.t.vendor} (${p.t.date})</title></circle>`;});
          });
          const legend=vendors.map((v,i)=>`<span class="inline-flex items-center gap-1 mr-3"><span class="w-3 h-3 rounded-full inline-block" style="background:${colors[i%colors.length]}"></span>${v}</span>`).join('');
          html+=`<div class="mb-3"><svg viewBox="0 0 ${w} ${h}" class="w-full" style="max-height:200px"><line x1="${pad}" y1="${h-pad}" x2="${w-pad}" y2="${h-pad}" stroke="#e5e7eb"/><line x1="${pad}" y1="${pad}" x2="${pad}" y2="${h-pad}" stroke="#e5e7eb"/>${paths}${dots}</svg><div class="text-xs mt-1">${legend}</div></div>`;
        }
        // Table
        let bestPrice=Infinity;
        prices.forEach(p=>{if(p.unit_price<bestPrice)bestPrice=p.unit_price;});
        html+=`<div class="flex justify-between items-center mb-2"><h3 class="text-sm font-semibold">Supplier Price Quotes</h3></div>`;
        if(prices.length===0){html+='<p class="text-gray-400 text-sm">No supplier price quotes</p>';}
        else{
          html+=`<table class="w-full text-sm"><thead><tr class="border-b text-gray-500 text-left"><th class="pb-1">Vendor</th><th class="pb-1">Price</th><th class="pb-1">Qty Break</th><th class="pb-1">Lead Time</th><th class="pb-1">Date</th></tr></thead><tbody>`;
          prices.forEach(p=>{
            const isBest=p.unit_price===bestPrice;
            html+=`<tr class="border-b border-gray-100"><td class="py-1">${p.vendor_name}</td><td class="py-1 font-mono ${isBest?'text-green-600 font-bold':''}">$${Number(p.unit_price).toFixed(4)}</td><td class="py-1">${p.quantity_break}</td><td class="py-1">${p.lead_time_days!=null?p.lead_time_days+'d':'—'}</td><td class="py-1">${p.quote_date||''}</td></tr>`;
          });
          html+='</tbody></table>';
        }
        panel.innerHTML=html;
      } catch(e) { panel.innerHTML = '<p class="text-red-500 text-sm">Error loading supplier prices</p>'; }
    };

    window._partsLoadPricing = async (ipn) => {
      if (document.getElementById('parts-details-panel')) document.getElementById('parts-details-panel').style.display = 'none';
      if (document.getElementById('parts-bom-panel')) document.getElementById('parts-bom-panel').style.display = 'none';
      if (document.getElementById('parts-supplier-prices-panel')) document.getElementById('parts-supplier-prices-panel').style.display = 'none';
      const panel = document.getElementById('parts-pricing-panel');
      if (!panel) return;
      panel.style.display = 'block';
      panel.innerHTML = '<p class="text-gray-400 text-center py-2">Loading pricing...</p>';
      try {
        const res = await api('GET', 'prices/' + encodeURIComponent(ipn));
        const prices = res.data || [];
        const trendRes = await api('GET', 'prices/' + encodeURIComponent(ipn) + '/trend');
        const trend = trendRes.data || [];

        // Sparkline SVG
        let sparkline = '';
        if (trend.length > 1) {
          const w = 300, h = 60, pad = 5;
          const ps = trend.map(t => t.price);
          const minP = Math.min(...ps), maxP = Math.max(...ps);
          const range = maxP - minP || 1;
          const points = trend.map((t, i) => {
            const x = pad + (i / (trend.length - 1)) * (w - 2 * pad);
            const y = h - pad - ((t.price - minP) / range) * (h - 2 * pad);
            return `${x},${y}`;
          }).join(' ');
          sparkline = `<div class="mb-3"><svg width="${w}" height="${h}" class="border rounded">
            <polyline points="${points}" fill="none" stroke="#3b82f6" stroke-width="2"/>
            ${trend.map((t, i) => {
              const x = pad + (i / (trend.length - 1)) * (w - 2 * pad);
              const y = h - pad - ((t.price - minP) / range) * (h - 2 * pad);
              return `<circle cx="${x}" cy="${y}" r="3" fill="#3b82f6"><title>$${t.price} — ${t.vendor} (${t.date})</title></circle>`;
            }).join('')}
          </svg><div class="text-xs text-gray-400 mt-1">Price trend over time</div></div>`;
        }

        // Find best price
        let bestIdx = -1;
        if (prices.length > 0) {
          let minPrice = Infinity;
          prices.forEach((p, i) => { if (p.unit_price < minPrice) { minPrice = p.unit_price; bestIdx = i; } });
        }

        let html = sparkline;
        html += `<div class="flex justify-between items-center mb-2">
          <h3 class="text-sm font-semibold">Price History</h3>
          <button class="btn btn-secondary btn-sm" onclick="window._partsAddPrice('${ipn}')">+ Add Price</button>
        </div>`;
        if (prices.length === 0) {
          html += '<p class="text-gray-400 text-sm">No price history</p>';
        } else {
          html += `<table class="w-full text-sm"><thead><tr class="border-b text-gray-500 text-left">
            <th class="pb-1">Vendor</th><th class="pb-1">Unit Price</th><th class="pb-1">Min Qty</th><th class="pb-1">Lead Time</th><th class="pb-1">Date</th>
          </tr></thead><tbody>`;
          prices.forEach((p, i) => {
            const isBest = i === bestIdx;
            html += `<tr class="border-b border-gray-100 ${isBest ? 'bg-green-50' : ''}">
              <td class="py-1">${p.vendor_name || p.vendor_id || '—'}${isBest ? ' <span class="badge bg-green-100 text-green-700">Best</span>' : ''}</td>
              <td class="py-1 font-mono">$${Number(p.unit_price).toFixed(4)}</td>
              <td class="py-1">${p.min_qty || 1}</td>
              <td class="py-1">${p.lead_time_days ? p.lead_time_days + 'd' : '—'}</td>
              <td class="py-1 text-gray-400 text-xs">${p.recorded_at ? new Date(p.recorded_at).toLocaleDateString() : '—'}</td>
            </tr>`;
          });
          html += '</tbody></table>';
        }
        panel.innerHTML = html;
      } catch(e) {
        panel.innerHTML = '<p class="text-red-500 text-sm">' + (e.message || 'Failed to load pricing') + '</p>';
      }
    };

    window._partsAddPrice = async (ipn) => {
      // Load vendors for dropdown
      let vendors = [];
      try {
        const vRes = await api('GET', 'vendors');
        vendors = vRes.data || [];
      } catch(e) {}

      const vendorOpts = vendors.map(v => `<option value="${v.id}">${v.name}</option>`).join('');
      showModal('Add Price Entry', `
        <div class="space-y-3">
          <div><label class="label">IPN</label><input class="input" data-field="ipn" value="${ipn}" readonly></div>
          <div><label class="label">Vendor</label><select class="input" data-field="vendor_id"><option value="">— Select —</option>${vendorOpts}</select></div>
          <div><label class="label">Unit Price ($)</label><input type="number" step="0.0001" class="input" data-field="unit_price" placeholder="0.0000"></div>
          <div><label class="label">Min Qty</label><input type="number" class="input" data-field="min_qty" value="1"></div>
          <div><label class="label">Lead Time (days)</label><input type="number" class="input" data-field="lead_time_days" placeholder=""></div>
        </div>
      `, async (modal) => {
        const vals = getModalValues(modal);
        if (!vals.unit_price || parseFloat(vals.unit_price) <= 0) { toast('Unit price is required', 'error'); return; }
        const btn = modal.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try {
          await api('POST', 'prices', {
            ipn: vals.ipn,
            vendor_id: vals.vendor_id || null,
            unit_price: parseFloat(vals.unit_price),
            min_qty: parseInt(vals.min_qty) || 1,
            lead_time_days: vals.lead_time_days ? parseInt(vals.lead_time_days) : null,
          });
          toast('Price entry added');
          modal.remove();
          window._partsLoadPricing(ipn);
        } catch(e) { toast(e.message, 'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
    };

    window._partsExport = () => { toast('CSV export: use gitplm CLI for full export'); };
    load();
  }
};
