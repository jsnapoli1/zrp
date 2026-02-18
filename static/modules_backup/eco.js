window.module_ecos = {
  render: async (container) => {
    async function load() {
      const res = await api('GET', 'ecos');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Engineering Change Orders</h2>
          <button class="btn btn-primary" onclick="window._ecoCreate()">+ New ECO</button>
        </div>
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Title</th><th class="pb-2">Status</th><th class="pb-2">Priority</th><th class="pb-2">Created</th>
        </tr></thead><tbody>
          ${items.map(e => `<tr class="table-row border-b border-gray-100" onclick="window._ecoEdit('${e.id}')">
            <td class="py-2 font-mono text-blue-600">${e.id}</td>
            <td class="py-2">${e.title}</td>
            <td class="py-2">${badge(e.status)}</td>
            <td class="py-2">${badge(e.priority)}</td>
            <td class="py-2 text-gray-500">${e.created_at?.substring(0,10)}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No ECOs</p>':''}
      </div>`;
    }

    function ipnAutocompleteHTML(fieldId, currentValue) {
      return `
        <div class="relative">
          <label class="label">Affected IPNs</label>
          <div id="${fieldId}-tags" class="flex flex-wrap gap-1 mb-1"></div>
          <input class="input" id="${fieldId}-input" placeholder="Type to search parts..." autocomplete="off">
          <input type="hidden" data-field="affected_ipns" id="${fieldId}-hidden" value="${currentValue||''}">
          <div id="${fieldId}-dropdown" class="absolute z-10 w-full bg-white border border-gray-200 rounded-lg shadow-lg mt-1 max-h-48 overflow-y-auto hidden"></div>
        </div>`;
    }

    function setupIPNAutocomplete(overlay, fieldId) {
      const input = overlay.querySelector(`#${fieldId}-input`);
      const dropdown = overlay.querySelector(`#${fieldId}-dropdown`);
      const tagsEl = overlay.querySelector(`#${fieldId}-tags`);
      const hiddenInput = overlay.querySelector(`#${fieldId}-hidden`);
      let selected = [];

      // Parse initial value
      const initVal = hiddenInput.value.trim();
      if (initVal) {
        try {
          if (initVal.startsWith('[')) selected = JSON.parse(initVal);
          else selected = initVal.split(',').map(s=>s.trim()).filter(Boolean);
        } catch(e) { selected = initVal.split(',').map(s=>s.trim()).filter(Boolean); }
      }

      function renderTags() {
        tagsEl.innerHTML = selected.map((ipn,i) =>
          `<span class="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-100 text-blue-800 rounded text-xs font-mono">
            ${ipn}<button onclick="window._removeIPN_${fieldId}(${i})" class="text-blue-500 hover:text-red-500">&times;</button>
          </span>`).join('');
        hiddenInput.value = selected.join(',');
      }

      window[`_removeIPN_${fieldId}`] = (idx) => { selected.splice(idx, 1); renderTags(); };
      renderTags();

      let debounceTimer;
      input.addEventListener('input', () => {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(async () => {
          const q = input.value.trim();
          if (q.length < 2) { dropdown.classList.add('hidden'); return; }
          try {
            const res = await api('GET', `parts?q=${encodeURIComponent(q)}&limit=10`);
            const parts = res.data || [];
            if (parts.length === 0) { dropdown.classList.add('hidden'); return; }
            dropdown.innerHTML = parts.map(p =>
              `<div class="px-3 py-2 hover:bg-blue-50 cursor-pointer text-sm" data-ipn="${p.ipn}">
                <span class="font-mono text-blue-600">${p.ipn}</span>
                <span class="text-gray-500 ml-2">${p.fields?.description || p.fields?.Description || ''}</span>
              </div>`).join('');
            dropdown.classList.remove('hidden');
            dropdown.querySelectorAll('[data-ipn]').forEach(el => {
              el.addEventListener('click', () => {
                const ipn = el.dataset.ipn;
                if (!selected.includes(ipn)) { selected.push(ipn); renderTags(); }
                input.value = '';
                dropdown.classList.add('hidden');
              });
            });
          } catch(e) { dropdown.classList.add('hidden'); }
        }, 300);
      });

      input.addEventListener('blur', () => setTimeout(() => dropdown.classList.add('hidden'), 200));
    }

    function affectedPartsTable(parts) {
      if (!parts || parts.length === 0) return '';
      return `<div class="mt-4"><h3 class="text-sm font-semibold text-gray-700 mb-2">Affected Parts</h3>
        <table class="w-full text-xs"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-1">IPN</th><th class="pb-1">Description</th><th class="pb-1">MPN</th><th class="pb-1">Manufacturer</th><th class="pb-1">Status</th>
        </tr></thead><tbody>
          ${parts.map(p => `<tr class="border-b border-gray-100">
            <td class="py-1 font-mono text-blue-600">${p.ipn||''}</td>
            <td class="py-1">${p.description||p.Description||''}</td>
            <td class="py-1">${p.mpn||p.MPN||p.part_number||''}</td>
            <td class="py-1">${p.manufacturer||p.Manufacturer||p.mfr||''}</td>
            <td class="py-1">${p.status||p.Status||''}</td>
          </tr>`).join('')}
        </tbody></table></div>`;
    }

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
        ${ipnAutocompleteHTML('eco-ipns', e.affected_ipns||'')}
      </div>`;

    window._ecoCreate = () => {
      const overlay = showModal('New ECO', formHTML(), async (overlay) => {
        const v = getModalValues(overlay);
        try { await api('POST', 'ecos', v); toast('ECO created'); overlay.remove(); load(); } catch(e) { toast(e.message, 'error'); }
      });
      setupIPNAutocomplete(overlay, 'eco-ipns');
    };

    window._ecoEdit = async (id) => {
      const res = await api('GET', 'ecos/' + id);
      const e = res.data;
      const ncrBadge = e.ncr_id ? `<div class="mb-3"><span class="badge bg-purple-100 text-purple-800 cursor-pointer" onclick="navigate('ncr')">From ${e.ncr_id}</span></div>` : '';
      const overlay = showModal('Edit ECO: ' + id, ncrBadge + formHTML(e) +
        affectedPartsTable(e.affected_parts) + `
        <div class="flex gap-2 mt-4">
          ${e.status==='review'||e.status==='draft'?`<button class="btn btn-success" id="eco-approve">✓ Approve</button>`:''}
          ${e.status==='approved'?`<button class="btn btn-primary" id="eco-implement">🚀 Implement</button>`:''}
        </div>` + attachmentsSection('eco', id), async (overlay) => {
        const v = getModalValues(overlay);
        try { await api('PUT', 'ecos/' + id, v); toast('ECO updated'); overlay.remove(); load(); } catch(e) { toast(e.message, 'error'); }
      });
      setupIPNAutocomplete(overlay, 'eco-ipns');
      initAttachments(overlay, 'eco', id);
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
