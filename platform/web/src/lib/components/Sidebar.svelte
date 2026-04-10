<script>
  export let brand;
  export let routes = [];
  export let currentRoute = 'dashboard';
  export let onNavigate = () => {};
  export let onUtilityAction = () => {};
</script>

<aside class="sidebar">
  <div>
    <div class="brand-lockup">
      <div class="brand-mark">
        <span class="material-symbols-outlined filled">shield</span>
      </div>
      <div>
        <div class="brand-name">{brand?.name}</div>
        <div class="brand-tagline">{brand?.tagline}</div>
      </div>
    </div>

    <nav class="nav-group">
      {#each routes as route}
        <button
          class:active={route.id === currentRoute}
          class="nav-item"
          on:click={() => onNavigate(route.id)}
          type="button"
        >
          <span class="material-symbols-outlined {route.id === currentRoute ? 'filled' : ''}">{route.icon}</span>
          <span>{route.label}</span>
        </button>
      {/each}
    </nav>
  </div>

  <div class="sidebar-footer">
    <button class="subtle-nav-item" on:click={() => onUtilityAction('settings')} type="button">
      <span class="material-symbols-outlined">settings</span>
      <span>Settings</span>
    </button>
    <button class="subtle-nav-item" on:click={() => onUtilityAction('security')} type="button">
      <span class="material-symbols-outlined">shield</span>
      <span>Security</span>
    </button>

    <div class="operator-card">
      <div class="operator-avatar">AD</div>
      <div>
        <div class="operator-name">Admin</div>
        <div class="operator-role">Fortified Serenity</div>
      </div>
    </div>
  </div>
</aside>