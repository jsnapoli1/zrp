window.module_testing = {
  render: async (container) => {
    async function load() {
      const res = await api('GET', 'tests');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Test Records</h2>
          <button class="btn btn-primary" onclick="window._testCreate()">+ New Test</button>
        </div>
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/></svg>
          <p class="text-gray-500 font-medium">No test records yet</p>
          <p class="text-gray-400 text-sm mt-1">Record your first test to get started</p>
          <button class="btn btn-primary mt-4" onclick="window._testCreate()">+ New Test</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Serial</th><th class="pb-2">IPN</th><th class="pb-2">Type</th><th class="pb-2">Result</th><th class="pb-2">FW</th><th class="pb-2">Date</th>
        </tr></thead><tbody>
          ${items.map(t => `<tr class="table-row border-b border-gray-100">
            <td class="py-2">${t.id}</td><td class="py-2 font-mono">${t.serial_number}</td>
            <td class="py-2">${t.ipn}</td><td class="py-2">${t.test_type||''}</td>
            <td class="py-2">${badge(t.result)}</td><td class="py-2">${t.firmware_version||''}</td>
            <td class="py-2 text-gray-500">${t.tested_at?.substring(0,16)}</td>
          </tr>`).join('')}
        </tbody></table>
        </div>`}
      </div>`;
    }
    window._testCreate = () => showModal('New Test Record', `<div class="space-y-3">
      <div><label class="label">Serial Number</label><input class="input" data-field="serial_number"></div>
      <div><label class="label">IPN</label><input class="input" data-field="ipn"></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Test Type</label><select class="input" data-field="test_type">
          <option>factory</option><option>qa</option><option>field</option>
        </select></div>
        <div><label class="label">Result</label><select class="input" data-field="result">
          <option>pass</option><option>fail</option><option>conditional</option>
        </select></div>
      </div>
      <div><label class="label">Firmware Version</label><input class="input" data-field="firmware_version"></div>
      <div><label class="label">Measurements (JSON)</label><input class="input" data-field="measurements" placeholder='{"voltage":12.0}'></div>
      <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2"></textarea></div>
    </div>`, async (o) => {
      const v = getModalValues(o);
      if (!v.serial_number?.trim()) { toast('Serial number is required', 'error'); return; }
      if (!v.ipn?.trim()) { toast('IPN is required', 'error'); return; }
      const btn = o.querySelector('#modal-save');
      btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
      try { await api('POST','tests',v); toast('Test recorded'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
    });
    load();
  }
};
