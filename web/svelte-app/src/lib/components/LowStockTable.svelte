<script>
  import { lowStock } from '$lib/stores.js';
</script>

<div class="table-wrap">
  <table>
    <thead>
      <tr>
        <th>Barcodes</th>
        <th>Name</th>
        <th>Category</th>
        <th>Stock</th>
        <th>Unit</th>
      </tr>
    </thead>
    <tbody>
      {#if $lowStock.length === 0}
        <tr>
          <td colspan="5" class="empty">All items have sufficient stock</td>
        </tr>
      {:else}
        {#each $lowStock as i}
          <tr>
            <td><code>{i.barcodes && i.barcodes.length ? i.barcodes.join(', ') : '—'}</code></td>
            <td>{i.name}</td>
            <td>{i.category || '—'}</td>
            <td class={i.quantity===0 ? 'qty-low' : 'qty-ok'}>{i.quantity}</td>
            <td>{i.unit}</td>
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>

<style>
  .table-wrap { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
  table { width: 100%; border-collapse: collapse; }
  thead { background: var(--bg); }
  th { padding: 10px 14px; text-align: left; font-size: 12px; color: var(--muted); font-weight: 500; border-bottom: 1px solid var(--border); text-transform: uppercase; letter-spacing: .04em; }
  td { padding: 10px 14px; border-bottom: 1px solid #f0ede6; font-size: 13px; }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #fafaf8; }
  .qty-low { color: var(--red); font-weight: 600; }
  .qty-ok { color: var(--green); font-weight: 600; }
  .empty { text-align: center; padding: 3rem 1rem; color: var(--muted); }
</style>
