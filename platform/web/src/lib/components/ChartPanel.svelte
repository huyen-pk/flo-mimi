<script>
  export let performance;
  export let onModeSelect = () => {};
  export let onBarSelect = () => {};
  let activeMode = performance?.activeMode || 'Daily';

  $: if (performance?.activeMode && performance.activeMode !== activeMode) {
    activeMode = performance.activeMode;
  }

  function selectMode(mode) {
    activeMode = mode;
    onModeSelect(mode);
  }
</script>

<section class="panel-card tall-panel">
  <div class="panel-header">
    <h3>{performance.title}</h3>
    <div class="segmented-control">
      {#each performance.modes as mode}
        <button
          class:active={mode === activeMode}
          on:click={() => selectMode(mode)}
          type="button"
        >
          {mode}
        </button>
      {/each}
    </div>
  </div>

  <div class="chart-bars">
    {#each performance.bars as bar}
      <button
        class:emphasis={bar.emphasis}
        class="chart-bar"
        on:click={() => onBarSelect(bar)}
        style={`height: ${bar.height}%`}
        type="button"
      >
        <span class="chart-value">{bar.value}</span>
      </button>
    {/each}
  </div>
  <div class="chart-labels">
    {#each performance.bars as bar}
      <span>{bar.label}</span>
    {/each}
  </div>
</section>