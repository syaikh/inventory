<script>
import { onMount } from 'svelte';

const API = '';
let currentPage = 'dashboard';
let scanMode = 'in';
let liveLog = [];
let stats = { total_items: 0, total_units: 0, out_of_stock: 0, scans_today: 0 };
let items = [];
let categories = [];
let lowStock = [];
let scanHistory = [];
let history = [];
let barcodeInput = '';
let scanQty = 1;
let histBarcode = '';
let itemModalOpen = false;
let editingItem = null;
let toastVisible = false;
let toastMsg = '';
let toastError = false;

let fBarcode = '';
let fName = '';
let fSKU = '';
let fCategory = '';
let fLocation = '';
let fQuantity = 0;
let fUnit = 'pcs';
let fPrice = 0;

function showToast(message, isError = false) {
  toastMsg = message;
  toastError = isError;
  toastVisible = true;
  setTimeout(() => (toastVisible = false), 1800);
}

function setPage(page) {
  currentPage = page;
  if (page === 'dashboard') loadDashboard();
  else if (page === 'items') {
    loadItems();
    loadCategories();
  }
  else if (page === 'scanner') loadScanHistory();
  else if (page === 'history') loadHistory();
}

async function refreshStats() {
  const res = await fetch(API + '/api/stats');
  if (!res.ok) return;
  stats = await res.json();
}

async function loadDashboard() {
  await refreshStats();
  const res = await fetch(API + '/api/items');
  if (!res.ok) return;
  const allItems = await res.json();
  lowStock = allItems
    .filter(i => i.quantity <= 5)
    .sort((a,b) => a.quantity - b.quantity)
    .slice(0, 10);
}

async function loadItems() {
  const search = document.getElementById('item-search').value;
  const cat = document.getElementById('cat-filter').value;
  const res = await fetch(`${API}/api/items?search=${encodeURIComponent(search)}&category=${encodeURIComponent(cat)}`);
  if (!res.ok) return;
  items = await res.json();
}

async function loadCategories() {
  const res = await fetch(API + '/api/categories');
  if (!res.ok) return;
  categories = await res.json();
}

function openItemModal(item) {
  if (item) {
    editingItem = item;
    fBarcode = item.barcode;
    fName = item.name;
    fSKU = item.sku;
    fCategory = item.category;
    fLocation = item.location;
    fQuantity = item.quantity;
    fUnit = item.unit;
    fPrice = item.price;
  } else {
    editingItem = null;
    fBarcode = '';
    fName = '';
    fSKU = '';
    fCategory = '';
    fLocation = '';
    fQuantity = 0;
    fUnit = 'pcs';
    fPrice = 0;
  }
  itemModalOpen = true;
}

function closeModal() {
  itemModalOpen = false;
}

async function saveItem() {
  if (!fBarcode.trim() || !fName.trim()) {
    showToast('Barcode and name are required', true);
    return;
  }
  const body = {
    barcode: fBarcode.trim(), name: fName.trim(), sku: fSKU.trim(), category: fCategory.trim(),
    location: fLocation.trim(), quantity: Number(fQuantity) || 0, unit: fUnit.trim() || 'pcs', price: Number(fPrice) || 0
  };
  const res = await fetch(API + '/api/items', {
    method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)
  });
  if (!res.ok) { showToast('Item save failed', true); return; }
  closeModal();
  loadItems(); loadCategories(); refreshStats();
  showToast('Item saved');
}

async function deleteItem(id) {
  if (!confirm('Delete this item?')) return;
  const res = await fetch(API + '/api/items/' + id, { method: 'DELETE' });
  if (!res.ok) { showToast('Failed to delete', true); return; }
  loadItems(); refreshStats(); showToast('Item deleted');
}

