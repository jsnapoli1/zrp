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
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No test records</p>':''}
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
      try { await api('POST','tests',getModalValues(o)); toast('Test recorded'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    load();
  }
};
