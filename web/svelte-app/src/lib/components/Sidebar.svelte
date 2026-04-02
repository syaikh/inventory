<script>
  import { currentPage } from '$lib/stores.js';
</script>

<aside>
  <div class="logo">
    <svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg" width="22" height="22">
      <rect x="2" y="4" width="16" height="12" rx="2" stroke="#1a6ef7" stroke-width="1.5"/>
      <path d="M6 8h8M6 12h5" stroke="#1a6ef7" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
    Inventory
  </div>
  <nav>
    <button 
      on:click={() => currentPage.set('dashboard')}
      class:active={$currentPage==='dashboard'}
    >
      <span>Dashboard</span>
    </button>
    <button 
      on:click={() => currentPage.set('items')}
      class:active={$currentPage==='items'}
    >
      <span>Items</span>
    </button>
    <button 
      on:click={() => currentPage.set('scanner')}
      class:active={$currentPage==='scanner'}
    >
      <span>Scanner</span>
    </button>
    <button 
      on:click={() => currentPage.set('history')}
      class:active={$currentPage==='history'}
    >
      <span>History</span>
    </button>
  </nav>
</aside>

<style>
  aside { width: var(--sidebar); background: var(--surface); border-right: 1px solid var(--border); display: flex; flex-direction: column; padding: 1.5rem 0; position: fixed; top: 0; bottom: 0; left: 0; }
  .logo { padding: 0 1.25rem 1.5rem; font-size: 16px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
  .logo svg { flex-shrink: 0; }
  nav { display: flex; flex-direction: column; }
  button { display: flex; align-items: center; gap: 10px; padding: 9px 1.25rem; color: var(--muted); text-decoration: none; border-left: 3px solid transparent; transition: all .15s; cursor: pointer; border: none; background: none; font-size: 14px; text-align: left; }
  button:hover { color: var(--text); background: var(--bg); }
  button.active { color: var(--accent); border-left-color: var(--accent); background: #eef4ff; }
</style>