async function submitScan() {
  if (!barcodeInput.trim()) return;
  const res = await fetch(API + '/api/scan', {
    method: 'POST', headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ barcode: barcodeInput.trim(), mode: scanMode, qty: Number(scanQty) || 1 })
  });
  if (!res.ok) { showToast('Scan failed', true); return; }
  const item = await res.json();
  barcodeInput = '';
  if (item && item.barcode) {
    showToast(`${scanMode === 'in' ? '▲ Scanned IN' : '▼ Scanned OUT'}: ${item.name}`);
  }
  loadScanHistory();
}

async function loadScanHistory() {
  const res = await fetch(API + '/api/transactions?limit=20');
  if (!res.ok) return;
  scanHistory = await res.json();
}

async function loadHistory() {
  const url = histBarcode.trim() ? `${API}/api/transactions?barcode=${encodeURIComponent(histBarcode.trim())}&limit=100` : `${API}/api/transactions?limit=100`;
  const res = await fetch(url);
  if (!res.ok) return;
  history = await res.json();
}

function setMode(m) {
  scanMode = m;
}

function addLiveLog(ev) {
  liveLog = [{ ...ev, time: new Date().toLocaleTimeString() }, ...liveLog].slice(0, 30);
}

onMount(() => {
  setPage('dashboard');
  const source = new EventSource(API + '/api/events');
  source.onmessage = event => {
    const ev = JSON.parse(event.data);
    addLiveLog(ev);
    refreshStats();
    if (currentPage === 'scanner') loadScanHistory();
    if (currentPage === 'history') loadHistory();
    if (currentPage === 'items') loadItems();
  };
});
</script>

