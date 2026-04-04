<script>
  import { history } from '$lib/stores.js';
  import { fetchHistory } from '$lib/api.js';
  import { onMount } from 'svelte';

  let barcodeFilter = '';

  onMount(async () => {
    await loadHistory();
  });

  async function loadHistory() {
    const txs = await fetchHistory(barcodeFilter, 100);
    if (txs) history.set(txs);
  }
</script>

<h1>Transaction History</h1>
<div class="toolbar">
  <input 
    type="text" 
    bind:value={barcodeFilter} 
    placeholder="Filter by barcode…" 
    on:input={loadHistory}
  />
  <button class="btn" on:click={loadHistory}>Refresh</button>
</div>

<div class="table-wrap">
  <table>
    <thead>
      <tr>
        <th>Time</th>
        <th>Barcode</th>
        <th>Product</th>
        <th>Type</th>
        <th>Qty</th>
        <th>Note</th>
      </tr>
    </thead>
    <tbody>
      {#if $history.length === 0}
        <tr>
          <td colspan="6" class="empty">No transactions</td>
        </tr>
      {:else}
        {#each $history as t}
          <tr>
            <td style="color:var(--muted)">{new Date(t.created_at).toLocaleString()}</td>
            <td><code>{t.barcode}</code></td>
            <td>{t.product_name}</td>
            <td>
              {#if t.type === 'scan_in'}
                <span class="badge badge-in">IN</span>
              {:else if t.type === 'scan_out'}
                <span class="badge badge-out">OUT</span>
              {:else}
                {t.type}
              {/if}
            </td>
            <td>{t.quantity}</td>
            <td style="color:var(--muted)">{t.note || '—'}</td>
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>

<style>
  h1 { font-size: 20px; font-weight: 600; margin-bottom: 1.5rem; }
  .toolbar { display: flex; gap: 8px; margin-bottom: 1rem; flex-wrap: wrap; align-items: center; }
  .toolbar input[type=text] { flex: 1; min-width: 180px; padding: 7px 12px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; background: var(--surface); }
  .btn { padding: 8px 16px; border-radius: var(--radius); border: 1px solid var(--border); cursor: pointer; font-size: 13px; font-weight: 500; transition: all .15s; background: var(--surface); }
  .btn:hover { background: var(--bg); }
  .table-wrap { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
  table { width: 100%; border-collapse: collapse; }
  thead { background: var(--bg); }
  th { padding: 10px 14px; text-align: left; font-size: 12px; color: var(--muted); font-weight: 500; border-bottom: 1px solid var(--border); text-transform: uppercase; letter-spacing: .04em; }
  td { padding: 10px 14px; border-bottom: 1px solid #f0ede6; font-size: 13px; }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #fafaf8; }
  .empty { text-align: center; padding: 3rem 1rem; color: var(--muted); }
  .badge { padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
  .badge-in { background: #e1f5ee; color: #0f6e56; }
  .badge-out { background: #faece7; color: #993c1d; }
</style>
