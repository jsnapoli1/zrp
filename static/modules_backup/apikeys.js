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
          ${keys.length === 0 ? '<p class="text-gray-400 text-center py-8">No API keys yet</p>' : ''}
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
          } catch(e) { toast(e.message, 'error'); }
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
