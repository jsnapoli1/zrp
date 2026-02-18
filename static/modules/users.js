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
          <table class="w-full text-sm"><thead><tr class="border-b text-left text-gray-500">
            <th class="pb-2">Username</th><th class="pb-2">Display Name</th><th class="pb-2">Role</th><th class="pb-2">Status</th><th class="pb-2">Last Login</th>
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
              </tr>`;
            }).join('')}
          </tbody></table>
          ${users.length === 0 ? '<p class="text-center text-gray-400 py-4">No users</p>' : ''}
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
        try { await api('POST', 'users', v); toast('User created'); overlay.remove(); load(); } catch(e) { toast(e.message, 'error'); }
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
        try {
          await api('PUT', 'users/' + id, { display_name: v.display_name, role: v.role, active });
          const newPw = overlay.querySelector('#user-new-password').value;
          if (newPw) {
            await api('PUT', 'users/' + id + '/password', { password: newPw });
          }
          toast('User updated'); overlay.remove(); load();
        } catch(e) { toast(e.message, 'error'); }
      });
    };

    load();
  }
};
