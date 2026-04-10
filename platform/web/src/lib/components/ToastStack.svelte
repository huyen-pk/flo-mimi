<script>
  export let items = [];
</script>

<div class="toast-stack" aria-live="polite" aria-atomic="true">
  {#each items as item (item.id)}
    <div class="toast {item.tone}">
      <span class="material-symbols-outlined">{item.tone === 'error' ? 'error' : 'check_circle'}</span>
      <div>
        <strong>{item.title}</strong>
        <p>{item.message}</p>
      </div>
    </div>
  {/each}
</div>