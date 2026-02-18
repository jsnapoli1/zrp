window.module_ncr = {
  render: async (container) => {
    const bulk = setupBulkOps(container, 'ncrs/bulk', [
      {action:'resolve', label:'✓ Resolve', class:'bg-green-600 hover:bg-green-700 text-white'},
      {action:'close', label:'Close', class:'bg-gray-600 hover:bg-gray-700 text-white'},
      {action:'delete', label:'🗑 Delete', class:'bg-red-600 hover:bg-red-700 text-white'},
    ]);
    async function load() {
      const res = await api('GET', 'ncrs');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Non-Conformance Reports</h2>
          <button class="btn btn-primary" onclick="window._ncrCreate()">+ New NCR</button>
        </div>
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2 w-8">${bulk.headerCheckbox()}</th>
          <th class="pb-2">ID</th><th class="pb-2">Title</th><th class="pb-2">Defect</th><th class="pb-2">Severity</th><th class="pb-2">Status</th>
        </tr></thead><tbody>
          ${items.map(n => `<tr class="table-row border-b border-gray-100" onclick="window._ncrEdit('${n.id}')">
            <td class="py-2">${bulk.checkbox(n.id)}</td>
            <td class="py-2 font-mono text-blue-600">${n.id}</td><td class="py-2">${n.title}</td>
            <td class="py-2">${n.defect_type||''}</td><td class="py-2">${badge(n.severity)}</td><td class="py-2">${badge(n.status)}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No NCRs</p>':''}
      </div>`;
      bulk.init();
    }
    container.addEventListener('bulk-reload', load);
    const form = (n={}) => `<div class="space-y-3">
      <div><label class="label">Title</label><input class="input" data-field="title" value="${n.title||''}"></div>
      <div><label class="label">Description</label><textarea class="input" data-field="description" rows="2">${n.description||''}</textarea></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">IPN</label><input class="input" data-field="ipn" value="${n.ipn||''}"></div>
        <div><label class="label">Serial Number</label><input class="input" data-field="serial_number" value="${n.serial_number||''}"></div>
      </div>
      <div class="grid grid-cols-3 gap-3">
        <div><label class="label">Defect Type</label><select class="input" data-field="defect_type">
          ${['workmanship','design','component','process','other'].map(s=>`<option ${n.defect_type===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
        <div><label class="label">Severity</label><select class="input" data-field="severity">
          ${['minor','major','critical'].map(s=>`<option ${n.severity===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
        <div><label class="label">Status</label><select class="input" data-field="status">
          ${['open','investigating','resolved','closed'].map(s=>`<option ${n.status===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
      </div>
      <div><label class="label">Root Cause</label><textarea class="input" data-field="root_cause" rows="2">${n.root_cause||''}</textarea></div>
      <div><label class="label">Corrective Action</label><textarea class="input" data-field="corrective_action" rows="2">${n.corrective_action||''}</textarea></div>
      <div id="ncr-eco-checkbox" class="hidden">
        <label class="flex items-center gap-2 text-sm">
          <input type="checkbox" checked id="ncr-create-eco"> Create linked ECO for corrective action
        </label>
      </div>
    </div>`;

    function setupEcoCheckbox(overlay) {
      const statusSel = overlay.querySelector('[data-field="status"]');
      const caField = overlay.querySelector('[data-field="corrective_action"]');
      const checkDiv = overlay.querySelector('#ncr-eco-checkbox');
      function update() {
        const status = statusSel?.value;
        const ca = caField?.value?.trim();
        if ((status === 'resolved' || status === 'closed') && ca) {
          checkDiv.classList.remove('hidden');
        } else {
          checkDiv.classList.add('hidden');
        }
      }
      statusSel?.addEventListener('change', update);
      caField?.addEventListener('input', update);
      update();
    }

    window._ncrCreate = () => {
      const o = showModal('New NCR', form(), async (o) => {
        try { await api('POST','ncrs',getModalValues(o)); toast('NCR created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
      });
      setupEcoCheckbox(o);
    };
    window._ncrEdit = async (id) => {
      const n = (await api('GET','ncrs/'+id)).data;
      const o = showModal('NCR: '+id, form(n) + attachmentsSection('ncr', id), async (o) => {
        const v = getModalValues(o);
        const createEco = o.querySelector('#ncr-create-eco')?.checked || false;
        v.create_eco = createEco;
        try {
          const res = await api('PUT','ncrs/'+id, v);
          const data = res.data;
          if (data.linked_eco_id) {
            toast(`NCR resolved. ${data.linked_eco_id} created for corrective action.`);
          } else {
            toast('NCR updated');
          }
          o.remove(); load();
        } catch(e) { toast(e.message,'error'); }
      });
      initAttachments(o, 'ncr', id);
      setupEcoCheckbox(o);
    };
    load();
  }
};
