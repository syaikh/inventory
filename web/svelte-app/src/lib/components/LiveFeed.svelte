<script>
  import { liveLog } from '$lib/stores.js';
</script>

<div class="scan-panel">
  <h2><span class="indicator live"></span> Live Scan Feed</h2>
  <div class="scan-log">
    {#if $liveLog.length === 0}
      <div class="empty" style="padding:1rem">Waiting for scans…</div>
    {:else}
      {#each $liveLog as ev}
        <div class="scan-log-item">
          <span>{ev.mode==='in' ? '▲' : '▼'} <strong>{ev.barcode}</strong></span>
          <span style="color:var(--muted)">{ev.time}</span>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .scan-panel { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 1.5rem; margin-bottom: 1.5rem; }
  .scan-log { max-height: 160px; overflow-y: auto; border: 1px solid var(--border); border-radius: var(--radius); background: #fafaf8; }
  .scan-log-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid var(--border); font-size: 13px; }
  .scan-log-item:last-child { border-bottom: none; }
  .empty { text-align: center; padding: 3rem 1rem; color: var(--muted); }
  .indicator { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-right: 4px; }
  .indicator.live { background: var(--green); animation: pulse 1.5s infinite; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }
</style>
