window.module_email = {
  render: async (container) => {
    const res = await api('GET', 'email/config');
    const c = res.data || {};

    const logRes = await api('GET', 'email-log');
    const logs = logRes.data || [];

    container.innerHTML = `
      <div class="card max-w-2xl">
        <h2 class="text-lg font-semibold mb-4">📧 Email Settings</h2>
        <p class="text-sm text-gray-500 mb-6">Configure SMTP to receive email notifications for low stock, overdue work orders, ECO approvals, and other alerts.</p>
        <div class="space-y-4">
          <div class="flex items-center gap-3 mb-4">
            <label class="label mb-0">Enable Email Notifications</label>
            <button id="email-toggle" class="relative w-12 h-6 rounded-full transition-colors ${c.enabled ? 'bg-blue-600' : 'bg-gray-300'}" onclick="window._emailToggle()">
              <span class="absolute top-0.5 ${c.enabled ? 'left-6' : 'left-0.5'} w-5 h-5 bg-white rounded-full shadow transition-all"></span>
            </button>
            <span id="email-toggle-label" class="text-sm ${c.enabled ? 'text-green-600' : 'text-gray-400'}">${c.enabled ? 'Enabled' : 'Disabled'}</span>
          </div>
          <div>
            <label class="label">SMTP Host</label>
            <input type="text" class="input" id="email-host" value="${c.smtp_host || ''}" placeholder="smtp.gmail.com">
          </div>
          <div>
            <label class="label">SMTP Port</label>
            <input type="number" class="input" id="email-port" value="${c.smtp_port || 587}" placeholder="587">
          </div>
          <div>
            <label class="label">Username</label>
            <input type="text" class="input" id="email-user" value="${c.smtp_user || ''}" placeholder="user@example.com">
          </div>
          <div>
            <label class="label">Password</label>
            <input type="password" class="input" id="email-pass" value="${c.smtp_password || ''}" placeholder="••••••••">
          </div>
          <div>
            <label class="label">From Address</label>
            <input type="email" class="input" id="email-from" value="${c.from_address || ''}" placeholder="notifications@example.com">
          </div>
          <div>
            <label class="label">From Name</label>
            <input type="text" class="input" id="email-name" value="${c.from_name || 'ZRP'}" placeholder="ZRP">
          </div>
          <div class="flex gap-2 pt-2">
            <button class="btn btn-primary" onclick="window._emailSave()">💾 Save</button>
          </div>
        </div>
        <div class="mt-6 pt-4 border-t">
          <h3 class="text-sm font-semibold mb-3">Send Test Email</h3>
          <div class="flex gap-2">
            <input type="email" class="input flex-1" id="email-test-to" placeholder="recipient@example.com">
            <button class="btn btn-secondary" onclick="window._emailTest()">📤 Send Test</button>
          </div>
        </div>
      </div>

      <div class="card max-w-4xl mt-6">
        <h2 class="text-lg font-semibold mb-4">📋 Email Log</h2>
        <div class="overflow-x-auto"><table class="w-full text-sm">
          <thead>
            <tr class="border-b">
              <th class="text-left py-2 px-2">To</th>
              <th class="text-left py-2 px-2">Subject</th>
              <th class="text-left py-2 px-2">Status</th>
              <th class="text-left py-2 px-2">Error</th>
              <th class="text-left py-2 px-2">Sent At</th>
            </tr>
          </thead>
          <tbody>
            ${logs.length === 0 ? '<tr><td colspan="5" class="text-center py-4 text-gray-400">No emails sent yet</td></tr>' :
              logs.map(l => `<tr class="border-b hover:bg-gray-50">
                <td class="py-2 px-2">${l.to_address}</td>
                <td class="py-2 px-2">${l.subject}</td>
                <td class="py-2 px-2"><span class="px-2 py-0.5 rounded text-xs font-medium ${l.status === 'sent' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}">${l.status}</span></td>
                <td class="py-2 px-2 text-red-600 text-xs">${l.error || ''}</td>
                <td class="py-2 px-2 text-gray-500">${l.sent_at}</td>
              </tr>`).join('')}
          </tbody>
        </table></div>
      </div>
    `;

    let enabled = c.enabled ? 1 : 0;

    window._emailToggle = () => {
      enabled = enabled ? 0 : 1;
      const btn = document.getElementById('email-toggle');
      const lbl = document.getElementById('email-toggle-label');
      const span = btn.querySelector('span');
      btn.className = 'relative w-12 h-6 rounded-full transition-colors ' + (enabled ? 'bg-blue-600' : 'bg-gray-300');
      span.className = 'absolute top-0.5 w-5 h-5 bg-white rounded-full shadow transition-all ' + (enabled ? 'left-6' : 'left-0.5');
      lbl.textContent = enabled ? 'Enabled' : 'Disabled';
      lbl.className = 'text-sm ' + (enabled ? 'text-green-600' : 'text-gray-400');
    };

    window._emailSave = async () => {
      const btn = container.querySelector('[onclick="window._emailSave()"]');
      if (btn) { btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...'; }
      try {
        await api('PUT', 'email/config', {
          smtp_host: document.getElementById('email-host').value,
          smtp_port: parseInt(document.getElementById('email-port').value) || 587,
          smtp_user: document.getElementById('email-user').value,
          smtp_password: document.getElementById('email-pass').value,
          from_address: document.getElementById('email-from').value,
          from_name: document.getElementById('email-name').value,
          enabled: enabled,
        });
        toast('Email settings saved');
      } catch(e) { toast(e.message, 'error'); } finally { if (btn) { btn.disabled = false; btn.textContent = '💾 Save'; } }
    };

    window._emailTest = async () => {
      const to = document.getElementById('email-test-to').value;
      if (!to) { toast('Enter a recipient address', 'error'); return; }
      try {
        await api('POST', 'email/test', { to });
        toast('Test email sent to ' + to);
        navigate('email'); // reload to show log
      } catch(e) { toast(e.message, 'error'); }
    };
  }
};
