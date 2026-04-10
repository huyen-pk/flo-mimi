<script>
  export let session;
  export let placeholder = 'Search secure archives...';
  export let value = '';
  export let onSearchInput = () => {};
  export let onSearchSubmit = () => {};
  export let onUtilityAction = () => {};

  function submitSearch() {
    onSearchSubmit(value);
  }
</script>

<header class="topbar">
  <form class="search-shell" on:submit|preventDefault={submitSearch}>
    <span class="material-symbols-outlined">search</span>
    <input
      aria-label="Search interface"
      class="search-input"
      on:input={(event) => onSearchInput(event.currentTarget.value)}
      placeholder={placeholder}
      type="text"
      value={value}
    />
  </form>

  <div class="topbar-actions">
    <button class="icon-button" on:click={() => onUtilityAction('notifications')} type="button">
      <span class="material-symbols-outlined">notifications</span>
    </button>
    <button class="icon-button" on:click={() => onUtilityAction('help')} type="button">
      <span class="material-symbols-outlined">help</span>
    </button>
    <div class="topbar-divider"></div>
    <div class="session-card">
      <div>
        <div class="session-label">{session?.label}</div>
        <div class="session-node">{session?.node}</div>
      </div>
      <span class="session-dot {session?.status === 'online' ? 'online' : 'offline'}"></span>
    </div>
  </div>
</header>