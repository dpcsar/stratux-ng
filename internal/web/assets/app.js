(() => {
  const views = [
    { key: 'ahrs', el: document.getElementById('view-ahrs') },
    { key: 'radar', el: document.getElementById('view-radar') },
    { key: 'map', el: document.getElementById('view-map') },
    { key: 'status', el: document.getElementById('view-status') },
    { key: 'settings', el: document.getElementById('view-settings') },
  ];

  const subtitle = document.getElementById('subtitle');
  const drawer = document.getElementById('drawer');
  const drawerBackdrop = document.getElementById('drawer-backdrop');
  const btnMore = document.getElementById('btn-more');
  const btnClose = document.getElementById('btn-close');
  const drawerStatus = document.getElementById('drawer-status');

  const ahrsSummary = document.getElementById('ahrs-summary');
  const radarSummary = document.getElementById('radar-summary');
  const mapSummary = document.getElementById('map-summary');
  const statusJSON = document.getElementById('status-json');

  const ahrsShowRaw = document.getElementById('ahrs-show-raw');
  const radarRange = document.getElementById('radar-range');
  const mapFollow = document.getElementById('map-follow');

  const settingsForm = document.getElementById('settings-form');
  const saveMsg = document.getElementById('save-msg');
  const setGDL90Dest = document.getElementById('set-gdl90-dest');
  const setOwnshipEnable = document.getElementById('set-ownship-enable');
  const setTrafficEnable = document.getElementById('set-traffic-enable');
  const setScenarioEnable = document.getElementById('set-scenario-enable');
  const setScenarioPath = document.getElementById('set-scenario-path');
  const setScenarioStart = document.getElementById('set-scenario-start');
  const setScenarioLoop = document.getElementById('set-scenario-loop');
  const setWebListen = document.getElementById('set-web-listen');
  const setWebEnable = document.getElementById('set-web-enable');

  function setView(key) {
    for (const v of views) {
      v.el.classList.toggle('active', v.key === key);
    }

    // Bottom-nav selection only applies to the primary three tabs.
    for (const btn of document.querySelectorAll('.navbtn')) {
      const isSelected = btn.dataset.view === key;
      btn.setAttribute('aria-selected', isSelected ? 'true' : 'false');
    }
    try {
      history.replaceState(null, '', `#${key}`);
    } catch {
      // ignore
    }
  }

  function openDrawer() {
    drawer.hidden = false;
    drawerBackdrop.hidden = false;

    document.body.classList.add('drawer-open');
  }

  function closeDrawer() {
    drawer.hidden = true;
    drawerBackdrop.hidden = true;

    document.body.classList.remove('drawer-open');
  }

  btnMore.addEventListener('click', openDrawer);
  btnClose.addEventListener('click', closeDrawer);
  drawerBackdrop.addEventListener('click', closeDrawer);
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeDrawer();
  });

  window.addEventListener('hashchange', () => {
    // Ensure the drawer doesn't remain open over content after navigation.
    closeDrawer();
  });

  for (const btn of document.querySelectorAll('.navbtn')) {
    btn.addEventListener('click', () => setView(btn.dataset.view));
  }

  for (const btn of document.querySelectorAll('.drawerbtn')) {
    btn.addEventListener('click', () => {
      setView(btn.dataset.view);
      closeDrawer();
    });
  }

  function pretty(obj) {
    return JSON.stringify(obj, null, 2);
  }

  function setStatusText(s) {
    const mode = s?.mode || '';
    const dest = s?.gdl90_dest || '';
    const interval = s?.interval || '';
    const frames = s?.frames_sent_total ?? 0;

    subtitle.textContent = `${mode} → ${dest} (${interval}) · frames ${frames}`;
    drawerStatus.textContent = `Uptime: ${s?.uptime_sec ?? 0}s\nLast tick: ${s?.last_tick_utc || '—'}\nScenario: ${s?.sim?.scenario ? 'on' : 'off'}`;

    statusJSON.textContent = pretty(s);

    const rr = radarRange?.value || '10';
    const follow = !!mapFollow?.checked;
    const showRaw = !!ahrsShowRaw?.checked;

    ahrsSummary.textContent = pretty({
      ahrs: 'planned',
      mode,
      interval,
      last_tick_utc: s?.last_tick_utc || null,
      show_raw_status_json: showRaw,
    });

    radarSummary.textContent = pretty({
      radar: 'planned',
      gdl90_dest: dest,
      frames_sent_total: frames,
      range_nm: Number(rr),
    });

    mapSummary.textContent = pretty({
      map: 'planned',
      now_utc: s?.now_utc || null,
      sim: s?.sim || {},
      follow_ownship: follow,
    });
  }

  function loadControlState() {
    try {
      const rr = localStorage.getItem('radar_range_nm');
      if (rr && radarRange) radarRange.value = rr;
      const mf = localStorage.getItem('map_follow');
      if (mf !== null && mapFollow) mapFollow.checked = mf === 'true';
      const raw = localStorage.getItem('ahrs_show_raw');
      if (raw !== null && ahrsShowRaw) ahrsShowRaw.checked = raw === 'true';
    } catch {
      // ignore
    }
  }

  function wireControlState() {
    radarRange?.addEventListener('change', () => {
      try {
        localStorage.setItem('radar_range_nm', radarRange.value);
      } catch {}
    });
    mapFollow?.addEventListener('change', () => {
      try {
        localStorage.setItem('map_follow', String(!!mapFollow.checked));
      } catch {}
    });
    ahrsShowRaw?.addEventListener('change', () => {
      try {
        localStorage.setItem('ahrs_show_raw', String(!!ahrsShowRaw.checked));
      } catch {}
    });
  }

  async function loadSettings() {
    saveMsg.textContent = '';
    try {
      const resp = await fetch('/api/settings', { cache: 'no-store' });
      if (!resp.ok) throw new Error(`settings ${resp.status}`);
      const p = await resp.json();

      setGDL90Dest.value = p.gdl90_dest || '';
      setOwnshipEnable.checked = !!p.ownship_enable;
      setTrafficEnable.checked = !!p.traffic_enable;
      setScenarioEnable.checked = !!p.scenario_enable;
      setScenarioPath.value = p.scenario_path || '';
      setScenarioStart.value = p.scenario_start_time_utc || '';
      setScenarioLoop.checked = !!p.scenario_loop;
      setWebListen.value = p.web_listen || '';
      setWebEnable.checked = !!p.web_enable;
    } catch (e) {
      saveMsg.textContent = `Settings unavailable (${String(e)})`;
    }
  }

  async function saveSettings() {
    saveMsg.textContent = 'Saving…';
    const payload = {
      gdl90_dest: setGDL90Dest.value,
      ownship_enable: !!setOwnshipEnable.checked,
      traffic_enable: !!setTrafficEnable.checked,
      scenario_enable: !!setScenarioEnable.checked,
      scenario_path: setScenarioPath.value,
      scenario_start_time_utc: setScenarioStart.value,
      scenario_loop: !!setScenarioLoop.checked,
      web_listen: setWebListen.value,
      web_enable: !!setWebEnable.checked,
    };
    try {
      const resp = await fetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const text = await resp.text();
      if (!resp.ok) throw new Error(text || `save ${resp.status}`);
      saveMsg.textContent = 'Saved. Restart required.';
    } catch (e) {
      saveMsg.textContent = `Save failed: ${String(e)}`;
    }
  }

  async function poll() {
    try {
      const resp = await fetch('/api/status', { cache: 'no-store' });
      if (!resp.ok) throw new Error(`status ${resp.status}`);
      const s = await resp.json();
      setStatusText(s);
    } catch (e) {
      subtitle.textContent = `Disconnected (${String(e)})`;
    }
  }

  // Initial view.
  const initial = (location.hash || '#radar').slice(1);
  setView(['ahrs', 'radar', 'map', 'status', 'settings'].includes(initial) ? initial : 'radar');

  loadControlState();
  wireControlState();
  loadSettings();
  settingsForm?.addEventListener('submit', (e) => {
    e.preventDefault();
    saveSettings();
  });

  poll();
  setInterval(poll, 1000);
})();
