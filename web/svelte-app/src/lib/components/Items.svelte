<script>
  import { items, categories } from '$lib/stores.js';
  import { fetchItems, fetchCategories, deleteItemById } from '$lib/api.js';
  import { onMount } from 'svelte';

  let searchInput = '';
  let selectedCategory = '';

  onMount(async () => {
    await loadData();

    const refresh = async () => {
      await loadData();
    };

    window.addEventListener('itemSaved', refresh);
    window.addEventListener('itemDeleted', refresh);

    return () => {
      window.removeEventListener('itemSaved', refresh);
      window.removeEventListener('itemDeleted', refresh);
    };
  });

  async function loadData() {
    const cats = await fetchCategories();
    if (cats) categories.set(cats);
    await loadItems();
  }

  async function loadItems() {
    const itemList = await fetchItems(searchInput, selectedCategory);
    if (itemList) items.set(itemList);
  }

  async function handleDelete(id) {
    if (!confirm('Delete this item?')) return;
    const success = await deleteItemById(id);
    if (success) {
      await loadData();
      window.dispatchEvent(new CustomEvent('itemDeleted'));
    }
  }

  export async function openAddModal() {
    window.dispatchEvent(new CustomEvent('openItemModal', { detail: null }));
  }

  export async function openEditModal(item) {
    window.dispatchEvent(new CustomEvent('openItemModal', { detail: item }));
  }
</script>

<h1>Items</h1>
<div class="toolbar">
  <input 
    type="text" 
    bind:value={searchInput} 
    placeholder="Search name, barcode, SKU…" 
    on:input={loadItems}
  />
  <select bind:value={selectedCategory} on:change={loadItems}>
    <option value="">All categories</option>
    {#each $categories as c}
      <option value={c}>{c}</option>
    {/each}
  </select>
  <button class="btn btn-primary" on:click={openAddModal}>+ Add Item</button>
</div>

<div class="table-wrap">
  <table>
    <thead>
      <tr>
        <th>Barcode</th>
        <th>Name</th>
        <th>SKU</th>
        <th>Category</th>
        <th>Stock</th>
        <th>Unit</th>
        <th>Price</th>
        <th>Location</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#if $items.length === 0}
        <tr>
          <td colspan="9" class="empty">No items found</td>
        </tr>
      {:else}
        {#each $items as i}
          <tr>
            <td><code>{i.barcode}</code></td>
            <td>{i.name}</td>
            <td>{i.sku || '—'}</td>
            <td>{i.category || '—'}</td>
            <td class={i.quantity===0 ? 'qty-low' : 'qty-ok'}>{i.quantity}</td>
            <td>{i.unit}</td>
            <td>{i.price ? `$${i.price.toFixed(2)}` : '—'}</td>
            <td>{i.location || '—'}</td>
            <td>
              <button class="btn btn-sm" on:click={() => openEditModal(i)}>Edit</button>
              <button class="btn btn-sm btn-danger" on:click={() => handleDelete(i.id)}>Del</button>
            </td>
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>

<style>
  h1 { font-size: 20px; font-weight: 600; margin-bottom: 1.5rem; }
  .toolbar { display: flex; gap: 8px; margin-bottom: 1rem; flex-wrap: wrap; align-items: center; }
  .toolbar input[type=text], .toolbar select { flex: 1; min-width: 180px; padding: 7px 12px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; background: var(--surface); }
  .btn { padding: 8px 16px; border-radius: var(--radius); border: 1px solid var(--border); cursor: pointer; font-size: 13px; font-weight: 500; transition: all .15s; background: var(--surface); }
  .btn:hover { background: var(--bg); }
  .btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); }
  .btn-primary:hover { background: #155fd4; }
  .btn-sm { padding: 5px 10px; font-size: 12px; }
  .btn-danger { background: var(--red); color: #fff; border-color: var(--red); }
  .btn-danger:hover { background: #b84a22; }
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
