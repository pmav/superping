const { createApp, ref, computed, onMounted, onBeforeUnmount, nextTick, watch } = Vue;

createApp({
  setup() {
    const pings = ref([]);
    const selectedPing = ref(null);
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
        const records = await request('/pings');
        pings.value = await Promise.all(records.map(async ping => {
          const results = await request(`/pings/${ping.uuid}/results`);
          return { ...ping, totalResults: ping.total_results ?? 0, results };
        }));
        if (selectedPing.value) selectedPing.value = pings.value.find(ping => ping.uuid === selectedPing.value.uuid) || null;
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
    function stopAutoRefresh() { if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null; } }
    function startAutoRefresh() {
      stopAutoRefresh();
      if (!selectedPing.value) return;
      refreshTimer = window.setInterval(async () => {
        await load();
      }, 1000);
    }
    function formatDate(value) { return value ? new Date(value).toLocaleString() : '--'; }
    function setChart(element, ping) { if (element) { element.dataset.uuid = ping.uuid; drawChart(element, ping); } }
    function drawChart(canvas, ping) {
      const chart = charts.get(ping.uuid); if (chart) chart.destroy();
      const results = (ping.results || []).slice(-60);
      const context = canvas.getContext('2d');
      const dark = darkMode.value;
      const colors = dark ? { line: '#6cdaa0', fill: 'rgba(108,218,160,.16)', grid: '#314238', text: '#9caf9f', tooltip: '#19251e' } : { line: '#168456', fill: 'rgba(22,132,86,.12)', grid: '#e3ebe5', text: '#718078', tooltip: '#ffffff' };
      charts.set(ping.uuid, new Chart(context, { type: 'line', data: { labels: results.map(result => new Date(result.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })), datasets: [{ data: results.map(result => result.duration), borderColor: colors.line, backgroundColor: colors.fill, pointRadius: 0, pointHoverRadius: 0, tension: .3, fill: true }] }, options: { responsive: true, maintainAspectRatio: false, animation: false, plugins: { legend: { display: false }, tooltip: { backgroundColor: colors.tooltip, titleColor: dark ? '#edf5ef' : '#17231f', bodyColor: dark ? '#edf5ef' : '#17231f', callbacks: { label: item => `${item.parsed.y} ms` } } }, scales: { x: { display: true, grid: { display: false }, ticks: { color: colors.text, autoSkip: true, maxTicksLimit: 8, maxRotation: 0 } }, y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.text, callback: value => `${value} ms` } } } } }));
    }
    watch(selectedPing, (ping) => {
      if (ping) startAutoRefresh();
      else stopAutoRefresh();
    }, { immediate: true });

    onMounted(() => { document.documentElement.classList.toggle('dark', darkMode.value); load(); });
    onBeforeUnmount(() => { stopAutoRefresh(); charts.forEach(chart => chart.destroy()); });
    return { pings, selectedPing, darkMode, filter, filters, form, formMessage, messageColor, connectionLabel, visiblePings, countFor, latestResult, totalResults, formatDate, openPing, closePing, toggleTheme, addPing, updateStatus, deletePing, load, setChart };
  }
}).mount('#app');
