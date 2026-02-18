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
          ${parts.length === 0 ? '<p class="text-gray-500 text-center py-8">No parts found. Configure --pmDir to load gitplm CSVs.</p>' : `
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead><tr class="border-b text-left text-gray-500">
                <th class="pb-2 pr-4">IPN</th>
                <th class="pb-2 pr-4">Category</th>
                ${parts[0] && parts[0].fields ? Object.keys(parts[0].fields).filter(k=>k!=='_category'&&k!=='ipn'&&k!=='IPN'&&k!=='pn'&&k!=='PN').slice(0,5).map(k=>`<th class="pb-2 pr-4">${k}</th>`).join('') : ''}
              </tr></thead>
              <tbody>
                ${parts.map(p => `<tr class="table-row border-b border-gray-100" onclick="window._partsDetail('${p.ipn}')">
                  <td class="py-2 pr-4 font-mono text-blue-600">${p.ipn}</td>
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
      let bomHTML = '';
      if (isAssembly) {
        bomHTML = `<div class="mt-4 border-t pt-4">
          <div class="flex gap-2 mb-3">
            <button class="btn btn-secondary btn-sm" id="parts-tab-details" onclick="document.getElementById('parts-details-panel').style.display='block';document.getElementById('parts-bom-panel').style.display='none';">Details</button>
            <button class="btn btn-secondary btn-sm" id="parts-tab-bom" onclick="window._partsLoadBOM('${ipn}')">📋 BOM</button>
          </div>
          <div id="parts-details-panel">${fields}</div>
          <div id="parts-bom-panel" style="display:none"><p class="text-gray-400 text-center py-2">Loading BOM...</p></div>
        </div>`;
      }
      showModal('Part: ' + ipn, `<div class="font-mono text-lg text-blue-600 mb-4">${ipn}</div>${isAssembly ? bomHTML : fields}`);
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
    window._partsExport = () => { toast('CSV export: use gitplm CLI for full export'); };
    load();
  }
};
