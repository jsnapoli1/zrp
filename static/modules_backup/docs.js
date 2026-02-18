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
        <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
          <th class="pb-2">ID</th><th class="pb-2">Title</th><th class="pb-2">Category</th><th class="pb-2">Rev</th><th class="pb-2">Status</th><th class="pb-2">📎</th>
        </tr></thead><tbody>
          ${items.map(d => `<tr class="table-row border-b border-gray-100" onclick="window._docEdit('${d.id}')">
            <td class="py-2 font-mono text-blue-600">${d.id}</td><td class="py-2">${d.title}</td>
            <td class="py-2">${d.category||''}</td><td class="py-2">${d.revision}</td><td class="py-2">${badge(d.status)}</td>
            <td class="py-2 text-center">${d.attachment_count ? '📎 '+d.attachment_count : ''}</td>
          </tr>`).join('')}
        </tbody></table>
        ${items.length===0?'<p class="text-center text-gray-400 py-4">No documents</p>':''}
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
      try { await api('POST', 'docs', getModalValues(o)); toast('Document created'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
    });
    window._docEdit = async (id) => {
      const d = (await api('GET', 'docs/' + id)).data;
      const legacyFile = d.file_path ? `<div class="mt-2 p-2 bg-yellow-50 border border-yellow-200 rounded text-xs">📄 Legacy file: <code>${d.file_path}</code></div>` : '';
      const o = showModal('Edit: ' + id, form(d) + legacyFile + (d.status!=='approved'?`<button class="btn btn-success mt-3" id="doc-approve">✓ Approve</button>`:'') + attachmentsSection('document', id), async (o) => {
        try { await api('PUT', 'docs/' + id, getModalValues(o)); toast('Updated'); o.remove(); load(); } catch(e) { toast(e.message,'error'); }
      });
      o.querySelector('#doc-approve')?.addEventListener('click', async () => { await api('POST','docs/'+id+'/approve'); toast('Approved'); o.remove(); load(); });
      initAttachments(o, 'document', id);
    };
    load();
  }
};
