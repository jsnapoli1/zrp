window.module_vendors = {
  render: async (container) => {
    async function load() {
      const res = await api('GET', 'vendors');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Vendors</h2>
          <button class="btn btn-primary" onclick="window._vendorCreate()">+ New Vendor</button>
        </div>
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/></svg>
          <p class="text-gray-500 font-medium">No vendors yet</p>
          <p class="text-gray-400 text-sm mt-1">Add your first vendor to get started</p>
          <button class="btn btn-primary mt-4" onclick="window._vendorCreate()">+ New Vendor</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Name</th><th class="pb-2">Contact</th><th class="pb-2">Status</th><th class="pb-2">Lead Time</th><th class="pb-2 w-8"></th>
        </tr></thead><tbody>
          ${items.map(v => `<tr class="table-row border-b border-gray-100" onclick="window._vendorEdit('${v.id}')">
            <td class="py-2 font-mono">${v.id}</td><td class="py-2 font-medium">${v.name}</td>
            <td class="py-2 text-gray-500">${v.contact_email||''}</td>
            <td class="py-2">${badge(v.status)}</td><td class="py-2">${v.lead_time_days}d</td>
            <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
          </tr>`).join('')}
        </tbody></table>
        </div>`}
      </div>`;
    }
    const form = (v={}) => `<div class="space-y-3">
      <div><label class="label">Name</label><input class="input" data-field="name" value="${v.name||''}"></div>
      <div><label class="label">Website</label><input class="input" data-field="website" value="${v.website||''}"></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Contact Name</label><input class="input" data-field="contact_name" value="${v.contact_name||''}"></div>
        <div><label class="label">Contact Email</label><input class="input" data-field="contact_email" value="${v.contact_email||''}"></div>
      </div>
      <div><label class="label">Phone</label><input class="input" data-field="contact_phone" value="${v.contact_phone||''}"></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Status</label><select class="input" data-field="status">
          ${['active','inactive','preferred'].map(s=>`<option ${v.status===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
        <div><label class="label">Lead Time (days)</label><input class="input" type="number" data-field="lead_time_days" value="${v.lead_time_days||0}"></div>
      </div>
      <div><label class="label">Notes</label><textarea class="input" data-field="notes" rows="2">${v.notes||''}</textarea></div>
    </div>`;
    window._vendorCreate = () => showModal('New Vendor', form(), async (o) => {
      const v = getModalValues(o); v.lead_time_days = parseInt(v.lead_time_days)||0;
      if (!v.name?.trim()) { toast('Vendor name is required', 'error'); return; }
      const btn = o.querySelector('#modal-save');
      btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
      try { await api('POST','vendors',v); toast('Vendor created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
    });
    window._vendorEdit = async (id) => {
      const v = (await api('GET','vendors/'+id)).data;
      showModal('Vendor: '+v.id+' — '+v.name, form(v)+`<button class="btn btn-danger mt-3" onclick="window._vendorDel('${id}')">Delete Vendor</button>`, async (o) => {
        const vals = getModalValues(o); vals.lead_time_days = parseInt(vals.lead_time_days)||0;
        if (!vals.name?.trim()) { toast('Vendor name is required', 'error'); return; }
        const btn = o.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('PUT','vendors/'+id,vals); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
    };
    window._vendorDel = async (id) => { if(confirm('Delete this vendor?')) { await api('DELETE','vendors/'+id); toast('Deleted'); document.querySelector('.modal-overlay')?.remove(); load(); }};
    load();
  }
};
