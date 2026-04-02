<script>
  import { stats } from '$lib/stores.js';
</script>

<div class="stats-grid">
  <div class="stat-card">
    <div class="label">Total Items</div>
    <div class="value">{$stats.total_items || 0}</div>
  </div>
  <div class="stat-card">
    <div class="label">Total Units</div>
    <div class="value">{$stats.total_units || 0}</div>
  </div>
  <div class="stat-card">
    <div class="label">Out of Stock</div>
    <div class="value" style="color:var(--red)">{$stats.out_of_stock || 0}</div>
  </div>
  <div class="stat-card">
    <div class="label">Total Categories</div>
    <div class="value">{$stats.total_categories || 0}</div>
  </div>
  <div class="stat-card">
    <div class="label">Scans Today</div>
    <div class="value">{$stats.scans_today || 0}</div>
  </div>
</div>

<style>
  .stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 2rem; }
  .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 1rem 1.25rem; }
  .stat-card .label { font-size: 12px; color: var(--muted); margin-bottom: 4px; }
  .stat-card .value { font-size: 24px; font-weight: 600; }
</style>
