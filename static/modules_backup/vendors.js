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
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Name</th><th class="pb-2">Contact</th><th class="pb-2">Status</th><th class="pb-2">Lead Time</th>
        </tr></thead><tbody>
          ${items.map(v => `<tr class="table-row border-b border-gray-100" onclick="window._vendorEdit('${v.id}')">
            <td class="py-2 font-mono">${v.id}</td><td class="py-2 font-medium">${v.name}</td>
            <td class="py-2 text-gray-500">${v.contact_email||''}</td>
            <td class="py-2">${badge(v.status)}</td><td class="py-2">${v.lead_time_days}d</td>
          </tr>`).join('')}
        </tbody></table>
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
      try { await api('POST','vendors',v); toast('Vendor created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    window._vendorEdit = async (id) => {
      const v = (await api('GET','vendors/'+id)).data;
      showModal('Edit Vendor: '+v.name, form(v)+`<button class="btn btn-danger mt-3" onclick="window._vendorDel('${id}')">Delete Vendor</button>`, async (o) => {
        const vals = getModalValues(o); vals.lead_time_days = parseInt(vals.lead_time_days)||0;
        try { await api('PUT','vendors/'+id,vals); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
      });
    };
    window._vendorDel = async (id) => { if(confirm('Delete this vendor?')) { await api('DELETE','vendors/'+id); toast('Deleted'); document.querySelector('.modal-overlay')?.remove(); load(); }};
    load();
  }
};
