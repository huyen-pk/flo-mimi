<script>
  import { onMount } from 'svelte';
  import Sidebar from './lib/components/Sidebar.svelte';
  import Topbar from './lib/components/Topbar.svelte';
  import ToastStack from './lib/components/ToastStack.svelte';
  import DashboardView from './lib/views/DashboardView.svelte';
  import CampaignsView from './lib/views/CampaignsView.svelte';
  import SubscribersView from './lib/views/SubscribersView.svelte';
  import AnalyticsView from './lib/views/AnalyticsView.svelte';
  import { fetchBootstrap, postInteraction } from './lib/api';

  const routes = [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'campaigns', label: 'Campaigns', icon: 'send' },
    { id: 'subscribers', label: 'Subscribers', icon: 'group' },
    { id: 'analytics', label: 'Analytics', icon: 'analytics' }
  ];

  let bootstrap;
  let loading = true;
  let loadError = '';
  let currentRoute = 'dashboard';
  let searchQuery = '';
  let sessionId = '';
  let toasts = [];

  onMount(() => {
    sessionId = ensureSessionId();
    syncRoute();
    window.addEventListener('hashchange', syncRoute);
    loadBootstrap();

    return () => {
      window.removeEventListener('hashchange', syncRoute);
    };
  });

  async function loadBootstrap() {
    loading = true;
    loadError = '';

    try {
      bootstrap = await fetchBootstrap();
    } catch (error) {
      loadError = error.message;
    } finally {
      loading = false;
    }
  }

  function syncRoute() {
    const nextRoute = normalizeRoute(window.location.hash);
    currentRoute = routes.some((route) => route.id === nextRoute) ? nextRoute : 'dashboard';
  }

  function navigate(route) {
    currentRoute = route;
    window.location.hash = route;
    track({
      route,
      surface: 'navigation',
      action: 'select',
      subjectType: 'navigation',
      subjectId: route,
      eventKind: 'analytics'
    });
  }

  async function track(payload) {
    const notify = payload.notify === true;
    const route = payload.route || currentRoute;
    const metadata = {
      ...(payload.metadata || {}),
      sourceRoute: route
    };

    try {
      const response = await postInteraction({
        sessionId,
        userId: 'curator-admin',
        route,
        surface: payload.surface,
        action: payload.action,
        subjectType: payload.subjectType,
        subjectId: payload.subjectId,
        campaignId: payload.campaignId,
        recipientId: payload.recipientId,
        eventKind: payload.eventKind,
        metadata
      });

      if (notify) {
        pushToast({
          tone: 'success',
          title: 'Event routed',
          message: `Published to ${response.published.join(' + ') || 'the platform'}.`
        });
      }

      if (response.stored) {
        await loadBootstrap();
      }
    } catch (error) {
      pushToast({
        tone: 'error',
        title: 'Routing failed',
        message: error.message
      });
    }
  }

  function handleSearchSubmit(query) {
    if (!query.trim()) {
      return;
    }
    track({
      surface: 'search',
      action: 'submit',
      subjectType: 'query',
      subjectId: query.trim(),
      eventKind: 'analytics'
    });
  }

  function handleUtilityAction(action) {
    track({
      surface: 'utility',
      action,
      subjectType: 'utility',
      subjectId: action,
      eventKind: 'analytics',
      notify: action === 'security'
    });
  }

  function pushToast(toast) {
    const entry = { id: crypto.randomUUID(), ...toast };
    toasts = [entry, ...toasts].slice(0, 4);
    window.setTimeout(() => {
      toasts = toasts.filter((item) => item.id !== entry.id);
    }, 3200);
  }

  function ensureSessionId() {
    const storageKey = 'curator-platform-session';
    const existing = window.localStorage.getItem(storageKey);
    if (existing) {
      return existing;
    }

    const generated = `session-${crypto.randomUUID()}`;
    window.localStorage.setItem(storageKey, generated);
    return generated;
  }

  function normalizeRoute(hashValue) {
    const normalized = hashValue.replace(/^#\/?/, '').trim();
    return normalized || 'dashboard';
  }
</script>

<div class="app-shell">
  {#if bootstrap}
    <Sidebar
      brand={bootstrap.brand}
      currentRoute={currentRoute}
      onNavigate={navigate}
      onUtilityAction={handleUtilityAction}
      routes={routes}
    />
  {/if}

  <div class="app-main">
    {#if bootstrap}
      <Topbar
        onSearchInput={(value) => (searchQuery = value)}
        onSearchSubmit={handleSearchSubmit}
        onUtilityAction={handleUtilityAction}
        placeholder={bootstrap.brand.searchPlaceholder}
        session={bootstrap.session}
        value={searchQuery}
      />
    {/if}

    <main class="page-shell">
      {#if loading}
        <section class="loading-card">
          <div class="section-heading">Loading</div>
          <h2>Bootstrapping the curator console</h2>
          <p>Fetching the embedded view model and preparing interaction routing.</p>
        </section>
      {:else if loadError}
        <section class="loading-card error-card">
          <div class="section-heading">Error</div>
          <h2>The platform UI could not load</h2>
          <p>{loadError}</p>
          <button class="cta-primary compact" on:click={loadBootstrap} type="button">Retry</button>
        </section>
      {:else if currentRoute === 'dashboard'}
        <DashboardView brand={bootstrap.brand} dashboard={bootstrap.dashboard} onTrack={track} />
      {:else if currentRoute === 'campaigns'}
        <CampaignsView campaigns={bootstrap.campaigns} onTrack={track} searchQuery={searchQuery} />
      {:else if currentRoute === 'subscribers'}
        <SubscribersView onTrack={track} searchQuery={searchQuery} subscribers={bootstrap.subscribers} />
      {:else}
        <AnalyticsView analytics={bootstrap.analytics} onTrack={track} />
      {/if}
    </main>
  </div>

  <ToastStack items={toasts} />
</div>