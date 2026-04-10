<script>
  import MetricCard from '../components/MetricCard.svelte';
  import ChartPanel from '../components/ChartPanel.svelte';
  import ActivityFeed from '../components/ActivityFeed.svelte';

  export let brand;
  export let dashboard;
  export let onTrack = () => {};

  function trackMetric(metric) {
    onTrack({
      surface: 'dashboard-metric',
      action: 'inspect',
      subjectType: 'metric',
      subjectId: metric.id,
      eventKind: 'analytics'
    });
  }

  function trackMode(mode) {
    onTrack({
      surface: 'campaign-performance',
      action: `switch-${mode.toLowerCase()}`,
      subjectType: 'chart',
      subjectId: 'campaign-performance',
      eventKind: 'analytics'
    });
  }

  function trackBar(bar) {
    onTrack({
      surface: 'campaign-performance',
      action: 'inspect-day',
      subjectType: 'chart-bar',
      subjectId: bar.label,
      eventKind: 'analytics',
      metadata: { value: bar.value }
    });
  }
</script>

<section class="hero-block">
  <div>
    <h1>
      {brand.heroTitle}
      <span>{brand.heroAccent}</span>
    </h1>
    <div class="hero-meta">
      <span class="shield-chip">
        <span class="material-symbols-outlined filled">verified_user</span>
        {brand.heroNote}
      </span>
      <p>{brand.lastSync}</p>
    </div>
  </div>
</section>

<section class="metric-grid">
  {#each dashboard.metrics as metric}
    <MetricCard metric={metric} onSelect={trackMetric} />
  {/each}
</section>

<section class="dashboard-grid">
  <div class="dashboard-main-column">
    <ChartPanel performance={dashboard.performance} onModeSelect={trackMode} onBarSelect={trackBar} />

    <div class="support-grid">
      <button
        class="panel-card security-card"
        on:click={() =>
          onTrack({
            surface: 'security-status',
            action: 'inspect-dmarc',
            subjectType: 'status-card',
            subjectId: 'dmarc-status',
            eventKind: 'analytics'
          })}
        type="button"
      >
        <div class="status-badge-icon">
          <span class="material-symbols-outlined">security</span>
        </div>
        <div class="section-heading">{dashboard.securityCard.title}</div>
        <h3 class="support-title">{dashboard.securityCard.status}</h3>
        <p>{dashboard.securityCard.description}</p>
      </button>

      <button
        class="panel-card billing-card"
        on:click={() =>
          onTrack({
            surface: 'billing',
            action: 'manage-subscription',
            subjectType: 'billing',
            subjectId: 'pro-plan',
            eventKind: 'analytics',
            notify: true
          })}
        type="button"
      >
        <div class="section-heading invert">{dashboard.billingCard.eyebrow}</div>
        <h3 class="support-title invert">{dashboard.billingCard.title}</h3>
        <p class="invert-copy">{dashboard.billingCard.date}</p>
        <span class="cta-secondary ghost">{dashboard.billingCard.action}</span>
      </button>
    </div>
  </div>

  <ActivityFeed
    activities={dashboard.activities}
    onSelectActivity={(activity) =>
      onTrack({
        surface: 'activity-feed',
        action: 'inspect-activity',
        subjectType: 'activity',
        subjectId: activity.id,
        eventKind: 'analytics'
      })}
    onSelectSegment={(segment) =>
      onTrack({
        surface: 'segment-feed',
        action: 'inspect-segment',
        subjectType: 'segment',
        subjectId: segment.id,
        eventKind: 'analytics'
      })}
    segments={dashboard.segments}
  />
</section>