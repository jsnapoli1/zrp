window.module_docs = {
  render: async (container) => {
    async function load() {
      const res = await api('GET', 'docs');
      const items = res.data || [];
      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Document Control</h2>
          <button class="btn btn-primary" onclick="window._docCreate()">+ New Document</button>
        </div>
        ${items.length===0?`<div class="text-center py-12">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
          <p class="text-gray-500 font-medium">No documents yet</p>
          <p class="text-gray-400 text-sm mt-1">Create your first document to get started</p>
          <button class="btn btn-primary mt-4" onclick="window._docCreate()">+ New Document</button>
        </div>`:`<div class="overflow-x-auto">
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Title</th><th class="pb-2">Category</th><th class="pb-2">Rev</th><th class="pb-2">Status</th><th class="pb-2">📎</th><th class="pb-2 w-8"></th>
        </tr></thead><tbody>
          ${items.map(d => `<tr class="table-row border-b border-gray-100" onclick="window._docEdit('${d.id}')">
            <td class="py-2 font-mono text-blue-600">${d.id}</td><td class="py-2">${d.title}</td>
            <td class="py-2">${d.category||''}</td><td class="py-2">${d.revision}</td><td class="py-2">${badge(d.status)}</td>
            <td class="py-2 text-center">${d.attachment_count ? '📎 '+d.attachment_count : ''}</td>
            <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
          </tr>`).join('')}
        </tbody></table>
        </div>`}
      </div>`;
    }
    const form = (d={}) => `<div class="space-y-3">
      <div><label class="label">Title</label><input class="input" data-field="title" value="${d.title||''}"></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Category</label><select class="input" data-field="category">
          ${['procedure','spec','drawing','datasheet','other'].map(s=>`<option ${d.category===s?'selected':''}>${s}</option>`).join('')}
        </select></div>
        <div><label class="label">Revision</label><input class="input" data-field="revision" value="${d.revision||'A'}"></div>
      </div>
      <div><label class="label">IPN (optional)</label><input class="input" data-field="ipn" value="${d.ipn||''}"></div>
      <div><label class="label">Content (Markdown)</label><textarea class="input" data-field="content" rows="5">${d.content||''}</textarea></div>
      <div><label class="label">Status</label><select class="input" data-field="status">
        ${['draft','review','approved','obsolete'].map(s=>`<option ${d.status===s?'selected':''}>${s}</option>`).join('')}
      </select></div>
    </div>`;
    window._docCreate = () => showModal('New Document', form(), async (o) => {
      const v = getModalValues(o);
      if (!v.title?.trim()) { toast('Title is required', 'error'); return; }
      const btn = o.querySelector('#modal-save');
      btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
      try { await api('POST', 'docs', v); toast('Document created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
    });
    window._docEdit = async (id) => {
      const d = (await api('GET', 'docs/' + id)).data;
      const legacyFile = d.file_path ? `<div class="mt-2 p-2 bg-yellow-50 border border-yellow-200 rounded text-xs">📄 Legacy file: <code>${d.file_path}</code></div>` : '';
      const o = showModal('Document: ' + id + ' — ' + (d.title||'').substring(0,40), form(d) + legacyFile + (d.status!=='approved'?`<button class="btn btn-success mt-3" id="doc-approve">✓ Approve</button>`:'') + attachmentsSection('document', id), async (o) => {
        const v = getModalValues(o);
        if (!v.title?.trim()) { toast('Title is required', 'error'); return; }
        const btn = o.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('PUT', 'docs/' + id, v); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
      o.querySelector('#doc-approve')?.addEventListener('click', async () => { await api('POST','docs/'+id+'/approve'); toast('Approved'); o.remove(); load(); });
      initAttachments(o, 'document', id);
    };
    load();
  }
};
