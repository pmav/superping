const { createApp, ref, computed, onMounted, onBeforeUnmount, nextTick, watch } = Vue;

createApp({
  setup() {
    const pings = ref([]);
    const selectedPing = ref(null);
    const liveUpdatesByPing = ref({});
    const filter = ref('all');
    const connectionLabel = ref('Ready');
    const formMessage = ref('');
    const messageColor = ref('var(--coral)');
    const darkMode = ref(localStorage.getItem('superping-theme') === 'dark');
    const form = ref({ name: '', ip: '', status: 'on' });
    const filters = [{ value: 'all', label: 'All' }, { value: 'on', label: 'Monitoring' }, { value: 'off', label: 'Paused' }];
    const charts = new Map();
    let refreshTimer = null;

    const visiblePings = computed(() => pings.value.filter(ping => filter.value === 'all' || ping.status === filter.value));
    const liveUpdatesEnabled = computed(() => selectedPing.value ? liveUpdatesByPing.value[selectedPing.value.uuid] !== false : false);

    function toggleTheme() {
      darkMode.value = !darkMode.value;
      localStorage.setItem('superping-theme', darkMode.value ? 'dark' : 'light');
      document.documentElement.classList.toggle('dark', darkMode.value);
    }

    async function request(path, options = {}) {
      const response = await fetch(path, { headers: { 'Content-Type': 'application/json' }, ...options });
      if (!response.ok) { const body = await response.json().catch(() => ({})); throw new Error(body.error || `Request failed (${response.status})`); }
      return response.status === 204 ? null : response.json();
    }

    async function load() {
      try {
        const selectedUUID = selectedPing.value?.uuid;
        const records = await request('/pings');
        pings.value = await Promise.all(records.map(async ping => {
          const results = await request(`/pings/${ping.uuid}/results`);
          return { ...ping, totalResults: ping.total_results ?? 0, results };
        }));
        if (selectedUUID) {
          const selected = pings.value.find(ping => ping.uuid === selectedUUID);
          if (selected) {
            selected.aggregations = await request(`/pings/${selected.uuid}/aggregations`);
            selectedPing.value = selected;
          } else {
            selectedPing.value = null;
          }
        }
        connectionLabel.value = 'Ready';
        await nextTick();
      } catch (error) { connectionLabel.value = 'Service unavailable'; formMessage.value = error.message; }
    }

    async function addPing() {
      formMessage.value = '';
      try { await request('/pings', { method: 'POST', body: JSON.stringify(form.value) }); form.value = { name: '', ip: '', status: 'on' }; messageColor.value = 'var(--green)'; formMessage.value = 'Target added'; await load(); }
      catch (error) { messageColor.value = 'var(--coral)'; formMessage.value = error.message; }
    }

    async function updateStatus(ping) { try { await request(`/pings/${ping.uuid}`, { method: 'PUT', body: JSON.stringify({ status: ping.status }) }); } catch (error) { formMessage.value = error.message; await load(); } }
    async function deletePing(uuid) { if (!window.confirm('Delete this ping target?')) return; try { await request(`/pings/${uuid}`, { method: 'DELETE' }); await load(); } catch (error) { formMessage.value = error.message; } }
    function countFor(value) { return value === 'all' ? pings.value.length : pings.value.filter(ping => ping.status === value).length; }
    function latestResult(ping) { return ping.results?.at(-1); }
    function totalResults(ping) { return ping.totalResults ?? ping.total_results ?? ping.results?.length ?? 0; }
    function openPing(ping) { selectedPing.value = ping; }
    function closePing() { selectedPing.value = null; }
    function toggleLiveUpdates() {
      if (!selectedPing.value) return;
      const uuid = selectedPing.value.uuid;
      liveUpdatesByPing.value = { ...liveUpdatesByPing.value, [uuid]: !liveUpdatesEnabled.value };
    }
    function stopAutoRefresh() { if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null; } }
    function startAutoRefresh() {
      stopAutoRefresh();
      if (!selectedPing.value) return;
      refreshTimer = window.setInterval(async () => {
        await load();
      }, 1000);
    }
    const dateFormatter = new Intl.DateTimeFormat('en-GB', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' });
    function formatDate(value) {
      if (!value) return '--';
      const parts = Object.fromEntries(dateFormatter.formatToParts(new Date(value)).map(part => [part.type, part.value]));
      return `${parts.day}/${parts.month}/${parts.year} ${parts.hour}:${parts.minute}`;
    }
    function setChart(element, ping) { if (element) { element.dataset.uuid = ping.uuid; drawChart(element, ping); } }
    function drawChart(canvas, ping) {
      const chart = charts.get(ping.uuid); if (chart) chart.destroy();
      const results = (ping.results || []).slice(-60);
      const context = canvas.getContext('2d');
      const dark = darkMode.value;
      const colors = dark ? { line: '#6cdaa0', fill: 'rgba(108,218,160,.16)', grid: '#314238', text: '#9caf9f', tooltip: '#19251e' } : { line: '#168456', fill: 'rgba(22,132,86,.12)', grid: '#e3ebe5', text: '#718078', tooltip: '#ffffff' };
      charts.set(ping.uuid, new Chart(context, { type: 'line', data: { labels: results.map(result => new Date(result.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })), datasets: [{ data: results.map(result => result.duration), borderColor: colors.line, backgroundColor: colors.fill, pointRadius: 0, pointHoverRadius: 0, tension: .3, fill: true }] }, options: { responsive: true, maintainAspectRatio: false, animation: false, plugins: { legend: { display: false }, tooltip: { backgroundColor: colors.tooltip, titleColor: dark ? '#edf5ef' : '#17231f', bodyColor: dark ? '#edf5ef' : '#17231f', callbacks: { label: item => `${item.parsed.y} ms` } } }, scales: { x: { display: true, grid: { display: false }, ticks: { color: colors.text, autoSkip: true, maxTicksLimit: 8, maxRotation: 0 } }, y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.text, callback: value => `${value} ms` } } } } }));
    }
    function setAggregationChart(element, ping) { if (element && ping) drawAggregationChart(element, ping); }
    function drawAggregationChart(canvas, ping) {
      const chartKey = `${ping.uuid}-aggregation`;
      const chart = charts.get(chartKey); if (chart) chart.destroy();
      const aggregations = ping.aggregations || [];
      const context = canvas.getContext('2d');
      const dark = darkMode.value;
      const colors = dark ? { average: '#6cdaa0', minimum: '#79b8e5', maximum: '#f0c66a', grid: '#314238', text: '#9caf9f', tooltip: '#19251e' } : { average: '#168456', minimum: '#397ca8', maximum: '#b87818', grid: '#e3ebe5', text: '#718078', tooltip: '#ffffff' };
      const line = (label, key, color) => ({ label, data: aggregations.map(item => item[key]), borderColor: color, backgroundColor: color, pointRadius: 2, pointHoverRadius: 4, tension: .25, fill: false });
      charts.set(chartKey, new Chart(context, { type: 'line', data: { labels: aggregations.map(item => new Date(item.period_start).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })), datasets: [line('Average', 'average_duration', colors.average), line('Min', 'minimum_duration', colors.minimum), line('Max', 'maximum_duration', colors.maximum)] }, options: { responsive: true, maintainAspectRatio: false, animation: false, interaction: { mode: 'index', intersect: false }, plugins: { legend: { display: true, position: 'bottom', labels: { color: colors.text, usePointStyle: true, boxWidth: 7, padding: 16 } }, tooltip: { backgroundColor: colors.tooltip, titleColor: dark ? '#edf5ef' : '#17231f', bodyColor: dark ? '#edf5ef' : '#17231f', callbacks: { label: item => `${item.dataset.label}: ${item.parsed.y} ms` } } }, scales: { x: { grid: { display: false }, ticks: { color: colors.text, autoSkip: true, maxTicksLimit: 8, maxRotation: 0 } }, y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.text, callback: value => `${value} ms` } } } } }));
    }
    watch([selectedPing, liveUpdatesEnabled], ([ping, enabled]) => {
      if (ping && enabled) startAutoRefresh();
      else stopAutoRefresh();
    }, { immediate: true });

    onMounted(() => { document.documentElement.classList.toggle('dark', darkMode.value); load(); });
    onBeforeUnmount(() => { stopAutoRefresh(); charts.forEach(chart => chart.destroy()); });
    return { pings, selectedPing, darkMode, liveUpdatesEnabled, filter, filters, form, formMessage, messageColor, connectionLabel, visiblePings, countFor, latestResult, totalResults, formatDate, openPing, closePing, toggleLiveUpdates, toggleTheme, addPing, updateStatus, deletePing, load, setChart, setAggregationChart };
  }
}).mount('#app');
