<script>
  import { toastVisible, toastMsg, toastError } from '$lib/stores.js';
</script>

<div class="toast" class:show={$toastVisible} style="background: {$toastError ? 'var(--red)' : 'var(--text)'};">
  {$toastMsg}
</div>

<style>
  .toast { position: fixed; bottom: 1.5rem; right: 1.5rem; background: var(--text); color: #fff; padding: 10px 18px; border-radius: var(--radius); font-size: 13px; opacity: 0; transition: opacity .25s; z-index: 999; pointer-events: none; }
  .toast.show { opacity: 1; }
</style>
