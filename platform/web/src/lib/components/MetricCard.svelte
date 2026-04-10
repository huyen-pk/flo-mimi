<script>
  export let metric;
  export let onSelect = () => {};
</script>

<button class="metric-card" on:click={() => onSelect(metric)} type="button">
  <div class="metric-copy">
    <p class="eyebrow">{metric.label}</p>
    <h3>{metric.value}</h3>
    <p class="metric-delta {metric.accent}">{metric.delta}</p>
  </div>
  <div class="metric-icon {metric.accent}">
    <span class="material-symbols-outlined">{metric.icon}</span>
  </div>
</button>