<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  :root { --bg: #f9f9f8; --surface: #fff; --border: #e4e2da; --text: #1a1a18; --muted: #6b6b66; --accent: #1a6ef7; --green: #1d9e75; --red: #d85a30; --amber: #ba7517; --radius: 8px; --sidebar: 220px; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: var(--bg); color: var(--text); font-size: 14px; display: flex; min-height: 100vh; margin:0; }
  aside { width: var(--sidebar); background: var(--surface); border-right: 1px solid var(--border); display: flex; flex-direction: column; padding: 1.5rem 0; position: fixed; top: 0; bottom: 0; left: 0; }
  .logo { padding: 0 1.25rem 1.5rem; font-size: 16px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
  .logo svg { flex-shrink: 0; }
  nav a { display: flex; align-items: center; gap: 10px; padding: 9px 1.25rem; color: var(--muted); text-decoration: none; border-left: 3px solid transparent; transition: all .15s; cursor:pointer; }
  nav a.active { color: var(--accent); border-left-color: var(--accent); background: #eef4ff; }
  main { margin-left: var(--sidebar); flex: 1; padding: 2rem; max-width: 1100px; }
  .page { display: none; }
  .page.active { display: block; }
  h1 { font-size: 20px; font-weight: 600; margin-bottom: 1.5rem; }
  .stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 2rem; }
  .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 1rem 1.25rem; }
  .stat-card .label { font-size: 12px; color: var(--muted); margin-bottom: 4px; }
  .stat-card .value { font-size: 24px; font-weight: 600; }
  .scan-panel { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 1.5rem; margin-bottom: 1.5rem; }
  .scan-row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .scan-row input[type=text], .scan-row input[type=number] { flex: 1; min-width: 200px; padding: 8px 12px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 14px; background: var(--bg); }
  .mode-toggle { display: flex; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
  .mode-toggle button { padding: 8px 16px; border: none; background: none; cursor: pointer; font-size: 13px; color: var(--muted); transition: all .15s; }
  .mode-toggle button.active { background: var(--accent); color: #fff; }
  .btn { padding: 8px 16px; border-radius: var(--radius); border: 1px solid var(--border); cursor: pointer; font-size: 13px; font-weight: 500; transition: all .15s; background: var(--surface); }
  .btn:hover { background: var(--bg); }
  .btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); }
  .btn-primary:hover { background: #155fd4; }
  .btn-danger { background: var(--red); color: #fff; border-color: var(--red); }
  .btn-danger:hover { background: #b84a22; }
  .btn-sm { padding: 5px 10px; font-size: 12px; }
  .scan-log { max-height: 160px; overflow-y: auto; border: 1px solid var(--border); border-radius: var(--radius); background: #fafaf8; }
  .scan-log-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid var(--border); font-size: 13px; }
  .scan-log-item:last-child { border-bottom: none; }
  .badge { padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
  .badge-in { background: #e1f5ee; color: #0f6e56; }
  .badge-out { background: #faece7; color: #993c1d; }
  .badge-warn { background: #faeeda; color: #854f0b; }
  .item-preview { margin-top: 1rem; padding: 12px; background: var(--bg); border-radius: var(--radius); border: 1px solid var(--border); display: none; }
  .item-preview.show { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
  .item-preview .name { font-weight: 600; }
  .item-preview .meta { font-size: 12px; color: var(--muted); margin-top: 2px; }
  .item-preview .qty-big { font-size: 22px; font-weight: 700; }
  .toolbar { display: flex; gap: 8px; margin-bottom: 1rem; flex-wrap: wrap; align-items: center; }
  .toolbar input[type=text], .toolbar select { flex: 1; min-width: 180px; padding: 7px 12px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; background: var(--surface); }
  .table-wrap { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
  table { width: 100%; border-collapse: collapse; }
  thead { background: var(--bg); }
  th { padding: 10px 14px; text-align: left; font-size: 12px; color: var(--muted); font-weight: 500; border-bottom: 1px solid var(--border); text-transform: uppercase; letter-spacing: .04em; }
  td { padding: 10px 14px; border-bottom: 1px solid #f0ede6; font-size: 13px; }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #fafaf8; }
  .qty-low { color: var(--red); font-weight: 600; }
  .qty-ok  { color: var(--green); font-weight: 600; }
  .overlay { position: fixed; inset: 0; background: rgba(0,0,0,.35); display: none; align-items: center; justify-content: center; z-index: 100; }
  .overlay.show { display: flex; }
  .modal { background: var(--surface); border-radius: 12px; padding: 1.5rem; width: 100%; max-width: 460px; border: 1px solid var(--border); }
  .modal h2 { font-size: 16px; font-weight: 600; margin-bottom: 1.25rem; }
  .form-group { margin-bottom: 12px; }
  label { display: block; font-size: 12px; color: var(--muted); margin-bottom: 4px; }
  input, select, textarea { width: 100%; padding: 8px 10px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; background: var(--bg); }
  input:focus, select:focus { outline: 2px solid var(--accent); outline-offset: -1px; }
  .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 1.25rem; }
  .toast { position: fixed; bottom: 1.5rem; right: 1.5rem; background: var(--text); color: #fff; padding: 10px 18px; border-radius: var(--radius); font-size: 13px; opacity: 0; transition: opacity .25s; z-index: 999; pointer-events: none; }
  .toast.show { opacity: 1; }
  .empty { text-align: center; padding: 3rem 1rem; color: var(--muted); }
  .indicator { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-right: 4px; }
  .indicator.live { background: var(--green); animation: pulse 1.5s infinite; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }
</style>

<aside>
  <div class="logo">
    <svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg" width="22" height="22">
      <rect x="2" y="4" width="16" height="12" rx="2" stroke="#1a6ef7" stroke-width="1.5"/>
      <path d="M6 8h8M6 12h5" stroke="#1a6ef7" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
    Inventory
  </div>
  <nav>
    <a class={currentPage==='dashboard' ? 'active' : ''} on:click={() => setPage('dashboard')}><span>Dashboard</span></a>
    <a class={currentPage==='items' ? 'active' : ''} on:click={() => setPage('items')}><span>Items</span></a>
    <a class={currentPage==='scanner' ? 'active' : ''} on:click={() => setPage('scanner')}><span>Scanner</span></a>
    <a class={currentPage==='history' ? 'active' : ''} on:click={() => setPage('history')}><span>History</span></a>
  </nav>
</aside>

<main>
  <section class="page" class:active={currentPage==='dashboard'}>
    <h1>Dashboard</h1>
    <div class="stats-grid">
      <div class="stat-card"><div class="label">Total Items</div><div class="value">{stats.total_items}</div></div>
      <div class="stat-card"><div class="label">Total Units</div><div class="value">{stats.total_units}</div></div>
      <div class="stat-card"><div class="label">Out of Stock</div><div class="value">{stats.out_of_stock}</div></div>
      <div class="stat-card"><div class="label">Scans Today</div><div class="value">{stats.scans_today}</div></div>
    </div>
    <div class="scan-panel">
      <h2><span class="indicator live"></span> Live Scan Feed</h2>
      <div id="live-log" class="scan-log">
        {#if liveLog.length===0}
          <div class="empty" style="padding:1rem">Waiting for scans…</div>
        {:else}
          {#each liveLog as ev}
            <div class="scan-log-item"><span>{ev.mode==='in' ? '▲' : '▼'} <strong>{ev.barcode}</strong></span><span style="color:var(--muted)">{ev.time}</span></div>
          {/each}
        {/if}
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>Barcode</th><th>Name</th><th>Category</th><th>Stock</th><th>Unit</th></tr></thead>
        <tbody>
          {#if lowStock.length===0}
            <tr><td colspan="5" class="empty">All items have sufficient stock</td></tr>
          {:else}
            {#each lowStock as i}
              <tr>
                <td><code>{i.barcode}</code></td><td>{i.name}</td><td>{i.category||'—'}</td>
                <td class={i.quantity===0 ? 'qty-low' : 'qty-ok'}>{i.quantity}</td><td>{i.unit}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </section>

  <section class="page" class:active={currentPage==='items'}>
    <h1>Items</h1>
    <div class="toolbar">
      <input id="item-search" type="text" placeholder="Search name, barcode, SKU…" on:input={loadItems} />
      <select id="cat-filter" on:change={loadItems}>
        <option value="">All categories</option>
        {#each categories as c}<option value={c}>{c}</option>{/each}
      </select>
      <button class="btn btn-primary" on:click={() => openItemModal(null)}>+ Add Item</button>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>Barcode</th><th>Name</th><th>SKU</th><th>Category</th><th>Stock</th><th>Unit</th><th>Price</th><th>Location</th><th></th></tr></thead>
        <tbody>
          {#if items.length===0}
            <tr><td colspan="9" class="empty">No items found</td></tr>
          {:else}
            {#each items as i}
              <tr>
                <td><code>{i.barcode}</code></td><td>{i.name}</td><td>{i.sku||'—'}</td><td>{i.category||'—'}</td>
                <td class={i.quantity===0 ? 'qty-low' : 'qty-ok'}>{i.quantity}</td><td>{i.unit}</td>
                <td>{i.price ? `$${i.price.toFixed(2)}` : '—'}</td><td>{i.location||'—'}</td>
                <td><button class="btn btn-sm" on:click={() => openItemModal(i)}>Edit</button>
                    <button class="btn btn-sm btn-danger" on:click={() => deleteItem(i.id)}>Del</button></td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </section>

  <section class="page" class:active={currentPage==='scanner'}>
    <h1>Scanner</h1>
    <div class="scan-panel">
      <h2>Manual Scan / Quick Entry</h2>
      <p style="color:var(--muted);font-size:12px;margin-bottom:1rem">USB barcode scanners work automatically — just scan any item. Use this form for manual entry or override quantity.</p>
      <div class="scan-row">
        <input type="text" bind:value={barcodeInput} placeholder="Scan or type barcode…" on:keydown={(e)=>e.key==='Enter' && submitScan()} />
        <div class="mode-toggle"><button class:active={scanMode==='in'} on:click={()=>setMode('in')}>▲ IN</button><button class:active={scanMode==='out'} on:click={()=>setMode('out')}>▼ OUT</button></div>
        <input type="number" class="qty-input" bind:value={scanQty} min="1" />
        <button class="btn btn-primary" on:click={submitScan}>Scan</button>
      </div>
      {#if false}<div class="item-preview" class:show={false}></div>{/if}
    </div>
    <h2 style="font-size:15px;font-weight:600;margin-bottom:.75rem">Recent Scans</h2>
    <div class="table-wrap"><table><thead><tr><th>Time</th><th>Barcode</th><th>Item</th><th>Type</th><th>Qty</th></tr></thead><tbody>
      {#if scanHistory.length===0}<tr><td colspan="5" class="empty">No scans yet</td></tr>{:else}{#each scanHistory as t}
        <tr><td style="color:var(--muted)">{new Date(t.created_at).toLocaleString()}</td><td><code>{t.barcode}</code></td><td>{t.item_name}</td><td>{@html t.type === 'scan_in' ? '<span class="badge badge-in">IN</span>' : t.type === 'scan_out' ? '<span class="badge badge-out">OUT</span>' : t.type}</td><td>{t.quantity}</td></tr>
      {/each}{/if}
    </tbody></table></div>
  </section>

  <section class="page" class:active={currentPage==='history'}>
    <h1>Transaction History</h1>
    <div class="toolbar"><input type="text" bind:value={histBarcode} placeholder="Filter by barcode…" on:input={loadHistory} /><button class="btn" on:click={loadHistory}>Refresh</button></div>
    <div class="table-wrap"><table><thead><tr><th>Time</th><th>Barcode</th><th>Item</th><th>Type</th><th>Qty</th><th>Note</th></tr></thead><tbody>
      {#if history.length===0}<tr><td colspan="6" class="empty">No transactions</td></tr>{:else}{#each history as t}
        <tr><td style="color:var(--muted)">{new Date(t.created_at).toLocaleString()}</td><td><code>{t.barcode}</code></td><td>{t.item_name}</td><td>{@html t.type === 'scan_in' ? '<span class="badge badge-in">IN</span>' : t.type === 'scan_out' ? '<span class="badge badge-out">OUT</span>' : t.type}</td><td>{t.quantity}</td><td style="color:var(--muted)">{t.note || '—'}</td></tr>
      {/each}{/if}
    </tbody></table></div>
  </section>
</main>

{#if itemModalOpen}
  <div class="overlay show"><div class="modal"><h2>{editingItem ? 'Edit Item' : 'Add Item'}</h2>
    <div class="form-row"><div class="form-group"><label>Barcode *</label><input type="text" bind:value={fBarcode} /></div><div class="form-group"><label>SKU</label><input type="text" bind:value={fSKU} /></div></div>
    <div class="form-group"><label>Name *</label><input type="text" bind:value={fName} /></div>
    <div class="form-row"><div class="form-group"><label>Category</label><input type="text" bind:value={fCategory} /></div><div class="form-group"><label>Location</label><input type="text" bind:value={fLocation} /></div></div>
    <div class="form-row"><div class="form-group"><label>Initial Quantity</label><input type="number" bind:value={fQuantity} min="0" /></div><div class="form-group"><label>Unit</label><input type="text" bind:value={fUnit} /></div></div>
    <div class="form-group"><label>Price</label><input type="number" bind:value={fPrice} min="0" step="0.01" /></div>
    <div class="modal-actions"><button class="btn" on:click={closeModal}>Cancel</button><button class="btn btn-primary" on:click={saveItem}>Save</button></div>
  </div></div>
{/if}

<div class="toast" class:show={toastVisible} style="background: {toastError ? 'var(--red)' : 'var(--text)'};">{toastMsg}</div>
