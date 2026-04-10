<script>
  export let subscribers;
  export let searchQuery = '';
  export let onTrack = () => {};

  let activeFilter = subscribers.filters[0] || 'All Contacts';

  $: filteredItems = subscribers.items.filter((item) => {
    const query = searchQuery.trim().toLowerCase();
    const passesQuery = !query || `${item.name} ${item.email}`.toLowerCase().includes(query);
    const tags = item.filterTags || [];
    const passesFilter = activeFilter === 'All Contacts' || tags.some((tag) => tag.toLowerCase() === activeFilter.toLowerCase());
    return passesQuery && passesFilter;
  });

  function selectFilter(filter) {
    activeFilter = filter;
    onTrack({
      surface: 'subscriber-filters',
      action: 'select-filter',
      subjectType: 'filter',
      subjectId: filter,
      eventKind: 'analytics'
    });
  }
</script>

<section class="page-header">
  <div>
    <h1>{subscribers.headline}</h1>
    <p>{subscribers.description}</p>
  </div>

  <button
    class="cta-primary"
    on:click={() =>
      onTrack({
        surface: 'subscribers-header',
        action: 'add-subscriber',
        subjectType: 'subscriber',
        subjectId: 'new-subscriber',
        eventKind: 'analytics',
        notify: true
      })}
    type="button"
  >
    <span class="material-symbols-outlined">person_add</span>
    Add Subscriber
  </button>
</section>

<section class="subscriber-filter-card">
  <div class="filter-row">
    {#each subscribers.filters as filter}
      <button class:active={filter === activeFilter} class="filter-chip" on:click={() => selectFilter(filter)} type="button">
        {filter}
      </button>
    {/each}
  </div>

  <div class="network-summary">
    <span>Total Network</span>
    <strong>{subscribers.networkSize}</strong>
  </div>
</section>

<section class="table-card">
  <div class="table-head subscriber-table">
    <span>Subscriber Details</span>
    <span>Security Status</span>
    <span>Assigned Segments</span>
    <span>Last Interaction</span>
    <span class="align-right">Actions</span>
  </div>

  <div class="table-body">
    {#each filteredItems as item}
      <div class="table-row subscriber-table">
        <div>
          <h3>{item.name}</h3>
          <p>{item.email}</p>
        </div>
        <div>
          <span class="status-pill {item.tone}">{item.securityStatus}</span>
        </div>
        <div class="segment-badges">
          {#each item.assignedSegments as segment}
            <span>{segment}</span>
          {/each}
        </div>
        <div>
          <strong>{item.lastInteraction}</strong>
        </div>
        <div class="align-right">
          <button
            class="cta-inline"
            on:click={() =>
              onTrack({
                surface: 'subscriber-row',
                action: 'inspect-subscriber',
                subjectType: 'subscriber',
                subjectId: item.id,
                eventKind: 'analytics',
                metadata: { securityStatus: item.securityStatus }
              })}
            type="button"
          >
            View Profile
          </button>
        </div>
      </div>
    {/each}
  </div>
</section>