window.module_costing = {
  render: async (container) => {
    container.innerHTML = `<div class="card">
      <h2 class="text-lg font-semibold mb-4">Project Costing</h2>
      <p class="text-gray-500">BOM cost rollup is available through the Quotes and Work Orders modules. Select a quote to view its cost breakdown, or a work order to see BOM requirements vs. inventory.</p>
      <div class="grid grid-cols-2 gap-4 mt-4">
        <div class="card cursor-pointer hover:shadow-md" onclick="navigate('quotes')">
          <h3 class="font-semibold">💰 Quotes</h3>
          <p class="text-sm text-gray-500 mt-1">View quote cost rollups</p>
        </div>
        <div class="card cursor-pointer hover:shadow-md" onclick="navigate('workorders')">
          <h3 class="font-semibold">⚙️ Work Orders</h3>
          <p class="text-sm text-gray-500 mt-1">View BOM requirements</p>
        </div>
      </div>
    </div>`;
  }
};
