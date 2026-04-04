const API = '';

export async function fetchStats() {
  const res = await fetch(API + '/api/stats');
  if (!res.ok) return null;
  return await res.json();
}

export async function fetchProducts(search = '', category = '') {
  const res = await fetch(`${API}/api/products?search=${encodeURIComponent(search)}&category=${encodeURIComponent(category)}`);
  if (!res.ok) return null;
  return await res.json();
}

export async function fetchCategories() {
  const res = await fetch(API + '/api/categories');
  if (!res.ok) return null;
  return await res.json();
}

export async function fetchLowStock(limit = 10) {
  const res = await fetch(API + '/api/products');
  if (!res.ok) return [];
  const allProducts = await res.json();
  return allProducts.filter(i => i.quantity <= 5).sort((a,b) => a.quantity - b.quantity).slice(0, limit);
}

export async function fetchScanHistory(limit = 20) {
  const res = await fetch(API + `/api/transactions?limit=${limit}`);
  if (!res.ok) return null;
  return await res.json();
}

export async function fetchHistory(barcode = '', limit = 100) {
  const url = barcode ? `${API}/api/transactions?barcode=${encodeURIComponent(barcode)}&limit=${limit}` : `${API}/api/transactions?limit=${limit}`;
  const res = await fetch(url);
  if (!res.ok) return null;
  return await res.json();
}

export async function submitScan(barcode, mode, qty = 1) {
  const res = await fetch(API + '/api/scan', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ barcode, mode, qty })
  });
  if (!res.ok) return null;
  return await res.json();
}

export async function saveProduct(product) {
  const res = await fetch(API + '/api/products', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(product)
  });
  if (!res.ok) return null;
  return await res.json();
}

export async function deleteProductById(id) {
  const res = await fetch(API + '/api/products/' + id, { method: 'DELETE' });
  return res.ok;
}

export async function uploadCSV(file) {
  const formData = new FormData();
  formData.append('file', file);
  
  const res = await fetch(API + '/api/products/upload-csv', {
    method: 'POST',
    body: formData
  });
  if (!res.ok) throw new Error(await res.text());
  return await res.json();
}

