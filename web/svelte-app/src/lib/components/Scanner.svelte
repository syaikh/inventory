<script>
  import { scanMode, scanHistory } from '$lib/stores.js';
  import { submitScan, fetchScanHistory } from '$lib/api.js';
  import { onMount } from 'svelte';

  let barcodeInput = '';
  let scanQty = 1;

  onMount(async () => {
    const txs = await fetchScanHistory(20);
    if (txs) scanHistory.set(txs);
  });

  async function handleScan() {
    if (!barcodeInput.trim()) return;
    const product = await submitScan(barcodeInput.trim(), $scanMode, Number(scanQty) || 1);
    barcodeInput = '';
    if (product) {
      const txs = await fetchScanHistory(20);
      if (txs) scanHistory.set(txs);
      window.dispatchEvent(new CustomEvent('scanSuccess'));
    }
  }
</script>

<div class="scan-panel">
  <h2>Manual Scan / Quick Entry</h2>
  <p style="color:var(--muted);font-size:12px;margin-bottom:1rem">
    USB barcode scanners work automatically — just scan any item.
    Use this form for manual entry or to override quantity.
  </p>
  <div class="scan-row">
    <input 
      type="text" 
      bind:value={barcodeInput} 
      placeholder="Scan or type barcode…" 
      on:keydown={(e)=>e.key==='Enter' && handleScan()} 
    />
    <div class="mode-toggle">
      <button 
        class:active={$scanMode==='in'} 
        on:click={()=>scanMode.set('in')}
      >▲ IN</button>
      <button 
        class:active={$scanMode==='out'} 
        on:click={()=>scanMode.set('out')}
      >▼ OUT</button>
    </div>
    <input type="number" class="qty-input" bind:value={scanQty} min="1" />
    <button class="btn btn-primary" on:click={handleScan}>Scan</button>
  </div>
</div>

<h2 style="font-size:15px;font-weight:600;margin-bottom:.75rem">Recent Scans</h2>
<div class="table-wrap">
  <table>
    <thead>
      <tr>
        <th>Time</th>
        <th>Barcode</th>
        <th>Product</th>
        <th>Type</th>
        <th>Qty</th>
      </tr>
    </thead>
    <tbody>
      {#if $scanHistory.length === 0}
        <tr>
          <td colspan="5" class="empty">No scans yet</td>
        </tr>
      {:else}
        {#each $scanHistory as t}
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
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>

<style>
  .scan-panel { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 1.5rem; margin-bottom: 1.5rem; }
  .scan-row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .scan-row input[type=text], .scan-row input[type=number] { flex: 1; min-width: 200px; padding: 8px 12px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 14px; background: var(--bg); }
  .mode-toggle { display: flex; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
  .mode-toggle button { padding: 8px 16px; border: none; background: none; cursor: pointer; font-size: 13px; color: var(--muted); transition: all .15s; }
  .mode-toggle button.active { background: var(--accent); color: #fff; }
  .qty-input { width: 64px; padding: 8px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 14px; text-align: center; }
  .btn { padding: 8px 16px; border-radius: var(--radius); border: 1px solid var(--border); cursor: pointer; font-size: 13px; font-weight: 500; transition: all .15s; background: var(--surface); }
  .btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); }
  .btn-primary:hover { background: #155fd4; }
  .table-wrap { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
  table { width: 100%; border-collapse: collapse; }
  thead { background: var(--bg); }
  th { padding: 10px 14px; text-align: left; font-size: 12px; color: var(--muted); font-weight: 500; border-bottom: 1px solid var(--border); text-transform: uppercase; letter-spacing: .04em; }
  td { padding: 10px 14px; border-bottom: 1px solid #f0ede6; font-size: 13px; }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #fafaf8; }
  .badge { padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
  .badge-in { background: #e1f5ee; color: #0f6e56; }
  .badge-out { background: #faece7; color: #993c1d; }
  .empty { text-align: center; padding: 3rem 1rem; color: var(--muted); }
</style>
