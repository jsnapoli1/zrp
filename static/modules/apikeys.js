window.module_apikeys = {
  render: async (container) => {
    async function load() {
      const res = await api('GET', 'apikeys');
      const keys = res.data || [];
      container.innerHTML = `
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">API Keys</h2>
          <button class="btn btn-primary" id="btn-new-key">+ Generate New Key</button>
        </div>
        <div class="card overflow-x-auto">
          <table class="w-full text-sm">
            <thead><tr class="border-b text-left text-gray-500">
              <th class="pb-2">Name</th><th class="pb-2">Prefix</th><th class="pb-2">Created</th>
              <th class="pb-2">Last Used</th><th class="pb-2">Expires</th><th class="pb-2">Enabled</th><th class="pb-2">Actions</th>
            </tr></thead>
            <tbody>${keys.map(k => `<tr class="border-b table-row">
              <td class="py-2 font-medium">${k.name}</td>
              <td class="py-2 font-mono text-xs text-gray-500">${k.key_prefix}…</td>
              <td class="py-2 text-gray-500">${k.created_at ? new Date(k.created_at).toLocaleDateString() : ''}</td>
              <td class="py-2 text-gray-500">${k.last_used ? new Date(k.last_used).toLocaleString() : 'Never'}</td>
              <td class="py-2 text-gray-500">${k.expires_at ? new Date(k.expires_at).toLocaleDateString() : 'Never'}</td>
              <td class="py-2">
                <button class="text-xs ${k.enabled ? 'text-green-600' : 'text-red-600'}" data-toggle="${k.id}" data-enabled="${k.enabled}">${k.enabled ? 'Active' : 'Disabled'}</button>
              </td>
              <td class="py-2">
                <button class="btn btn-danger text-xs" data-revoke="${k.id}">Revoke</button>
              </td>
            </tr>`).join('')}</tbody>
          </table>
          ${keys.length === 0 ? `<div class="text-center py-12">
            <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z"/></svg>
            <p class="text-gray-500 font-medium">No API keys yet</p>
            <p class="text-gray-400 text-sm mt-1">Generate a key to access the API programmatically</p>
          </div>` : ''}
        </div>`;

      document.getElementById('btn-new-key').onclick = () => {
        showModal('Generate API Key', `
          <label class="label">Key Name</label>
          <input class="input mb-3" data-field="name" placeholder="e.g. ERP Integration">
          <label class="label">Expires (optional)</label>
          <input class="input" data-field="expires_at" type="date">
        `, async (overlay) => {
          const vals = getModalValues(overlay);
          if (!vals.name) { toast('Name is required', 'error'); return; }
          const btn = overlay.querySelector('#modal-save');
          btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Generating...';
          try {
            const body = { name: vals.name };
            if (vals.expires_at) body.expires_at = vals.expires_at;
            const res = await fetch('/api/v1/apikeys', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(body)
            });
            const data = await res.json();
            overlay.remove();
            // Show the key once
            showModal('API Key Generated', `
              <div class="bg-yellow-50 border border-yellow-200 rounded-lg p-3 mb-3">
                <p class="text-yellow-800 text-sm font-medium">⚠️ This key will not be shown again. Copy it now.</p>
              </div>
              <label class="label">API Key</label>
              <div class="flex gap-2">
                <input class="input font-mono text-sm" value="${data.key}" readonly id="generated-key">
                <button class="btn btn-primary text-xs" id="copy-key-btn">Copy</button>
              </div>
            `);
            document.getElementById('copy-key-btn').onclick = () => {
              navigator.clipboard.writeText(data.key);
              toast('Key copied to clipboard');
            };
            setTimeout(load, 500);
          } catch(e) { toast(e.message, 'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
        });
      };

      container.querySelectorAll('[data-revoke]').forEach(btn => {
        btn.onclick = async () => {
          if (!confirm('Revoke this API key? This cannot be undone.')) return;
          await fetch('/api/v1/apikeys/' + btn.dataset.revoke, { method: 'DELETE' });
          toast('API key revoked');
          load();
        };
      });

      container.querySelectorAll('[data-toggle]').forEach(btn => {
        btn.onclick = async () => {
          const newEnabled = btn.dataset.enabled === '1' ? 0 : 1;
          await fetch('/api/v1/apikeys/' + btn.dataset.toggle, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ enabled: newEnabled })
          });
          load();
        };
      });
    }
    load();
  }
};
