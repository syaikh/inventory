<script>
  import { itemModalOpen, editingItem, toastVisible, toastMsg, toastError } from '$lib/stores.js';
  import { saveProduct } from '$lib/api.js';

  let fBarcodes = '';
  let fName = '';
  let fSKU = '';
  let fCategory = '';
  let fLocation = '';
  let fQuantity = 0;
  let fUnit = 'pcs';
  let fPrice = 0;

  function showToast(message, isError = false) {
    toastMsg.set(message);
    toastError.set(isError);
    toastVisible.set(true);
    setTimeout(() => toastVisible.set(false), 1800);
  }

  export async function openModal(product = null) {
    if (product) {
      editingItem.set(product);
      fBarcodes = product.barcodes ? product.barcodes.join(', ') : '';
      fName = product.name;
      fSKU = product.sku;
      fCategory = product.category;
      fLocation = product.location;
      fQuantity = product.quantity;
      fUnit = product.unit;
      fPrice = product.price;
    } else {
      editingItem.set(null);
      fBarcodes = '';
      fName = '';
      fSKU = '';
      fCategory = '';
      fLocation = '';
      fQuantity = 0;
      fUnit = 'pcs';
      fPrice = 0;
    }
    itemModalOpen.set(true);
  }

  function closeModal() {
    itemModalOpen.set(false);
  }

  async function handleSave() {
    if (!fName.trim()) {
      showToast('Name is required', true);
      return;
    }
    
    // Parse comma-separated barcodes into an array
    const barcodesArray = fBarcodes.split(',').map(b => b.trim()).filter(b => b.length > 0);

    const body = {
      barcodes: barcodesArray,
      name: fName.trim(),
      sku: fSKU.trim(),
      category: fCategory.trim(),
      location: fLocation.trim(),
      quantity: Number(fQuantity) || 0,
      unit: fUnit.trim() || 'pcs',
      price: Number(fPrice) || 0
    };
    
    // If we're editing, we should send the ID
    if ($editingItem && $editingItem.id) {
      body.id = $editingItem.id;
    }

    const result = await saveProduct(body);
    if (!result) {
      showToast('Failed to save product', true);
      return;
    }
    closeModal();
    showToast('Product saved');
    window.dispatchEvent(new CustomEvent('itemSaved'));
  }
</script>

{#if $itemModalOpen}
  <div class="overlay show">
    <div class="modal">
      <h2>{$editingItem ? 'Edit Product' : 'Add Product'}</h2>
      <div class="form-row">
        <div class="form-group">
          <label for="f-barcode">Barcodes (Comma Separated)</label>
          <input id="f-barcode" type="text" bind:value={fBarcodes} placeholder="e.g. 123, 456" />
        </div>
        <div class="form-group">
          <label for="f-sku">SKU</label>
          <input id="f-sku" type="text" bind:value={fSKU} placeholder="e.g. PROD-001" />
        </div>
      </div>
      <div class="form-group">
        <label for="f-name">Name *</label>
        <input id="f-name" type="text" bind:value={fName} placeholder="Product name" />
      </div>
      <div class="form-row">
        <div class="form-group">
          <label for="f-category">Category</label>
          <input id="f-category" type="text" bind:value={fCategory} placeholder="e.g. Electronics" />
        </div>
        <div class="form-group">
          <label for="f-location">Location</label>
          <input id="f-location" type="text" bind:value={fLocation} placeholder="e.g. A-12" />
        </div>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label for="f-quantity">Initial Quantity</label>
          <input id="f-quantity" type="number" bind:value={fQuantity} min="0" />
        </div>
        <div class="form-group">
          <label for="f-unit">Unit</label>
          <input id="f-unit" type="text" bind:value={fUnit} />
        </div>
      </div>
      <div class="form-group">
        <label for="f-price">Price</label>
        <input id="f-price" type="number" bind:value={fPrice} step="0.01" min="0" />
      </div>
      <div class="modal-actions">
        <button class="btn" on:click={closeModal}>Cancel</button>
        <button class="btn btn-primary" on:click={handleSave}>Save</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay { position: fixed; inset: 0; background: rgba(0,0,0,.35); display: none; align-items: center; justify-content: center; z-index: 100; }
  .overlay.show { display: flex; }
  .modal { background: var(--surface); border-radius: 12px; padding: 1.5rem; width: 100%; max-width: 460px; border: 1px solid var(--border); }
  .modal h2 { font-size: 16px; font-weight: 600; margin-bottom: 1.25rem; }
  .form-group { margin-bottom: 12px; }
  label { display: block; font-size: 12px; color: var(--muted); margin-bottom: 4px; }
  input { width: 100%; padding: 8px 10px; border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; background: var(--bg); }
  input:focus { outline: 2px solid var(--accent); outline-offset: -1px; }
  .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 1.25rem; }
  .btn { padding: 8px 16px; border-radius: var(--radius); border: 1px solid var(--border); cursor: pointer; font-size: 13px; font-weight: 500; transition: all .15s; background: var(--surface); }
  .btn:hover { background: var(--bg); }
  .btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); }
  .btn-primary:hover { background: #155fd4; }
</style>
