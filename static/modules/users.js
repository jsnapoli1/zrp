window.module_users = {
  render: async (container) => {
    async function load() {
      try {
        const res = await api('GET', 'users');
        const users = res.data || [];
        container.innerHTML = `<div class="card">
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-lg font-semibold">User Management</h2>
            <button class="btn btn-primary" onclick="window._userCreate()">+ New User</button>
          </div>
          ${users.length === 0 ? `<div class="text-center py-12">
            <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"/></svg>
            <p class="text-gray-500 font-medium">No users yet</p>
            <button class="btn btn-primary mt-4" onclick="window._userCreate()">+ New User</button>
          </div>` : `<div class="overflow-x-auto">
          <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
            <th class="pb-2">Username</th><th class="pb-2">Display Name</th><th class="pb-2">Role</th><th class="pb-2">Status</th><th class="pb-2">Last Login</th><th class="pb-2 w-8"></th>
          </tr></thead><tbody>
            ${users.map(u => {
              const roleBadge = u.role === 'admin' ? '<span class="badge bg-red-100 text-red-800">admin</span>'
                : u.role === 'readonly' ? '<span class="badge bg-gray-200 text-gray-700">readonly</span>'
                : '<span class="badge bg-blue-100 text-blue-800">user</span>';
              const statusBadge = u.active ? '<span class="badge bg-green-100 text-green-800">active</span>' : '<span class="badge bg-gray-200 text-gray-600">inactive</span>';
              return `<tr class="table-row border-b border-gray-100" onclick="window._userEdit(${u.id})">
                <td class="py-2 font-mono">${u.username}</td>
                <td class="py-2">${u.display_name || ''}</td>
                <td class="py-2">${roleBadge}</td>
                <td class="py-2">${statusBadge}</td>
                <td class="py-2 text-gray-500">${u.last_login ? u.last_login.substring(0,16) : 'Never'}</td>
                <td class="py-2 text-gray-400"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></td>
              </tr>`;
            }).join('')}
          </tbody></table></div>`}
        </div>`;
      } catch(e) {
        container.innerHTML = `<div class="card"><p class="text-red-500">Admin access required</p></div>`;
      }
    }

    window._userCreate = () => {
      showModal('New User', `<div class="space-y-3">
        <div><label class="label">Username</label><input class="input" data-field="username"></div>
        <div><label class="label">Display Name</label><input class="input" data-field="display_name"></div>
        <div><label class="label">Password</label><input class="input" type="password" data-field="password"></div>
        <div><label class="label">Role</label><select class="input" data-field="role">
          <option value="user">User</option>
          <option value="admin">Admin</option>
          <option value="readonly">Read Only</option>
        </select></div>
      </div>`, async (overlay) => {
        const v = getModalValues(overlay);
        if (!v.username?.trim()) { toast('Username is required', 'error'); return; }
        if (!v.password?.trim()) { toast('Password is required', 'error'); return; }
        const btn = overlay.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try { await api('POST', 'users', v); toast('User created'); overlay.remove(); load(); } catch(e) { toast(e.message, 'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
    };

    window._userEdit = async (id) => {
      const res = await api('GET', 'users');
      const u = (res.data || []).find(x => x.id === id);
      if (!u) return;
      const isSelf = currentUser && currentUser.id === id;
      showModal('Edit User: ' + u.username, `<div class="space-y-3">
        <div><label class="label">Display Name</label><input class="input" data-field="display_name" value="${u.display_name || ''}"></div>
        <div><label class="label">Role</label><select class="input" data-field="role">
          ${['admin','user','readonly'].map(r => `<option value="${r}" ${u.role===r?'selected':''}>${r}</option>`).join('')}
        </select></div>
        <div class="flex items-center gap-2">
          <label class="label mb-0">Active</label>
          <input type="checkbox" id="user-active-toggle" ${u.active ? 'checked' : ''} ${isSelf ? 'disabled' : ''}>
          ${isSelf ? '<span class="text-xs text-gray-400">(cannot deactivate yourself)</span>' : ''}
        </div>
        <hr class="my-2">
        <div><label class="label">Reset Password</label><input class="input" type="password" id="user-new-password" placeholder="Leave blank to keep current"></div>
      </div>`, async (overlay) => {
        const v = getModalValues(overlay);
        const active = overlay.querySelector('#user-active-toggle').checked ? 1 : 0;
        const btn = overlay.querySelector('#modal-save');
        btn.disabled = true; btn.innerHTML = '<svg class="animate-spin h-4 w-4 inline mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg> Saving...';
        try {
          await api('PUT', 'users/' + id, { display_name: v.display_name, role: v.role, active });
          const newPw = overlay.querySelector('#user-new-password').value;
          if (newPw) {
            await api('PUT', 'users/' + id + '/password', { password: newPw });
          }
          toast('User updated'); overlay.remove(); load();
        } catch(e) { toast(e.message, 'error'); } finally { btn.disabled = false; btn.textContent = 'Save'; }
      });
    };

    load();
  }
};
