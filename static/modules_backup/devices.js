window.module_devices = {
  render: async (container) => {
    const bulk = setupBulkOps(container, 'devices/bulk', [
      {action:'decommission', label:'⛔ Decommission', class:'bg-yellow-600 hover:bg-yellow-700 text-white'},
      {action:'delete', label:'🗑 Delete', class:'bg-red-600 hover:bg-red-700 text-white'},
    ]);
    async function load() {
      const res = await api('GET', 'devices');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Device Registry</h2>
          <div class="flex gap-2">
            <button class="btn btn-secondary" onclick="window._devExportCSV()">📥 Export CSV</button>
            <button class="btn btn-secondary" onclick="document.getElementById('dev-import-file').click()">📤 Import CSV</button>
            <input type="file" id="dev-import-file" accept=".csv" class="hidden" onchange="window._devImportCSV(this)">
            <button class="btn btn-primary" onclick="window._devCreate()">+ Register Device</button>
          </div>
        </div>
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2 w-8">${bulk.headerCheckbox()}</th>
          <th class="pb-2">Serial</th><th class="pb-2">IPN</th><th class="pb-2">Customer</th><th class="pb-2">FW</th><th class="pb-2">Status</th><th class="pb-2">Location</th>
        </tr></thead><tbody>
          ${items.map(d => `<tr class="table-row border-b border-gray-100" onclick="window._devEdit('${d.serial_number}')">
            <td class="py-2">${bulk.checkbox(d.serial_number)}</td>
            <td class="py-2 font-mono text-blue-600">${d.serial_number}</td><td class="py-2">${d.ipn}</td>
            <td class="py-2">${d.customer||''}</td><td class="py-2">${d.firmware_version||''}</td>
            <td class="py-2">${badge(d.status)}</td><td class="py-2">${d.location||''}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No devices</p>':''}
      </div>`;
      bulk.init();
    }
    container.addEventListener('bulk-reload', load);
    const form = (d={}) => `<div class="space-y-3">
      <div><label class="label">Serial Number</label><input class="input" data-field="serial_number" value="${d.serial_number||''}" ${d.serial_number?'readonly':''}></div>
      <div><label class="label">IPN</label><input class="input" data-field="ipn" value="${d.ipn||''}"></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Firmware</label><input class="input" data-field="firmware_version" value="${d.firmware_version||''}"></div>
        <div><label class="label">Status</label><select class="input" data-field="status">
          ${['active','inactive','rma','decommissioned'].map(s=>`<option ${d.status===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
      </div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Customer</label><input class="input" data-field="customer" value="${d.customer||''}"></div>
        <div><label class="label">Location</label><input class="input" data-field="location" value="${d.location||''}"></div>
      </div>
      <div><label class="label">Install Date</label><input class="input" type="date" data-field="install_date" value="${d.install_date||''}"></div>
      <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2">${d.notes||''}</textarea></div>
    </div>`;
    window._devExportCSV = () => { window.location.href = '/api/v1/devices/export'; };
    window._devImportCSV = async (input) => {
      const file = input.files[0]; if (!file) return;
      const fd = new FormData(); fd.append('file', file);
      try {
        const res = await fetch('/api/v1/devices/import', { method: 'POST', body: fd });
        const json = await res.json(); const d = json.data || json;
        toast(`Imported ${d.imported||0} devices${d.skipped?', '+d.skipped+' skipped':''}${(d.errors||[]).length?', '+(d.errors||[]).length+' errors':''}`);
        load();
      } catch(e) { toast(e.message, 'error'); }
      input.value = '';
    };
    window._devCreate = () => showModal('Register Device', form(), async (o) => {
      try { await api('POST','devices',getModalValues(o)); toast('Device registered'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    window._devEdit = async (sn) => {
      const d = (await api('GET','devices/'+sn)).data;
      showModal('Device: '+sn, form(d)+`<button class="btn btn-secondary mt-3" id="dev-hist">📜 View History</button>`, async (o) => {
        try { await api('PUT','devices/'+sn,getModalValues(o)); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
      });
      document.getElementById('dev-hist')?.addEventListener('click', async () => {
        const h = (await api('GET','devices/'+sn+'/history')).data;
        showModal('History: '+sn, `<h3 class="font-semibold mb-2">Test Records</h3>
          ${(h.tests||[]).map(t=>`<div class="text-sm border-b py-1">${t.tested_at?.substring(0,16)} — ${badge(t.result)} ${t.test_type||''}</div>`).join('')||'<p class="text-gray-400">None</p>'}
          <h3 class="font-semibold mb-2 mt-4">Firmware Campaigns</h3>
          ${(h.campaigns||[]).map(c=>`<div class="text-sm border-b py-1">${c.campaign_id} — ${badge(c.status)}</div>`).join('')||'<p class="text-gray-400">None</p>'}`);
      });
    };
    load();
  }
};
