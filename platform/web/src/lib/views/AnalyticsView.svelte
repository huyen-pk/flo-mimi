<script>
  export let analytics;
  export let onTrack = () => {};

  function fireSignal(signal) {
    onTrack({
      surface: 'telemetry-signal',
      action: signal.action,
      subjectType: signal.subjectType,
      subjectId: signal.subjectId,
      eventKind: signal.eventKind,
      notify: true,
      metadata: { origin: 'analytics-view' }
    });
  }
</script>

<section class="page-header narrow">
  <div>
    <h1>{analytics.headline}</h1>
    <p>{analytics.description}</p>
  </div>
</section>

<section class="pipeline-grid">
  {#each analytics.pipelines as stage}
    <button
      class="panel-card pipeline-card"
      on:click={() =>
        onTrack({
          surface: 'pipeline-stage',
          action: 'inspect-stage',
          subjectType: 'pipeline',
          subjectId: stage.id,
          eventKind: 'analytics'
        })}
      type="button"
    >
      <div class="section-heading">{stage.status}</div>
      <h3>{stage.title}</h3>
      <p>{stage.description}</p>
    </button>
  {/each}
</section>

<section class="signal-grid">
  {#each analytics.signals as signal}
    <div class="panel-card signal-card">
      <div>
        <div class="section-heading">{signal.eventKind}</div>
        <h3>{signal.title}</h3>
        <p>{signal.description}</p>
      </div>
      <button class="cta-primary compact" on:click={() => fireSignal(signal)} type="button">
        {signal.action}
      </button>
    </div>
  {/each}
</section>