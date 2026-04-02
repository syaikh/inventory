import { writable } from 'svelte/store';

export const currentPage = writable('dashboard');
export const scanMode = writable('in');
export const stats = writable({ total_items: 0, total_units: 0, out_of_stock: 0, scans_today: 0 });
export const items = writable([]);
export const categories = writable([]);
export const lowStock = writable([]);
export const scanHistory = writable([]);
export const history = writable([]);
export const liveLog = writable([]);
export const itemModalOpen = writable(false);
export const editingItem = writable(null);
export const toastVisible = writable(false);
export const toastMsg = writable('');
export const toastError = writable(false);
