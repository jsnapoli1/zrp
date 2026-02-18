window.module_firmware = {
  render: async (container) => {
    let activeSSE = null;

    async function load() {
      if (activeSSE) { activeSSE.close(); activeSSE = null; }
      const res = await api('GET', 'campaigns');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Firmware Campaigns</h2>
          <button class="btn btn-primary" onclick="window._fwCreate()">+ New Campaign</button>
        </div>
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8.25 3v1.5M4.5 8.25H3m18 0h-1.5M4.5 12H3m18 0h-1.5m-15 3.75H3m18 0h-1.5M8.25 19.5V21M12 3v1.5m0 15V21m3.75-18v1.5m0 15V21m-9-1.5h10.5a2.25 2.25 0 002.25-2.25V6.75a2.25 2.25 0 00-2.25-2.25H6.75A2.25 2.25 0 004.5 6.75v10.5a2.25 2.25 0 002.25 2.25zm.75-12h9v9h-9v-9z"/></svg>
          <p class="text-gray-500 font-medium">No firmware campaigns</p>
          <p class="text-gray-400 text-sm mt-1">Create a campaign to push firmware updates</p>
          <button class="btn btn-primary mt-4" onclick="window._fwCreate()">+ New Campaign</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Name</th><th class="pb-2">Version</th><th class="pb-2">Category</th><th class="pb-2">Status</th><th class="pb-2 w-8"></th>
        </tr></thead><tbody>
          ${items.map(f => `<tr class="table-row border-b border-gray-100" onclick="window._fwEdit('${f.id}')">
            <td class="py-2 font-mono text-blue-600">${f.id}</td><td class="py-2">${f.name}</td>
            <td class="py-2 font-mono">${f.version}</td><td class="py-2">${badge(f.category)}</td><td class="py-2">${badge(f.status)}</td>
            <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
          </tr>`).join('')}
        </tbody></table></div>`}
      </div>`;
    }
    const form = (f={}) => `<div class="space-y-3">
      <div><label class="label">Name</label><input class="input" data-field="name" value="${f.name||''}"></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Version</label><input class="input" data-field="version" value="${f.version||''}"></div>
        <div><label class="label">Category</label><select class="input" data-field="category">
          ${['dev','beta','public'].map(s=>`<option ${f.category===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
      </div>
      <div><label class="label">Status</label><select class="input" data-field="status">
        ${['draft','active','completed','cancelled'].map(s=>`<option ${f.status===s?'selected':''}>${s}</option>`).join('')}
      </select></div>
      <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2">${f.notes||''}</textarea></div>
    </div>`;
    window._fwCreate = () => showModal('New Campaign', form(), async (o) => {
      const v = getModalValues(o);
      if (!v.name?.trim()) { toast('Campaign name is required', 'error'); return; }
      if (!v.version?.trim()) { toast('Version is required', 'error'); return; }
      const btn = o.querySelector('#modal-save');
      btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
      try { await api('POST','campaigns',v); toast('Campaign created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
    });
    window._fwEdit = async (id) => {
      const f = (await api('GET','campaigns/'+id)).data;
      const devRes = await api('GET','campaigns/'+id+'/devices');
      const devices = devRes.data || [];

      const devicesHTML = devices.length ? `
        <div class="mt-4 border-t pt-4">
          <h3 class="font-semibold mb-2">Campaign Devices</h3>
          <table class="w-full text-sm"><thead><tr class="border-b text-gray-500">
            <th class="pb-1 text-left">Serial</th><th class="pb-1 text-left">Status</th><th class="pb-1 text-left">Actions</th>
          </tr></thead><tbody id="fw-dev-list">
            ${devices.map(d => `<tr class="border-b border-gray-100" data-serial="${d.serial_number}">
              <td class="py-1 font-mono">${d.serial_number}</td>
              <td class="py-1"><span class="badge badge-${d.status}">${d.status}</span></td>
              <td class="py-1">${d.status==='pending'||d.status==='sent'?`
                <button class="btn btn-success text-xs py-0.5 px-2" onclick="window._fwMark('${id}','${d.serial_number}','updated')">✅ Updated</button>
                <button class="btn btn-danger text-xs py-0.5 px-2" onclick="window._fwMark('${id}','${d.serial_number}','failed')">❌ Failed</button>
              `:''}</td>
            </tr>`).join('')}
          </tbody></table>
        </div>` : '';

      showModal('Campaign: '+id, form(f)+`
        <div class="mt-4 p-3 bg-gray-50 rounded-lg">
          <div class="flex justify-between text-sm mb-1"><span>Progress</span><span id="fw-pct-label">...</span></div>
          <div class="w-full bg-gray-200 rounded-full h-3"><div id="fw-progress-bar" class="bg-blue-600 h-3 rounded-full transition-all duration-500" style="width:0%"></div></div>
          <div class="flex gap-4 text-xs mt-2 text-gray-500" id="fw-stats"></div>
        </div>
        ${f.status==='draft'?`<button class="btn btn-success mt-3" id="fw-launch">🚀 Launch Campaign</button>`:''}
        ${devicesHTML}
      `, async (o) => {
        const v = getModalValues(o);
        const btn = o.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('PUT','campaigns/'+id,v); toast('Updated'); o.remove(); if(activeSSE){activeSSE.close();activeSSE=null;} load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });

      // Start SSE for live progress
      if (f.status === 'active') {
        startSSE(id);
      } else {
        // Just fetch once
        const prog = (await api('GET','campaigns/'+id+'/progress')).data;
        updateProgress(prog);
      }

      document.getElementById('fw-launch')?.addEventListener('click', async () => {
        const r = await api('POST','campaigns/'+id+'/launch');
        toast(`Campaign launched! ${r.data.devices_added} devices added`);
        document.querySelector('.modal-overlay')?.remove();
        if(activeSSE){activeSSE.close();activeSSE=null;}
        load();
      });
    };

    function updateProgress(prog) {
      const total = prog.total||0;
      const pct = total ? Math.round(((prog.updated||0)+(prog.failed||0))/total*100) : 0;
      const bar = document.getElementById('fw-progress-bar');
      const label = document.getElementById('fw-pct-label');
      const stats = document.getElementById('fw-stats');
      if (bar) bar.style.width = pct+'%';
      if (label) label.textContent = `${(prog.updated||0)+(prog.failed||0)}/${total} (${pct}%)`;
      if (stats) stats.innerHTML = `<span>⏳ Pending: ${prog.pending||0}</span><span>📤 Sent: ${prog.sent||0}</span><span>✅ Updated: ${prog.updated||0}</span><span>❌ Failed: ${prog.failed||0}</span>`;
    }

    function startSSE(id) {
      if (activeSSE) activeSSE.close();
      activeSSE = new EventSource('/api/v1/campaigns/'+id+'/stream');
      activeSSE.onmessage = (e) => {
        try {
          const prog = JSON.parse(e.data);
          updateProgress(prog);
        } catch(err) {}
      };
      activeSSE.onerror = () => { activeSSE.close(); activeSSE = null; };
    }

    window._fwMark = async (campaignId, serial, status) => {
      try {
        await api('POST', `campaigns/${campaignId}/devices/${serial}/mark`, { status });
        toast(`Marked ${serial} as ${status}`);
        // Update the row in-place
        const row = document.querySelector(`tr[data-serial="${serial}"]`);
        if (row) {
          row.children[1].innerHTML = `<span class="badge badge-${status}">${status}</span>`;
          row.children[2].innerHTML = '';
        }
      } catch(e) { toast(e.message, 'error'); }
    };

    load();
  }
};
