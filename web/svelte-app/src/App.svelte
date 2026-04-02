<script>
  import { onMount } from 'svelte';
  import { currentPage, liveLog, stats, lowStock } from '$lib/stores.js';
  import { fetchStats, fetchLowStock } from '$lib/api.js';
  
  // Components
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Stats from '$lib/components/Stats.svelte';
  import LiveFeed from '$lib/components/LiveFeed.svelte';
  import LowStockTable from '$lib/components/LowStockTable.svelte';
  import Scanner from '$lib/components/Scanner.svelte';
  import Items from '$lib/components/Items.svelte';
  import History from '$lib/components/History.svelte';
  import ItemModal from '$lib/components/ItemModal.svelte';
  import Toast from '$lib/components/Toast.svelte';

  let itemModalRef;

  onMount(async () => {
    // Load initial dashboard
    await loadDashboard();

    // SSE for live scan events
    const source = new EventSource('/api/events');
    source.onmessage = event => {
      const ev = JSON.parse(event.data);
      const time = new Date().toLocaleTimeString();
      liveLog.update(log => [{ ...ev, time }, ...log].slice(0, 30));
      refreshStats();
      if ($currentPage === 'scanner') window.dispatchEvent(new CustomEvent('refreshScanner'));
      if ($currentPage === 'history') window.dispatchEvent(new CustomEvent('refreshHistory'));
      if ($currentPage === 'items') window.dispatchEvent(new CustomEvent('refreshItems'));
    };

    // Listen for item modal events
    window.addEventListener('openItemModal', (e) => {
      itemModalRef.openModal(e.detail);
    });

    window.addEventListener('itemSaved', async () => {
      await loadDashboard();
    });

    window.addEventListener('itemDeleted', async () => {
      await loadDashboard();
    });

    window.addEventListener('scanSuccess', async () => {
      await loadDashboard();
    });
  });

  async function loadDashboard() {
    await refreshStats();
    const low = await fetchLowStock(10);
    if (low) lowStock.set(low);
  }

  async function refreshStats() {
    const s = await fetchStats();
    if (s) stats.set(s);
  }
</script>

<Sidebar />

<main>
  <!-- Dashboard -->
  <section class="page" class:active={$currentPage==='dashboard'}>
    <h1>Dashboard</h1>
    <Stats />
    <LiveFeed />
    <LowStockTable />
  </section>

  <!-- Items -->
  <section class="page" class:active={$currentPage==='items'}>
    <Items />
  </section>

  <!-- Scanner -->
  <section class="page" class:active={$currentPage==='scanner'}>
    <h1>Scanner</h1>
    <Scanner />
  </section>

  <!-- History -->
  <section class="page" class:active={$currentPage==='history'}>
    <History />
  </section>
</main>

<ItemModal bind:this={itemModalRef} />
<Toast />

<style global>
  :root {
    --bg: #f9f9f8;
    --surface: #fff;
    --border: #e4e2da;
    --text: #1a1a18;
    --muted: #6b6b66;
    --accent: #1a6ef7;
    --green: #1d9e75;
    --red: #d85a30;
    --amber: #ba7517;
    --radius: 8px;
    --sidebar: 220px;
  }

  :global(*, *::before, *::after) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background: var(--bg);
    color: var(--text);
    font-size: 14px;
    display: flex;
    min-height: 100vh;
  }

  :global(main) {
    margin-left: var(--sidebar);
    flex: 1;
    padding: 2rem;
    max-width: 1100px;
  }

  :global(.page) {
    display: none;
  }

  :global(.page.active) {
    display: block;
  }

  :global(h1) {
    font-size: 20px;
    font-weight: 600;
    margin-bottom: 1.5rem;
  }

  :global(h2) {
    font-size: 15px;
    font-weight: 600;
    margin-bottom: 1rem;
  }

  :global(code) {
    font-family: 'Courier New', monospace;
    font-size: 11px;
  }
</style>
