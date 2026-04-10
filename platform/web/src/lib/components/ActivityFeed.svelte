<script>
  export let activities = [];
  export let segments = [];
  export let onSelectActivity = () => {};
  export let onSelectSegment = () => {};
</script>

<section class="activity-column">
  <div class="panel-card">
    <div class="panel-header panel-header-tight">
      <h3>Recent Activity</h3>
      <button class="icon-button subtle" type="button">
        <span class="material-symbols-outlined">more_horiz</span>
      </button>
    </div>

    <div class="activity-list">
      {#each activities as activity}
        <button class="activity-item" on:click={() => onSelectActivity(activity)} type="button">
          <span class="activity-dot {activity.tone}"></span>
          <div>
            <h4>{activity.title}</h4>
            <p>{activity.description}</p>
            <span>{activity.at}</span>
          </div>
        </button>
      {/each}
    </div>
  </div>

  <div class="panel-card">
    <div class="section-heading">Top Growth Segments</div>
    <div class="segment-list">
      {#each segments as segment}
        <button class="segment-item" on:click={() => onSelectSegment(segment)} type="button">
          <span>{segment.label}</span>
          <div class="segment-meter">
            <div class="segment-meter-fill" style={`width: ${segment.value}`}></div>
          </div>
          <strong>{segment.value}</strong>
        </button>
      {/each}
    </div>
  </div>
</section>