<script>
  export let campaigns;
  export let searchQuery = '';
  export let onTrack = () => {};

  $: filteredItems = campaigns.items.filter((item) => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) {
      return true;
    }
    return `${item.title} ${item.summary} ${item.audienceLabel}`.toLowerCase().includes(query);
  });

  function createBrief() {
    onTrack({
      surface: 'campaigns-header',
      action: 'create-brief',
      subjectType: 'campaign',
      subjectId: 'new-brief',
      eventKind: 'both',
      notify: true,
      metadata: { source: 'campaigns-view' }
    });
  }

  function trackCampaignAction(item) {
    onTrack({
      surface: 'campaign-row',
      action: item.actionLabel.toLowerCase().replace(/\s+/g, '-'),
      subjectType: 'campaign',
      subjectId: item.id,
      campaignId: item.id,
      eventKind: 'both',
      notify: true,
      metadata: {
        status: item.status,
        title: item.title
      }
    });
  }
</script>

<section class="page-header">
  <div>
    <h1>{campaigns.headline}</h1>
    <p>{campaigns.description}</p>
  </div>

  <button class="cta-primary" on:click={createBrief} type="button">
    <span class="material-symbols-outlined">add</span>
    Create New Brief
  </button>
</section>

<section class="summary-grid">
  {#each campaigns.stats as stat}
    <button
      class="summary-card {stat.tone}"
      on:click={() =>
        onTrack({
          surface: 'campaign-summary',
          action: 'inspect-summary',
          subjectType: 'summary',
          subjectId: stat.id,
          eventKind: 'analytics'
        })}
      type="button"
    >
      <span class="eyebrow">{stat.label}</span>
      <h3>{stat.value}</h3>
      <p>{stat.detail}</p>
      {#if stat.progress > 0}
        <span class="progress-rail"><span class="progress-fill" style={`width: ${stat.progress}%`}></span></span>
      {/if}
    </button>
  {/each}
</section>

<section class="table-card">
  <div class="table-head campaign-table">
    <span>Campaign Identity</span>
    <span>Status</span>
    <span>Audience</span>
    <span>Performance</span>
    <span class="align-right">Actions</span>
  </div>

  <div class="table-body">
    {#each filteredItems as item}
      <div class="table-row campaign-table">
        <div>
          <h3>{item.title}</h3>
          <p>{item.summary}</p>
        </div>
        <div>
          <span class="status-pill {item.tone}">{item.status}</span>
        </div>
        <div>
          <strong>{item.audience}</strong>
          <p>{item.audienceLabel}</p>
        </div>
        <div class="performance-meta">
          <div>
            <strong>{item.openRate}</strong>
            <span>Open Rate</span>
          </div>
          <div>
            <strong>{item.clickRate}</strong>
            <span>Click Rate</span>
          </div>
        </div>
        <div class="align-right">
          <button class="cta-inline" on:click={() => trackCampaignAction(item)} type="button">
            {item.actionLabel}
          </button>
        </div>
      </div>
    {/each}
  </div>
</section>