window.module_calendar = {
  render: async (container) => {
    let currentYear = new Date().getFullYear();
    let currentMonth = new Date().getMonth() + 1;

    async function load() {
      const res = await api('GET', `calendar?year=${currentYear}&month=${currentMonth}`);
      const events = res.data || [];
      const today = new Date();
      const todayStr = today.toISOString().slice(0, 10);

      const monthNames = ['January','February','March','April','May','June','July','August','September','October','November','December'];
      const dayNames = ['Mon','Tue','Wed','Thu','Fri','Sat','Sun'];

      // Build calendar grid
      const firstDay = new Date(currentYear, currentMonth - 1, 1);
      const lastDay = new Date(currentYear, currentMonth, 0);
      const daysInMonth = lastDay.getDate();
      // Monday=0..Sunday=6
      let startDow = firstDay.getDay() - 1;
      if (startDow < 0) startDow = 6;

      // Group events by date
      const eventsByDate = {};
      for (const ev of events) {
        if (!eventsByDate[ev.date]) eventsByDate[ev.date] = [];
        eventsByDate[ev.date].push(ev);
      }

      let cells = '';
      // Empty cells before first day
      for (let i = 0; i < startDow; i++) {
        cells += `<div class="min-h-[80px] border border-gray-100 dark:border-gray-700 bg-gray-50 dark:bg-gray-900"></div>`;
      }
      for (let d = 1; d <= daysInMonth; d++) {
        const dateStr = `${currentYear}-${String(currentMonth).padStart(2,'0')}-${String(d).padStart(2,'0')}`;
        const isToday = dateStr === todayStr;
        const dayEvents = eventsByDate[dateStr] || [];
        const pills = dayEvents.map(ev => {
          const colorMap = {blue:'bg-blue-500',green:'bg-green-500',orange:'bg-orange-500',red:'bg-red-500',purple:'bg-purple-500'};
          const bg = colorMap[ev.color] || 'bg-gray-500';
          return `<div class="${bg} text-white text-xs px-1.5 py-0.5 rounded-full truncate cursor-pointer hover:opacity-80" 
            onclick="event.stopPropagation(); window._calNav('${ev.type}','${ev.id}')" 
            title="${ev.title}">${ev.title}</div>`;
        }).join('');

        cells += `<div class="min-h-[80px] border border-gray-100 dark:border-gray-700 p-1">
          <div class="text-sm font-medium ${isToday ? 'inline-flex items-center justify-center w-7 h-7 rounded-full ring-2 ring-blue-500 text-blue-600' : 'text-gray-700 dark:text-gray-300'}">${d}</div>
          <div class="space-y-0.5 mt-0.5">${pills}</div>
        </div>`;
      }

      container.innerHTML = `<div class="card">
        <div class="flex justify-between items-center mb-4">
          <button class="btn btn-secondary" onclick="window._calPrev()">← Prev</button>
          <h2 class="text-lg font-semibold">${monthNames[currentMonth-1]} ${currentYear}</h2>
          <button class="btn btn-secondary" onclick="window._calNext()">Next →</button>
        </div>
        <div class="grid grid-cols-7 gap-0">
          ${dayNames.map(d => `<div class="text-center text-xs font-semibold text-gray-500 py-2 border-b border-gray-200 dark:border-gray-700">${d}</div>`).join('')}
          ${cells}
        </div>
        <div class="flex gap-4 mt-3 text-xs text-gray-500">
          <span><span class="inline-block w-3 h-3 rounded-full bg-blue-500 mr-1"></span>Work Orders</span>
          <span><span class="inline-block w-3 h-3 rounded-full bg-green-500 mr-1"></span>Purchase Orders</span>
          <span><span class="inline-block w-3 h-3 rounded-full bg-orange-500 mr-1"></span>Quotes</span>
        </div>
      </div>`;
    }

    window._calPrev = () => { currentMonth--; if (currentMonth < 1) { currentMonth = 12; currentYear--; } load(); };
    window._calNext = () => { currentMonth++; if (currentMonth > 12) { currentMonth = 1; currentYear++; } load(); };
    window._calNav = (type, id) => {
      const routes = {workorder:'workorders', po:'procurement', quote:'quotes'};
      navigate(routes[type] || type);
    };
    load();
  }
};
