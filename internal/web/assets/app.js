(() => {
  const views = [
    { key: 'attitude', el: document.getElementById('view-attitude') },
    { key: 'radar', el: document.getElementById('view-radar') },
    { key: 'map', el: document.getElementById('view-map') },
    { key: 'status', el: document.getElementById('view-status') },
    { key: 'settings', el: document.getElementById('view-settings') },
    { key: 'logs', el: document.getElementById('view-logs') },
    { key: 'about', el: document.getElementById('view-about') },
  ];

  const subtitle = document.getElementById('subtitle');
  const drawer = document.getElementById('drawer');
  const drawerBackdrop = document.getElementById('drawer-backdrop');
  const btnMore = document.getElementById('btn-more');
  const btnClose = document.getElementById('btn-close');

  const stUptime = document.getElementById('st-uptime');
  const stNow = document.getElementById('st-now');
  const stLastTick = document.getElementById('st-last-tick');
  const stGDL90Dest = document.getElementById('st-gdl90-dest');
  const stInterval = document.getElementById('st-interval');
  const stFrames = document.getElementById('st-frames');
  const stScenario = document.getElementById('st-sim-scenario');
  const stTraffic = document.getElementById('st-sim-traffic');
  const stRecord = document.getElementById('st-record');
  const stReplay = document.getElementById('st-replay');

  const stGpsEnabled = document.getElementById('st-gps-enabled');
  const stGpsValid = document.getElementById('st-gps-valid');
  const stGpsStale = document.getElementById('st-gps-stale');
  const stGpsAge = document.getElementById('st-gps-age');
  const stGpsSource = document.getElementById('st-gps-source');
  const stGpsDevice = document.getElementById('st-gps-device');
  const stGpsGPSDAddr = document.getElementById('st-gps-gpsd-addr');
  const stGpsBaud = document.getElementById('st-gps-baud');
  const stGpsLastFix = document.getElementById('st-gps-last-fix');
  const stGpsFixQ = document.getElementById('st-gps-fixq');
  const stGpsFixMode = document.getElementById('st-gps-fixmode');
  const stGpsSats = document.getElementById('st-gps-sats');
  const stGpsHdop = document.getElementById('st-gps-hdop');
  const stGpsHAcc = document.getElementById('st-gps-hacc');
  const stGpsVAcc = document.getElementById('st-gps-vacc');
  const stGpsLat = document.getElementById('st-gps-lat');
  const stGpsLon = document.getElementById('st-gps-lon');
  const stGpsAlt = document.getElementById('st-gps-alt');
  const stGpsGround = document.getElementById('st-gps-ground');
  const stGpsTrack = document.getElementById('st-gps-track');
  const stGpsVSpeed = document.getElementById('st-gps-vspeed');
  const stGpsError = document.getElementById('st-gps-error');

  const stFanEnabled = document.getElementById('st-fan-enabled');
  const stFanCpuTemp = document.getElementById('st-fan-cpu-temp');
  const stFanDuty = document.getElementById('st-fan-duty');
  const stFanError = document.getElementById('st-fan-error');

  const stAhrsImuDetected = document.getElementById('st-ahrs-imu-detected');
  const stAhrsImuWorking = document.getElementById('st-ahrs-imu-working');
  const stAhrsImuUpdated = document.getElementById('st-ahrs-imu-updated');
  const stAhrsBaroDetected = document.getElementById('st-ahrs-baro-detected');
  const stAhrsBaroWorking = document.getElementById('st-ahrs-baro-working');
  const stAhrsBaroUpdated = document.getElementById('st-ahrs-baro-updated');
  const stAhrsError = document.getElementById('st-ahrs-error');
  const stAhrsOrientationSet = document.getElementById('st-ahrs-orientation-set');
  const stAhrsForwardAxis = document.getElementById('st-ahrs-forward-axis');
  const btnAhrsLevel = document.getElementById('btn-ahrs-level');
  const btnAhrsZeroDrift = document.getElementById('btn-ahrs-zero-drift');
  const btnAhrsOrientForward = document.getElementById('btn-ahrs-orient-forward');
  const btnAhrsOrientDone = document.getElementById('btn-ahrs-orient-done');
  const ahrsMsg = document.getElementById('ahrs-msg');

  const attValid = document.getElementById('att-valid');
  const attRoll = document.getElementById('att-roll');
  const attPitch = document.getElementById('att-pitch');
  const attHeading = document.getElementById('att-heading');
  const attUpdated = document.getElementById('att-updated');

  const settingsForm = document.getElementById('settings-form');
  const saveMsg = document.getElementById('save-msg');
  const setGDL90Dest = document.getElementById('set-gdl90-dest');
  const setIntervalInput = document.getElementById('set-interval');
  const setTrafficEnable = document.getElementById('set-traffic-enable');
  const setScenarioEnable = document.getElementById('set-scenario-enable');
  const setScenarioPath = document.getElementById('set-scenario-path');
  const setScenarioStart = document.getElementById('set-scenario-start');
  const setScenarioLoop = document.getElementById('set-scenario-loop');

  const logsText = document.getElementById('logs-text');
  const logsMeta = document.getElementById('logs-meta');
  const logsTail = document.getElementById('logs-tail');
  const logsRefresh = document.getElementById('logs-refresh');

  if (subtitle) subtitle.textContent = 'Connecting…';

  function timeNow() {
    try {
      return new Date().toLocaleTimeString();
    } catch {
      return '';
    }
  }

  function setInput(el, value) {
    if (!el) return;
    el.value = value == null ? '' : String(value);
  }

  function setChecked(el, value) {
    if (!el) return;
    el.checked = !!value;
  }

  function formatUptime(sec) {
    const n = Number(sec);
    if (!Number.isFinite(n) || n < 0) return '';
    const s = Math.floor(n);
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const ss = s % 60;
    const pad2 = (x) => String(x).padStart(2, '0');
    if (h > 0) return `${h}:${pad2(m)}:${pad2(ss)}`;
    return `${m}:${pad2(ss)}`;
  }

  async function loadLogs() {
    if (!logsText) return;

    const tail = parseInt(logsTail?.value || '200', 10) || 200;
    logsText.textContent = 'Loading…';
    if (logsMeta) logsMeta.textContent = '';

    try {
      const url = `/api/logs?format=text&tail=${encodeURIComponent(String(tail))}`;
      const resp = await fetch(url, { cache: 'no-store' });
      if (!resp.ok) {
        if (resp.status === 404) {
          logsText.textContent = 'Logs endpoint is not enabled in this build.';
        } else {
          logsText.textContent = `Failed to load logs: HTTP ${resp.status}`;
        }
        return;
      }
      const text = await resp.text();
      logsText.textContent = text || '(no logs)';

      // Optional: pull dropped/meta from JSON snapshot.
      try {
        const j = await fetch(`/api/logs?tail=${encodeURIComponent(String(tail))}`, { cache: 'no-store' });
        if (j.ok) {
          const js = await j.json();
          const dropped = (typeof js.dropped === 'number') ? js.dropped : 0;
          if (logsMeta) logsMeta.textContent = `Updated ${timeNow()}${dropped ? ` · dropped ${dropped}` : ''}`;
        } else if (logsMeta) {
          logsMeta.textContent = `Updated ${timeNow()}`;
        }
      } catch {
        if (logsMeta) logsMeta.textContent = `Updated ${timeNow()}`;
      }
    } catch (e) {
      logsText.textContent = `Failed to load logs: ${String(e)}`;
    }
  }


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

    if (key === 'logs') loadLogs();
    if (key === 'settings') loadSettings();
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

  function setStatusText(s) {
    const dest = s?.gdl90_dest || '';
    const interval = s?.interval || '';
    const frames = s?.frames_sent_total ?? 0;

    setInput(stUptime, formatUptime(s?.uptime_sec));
    setInput(stNow, s?.now_utc || '');
    setInput(stLastTick, s?.last_tick_utc || '');
    setInput(stGDL90Dest, dest);
    setInput(stInterval, interval);
    setInput(stFrames, frames);

    const sim = s?.sim || {};
    setChecked(stScenario, !!sim.scenario);
    setChecked(stTraffic, !!sim.traffic);
    setChecked(stRecord, !!sim.record);
    setChecked(stReplay, !!sim.replay);

    const gps = s?.gps || {};

    function fmtFixMode(gpsSource, mode) {
      const src = String(gpsSource || '').toLowerCase();
      if (src !== 'gpsd') return '-';
      if (mode == null) return '-';
      switch (Number(mode)) {
        case 1:
          return 'no fix';
        case 2:
          return '2d';
        case 3:
          return '3d';
        default:
          return String(mode);
      }
    }

    setChecked(stGpsEnabled, !!gps.enabled);
    setChecked(stGpsValid, !!gps.valid);
    setChecked(stGpsStale, !!gps.fix_stale);
    setInput(stGpsAge, gps.fix_age_sec == null ? '' : fmtNum(gps.fix_age_sec, 1));
    setInput(stGpsSource, gps.source || '');
    setInput(stGpsDevice, gps.device || '');
    setInput(stGpsGPSDAddr, gps.gpsd_addr || '');
    setInput(stGpsBaud, gps.baud == null ? '' : String(gps.baud));
    setInput(stGpsLastFix, gps.last_fix_utc || '');
    setInput(stGpsFixQ, gps.fix_quality == null ? '' : String(gps.fix_quality));
    setInput(stGpsFixMode, fmtFixMode(gps.source, gps.fix_mode));
    setInput(stGpsSats, gps.satellites == null ? '' : String(gps.satellites));
    setInput(stGpsHdop, gps.hdop == null ? '' : fmtNum(gps.hdop, 1));
    setInput(stGpsHAcc, gps.horiz_acc_m == null ? '' : fmtNum(gps.horiz_acc_m, 1));
    setInput(stGpsVAcc, gps.vert_acc_m == null ? '' : fmtNum(gps.vert_acc_m, 1));
    setInput(stGpsLat, gps.enabled && gps.valid ? fmtNum(gps.lat_deg, 6) : '');
    setInput(stGpsLon, gps.enabled && gps.valid ? fmtNum(gps.lon_deg, 6) : '');
    setInput(stGpsAlt, gps.alt_feet == null ? '' : String(gps.alt_feet));
    setInput(stGpsGround, gps.ground_kt == null ? '' : String(gps.ground_kt));
    setInput(stGpsTrack, gps.track_deg == null ? '' : fmtNum(gps.track_deg, 1));
    setInput(stGpsVSpeed, gps.vert_speed_fpm == null ? '' : String(gps.vert_speed_fpm));
    setInput(stGpsError, gps.last_error || '');

    const fan = s?.fan || {};
    setChecked(stFanEnabled, !!fan.enabled);
    setInput(stFanCpuTemp, fan.cpu_valid ? fmtNum(fan.cpu_temp_c, 1) : '');
    setInput(stFanDuty, fan.pwm_available ? String(fan.pwm_duty ?? '') : '');
    setInput(stFanError, fan.last_error || '');

    const ahrs = s?.ahrs || {};
    setChecked(stAhrsImuDetected, !!ahrs.imu_detected);
    setChecked(stAhrsImuWorking, !!ahrs.imu_working);
    setInput(stAhrsImuUpdated, ahrs.imu_last_update_utc || '');
    setChecked(stAhrsBaroDetected, !!ahrs.baro_detected);
    setChecked(stAhrsBaroWorking, !!ahrs.baro_working);
    setInput(stAhrsBaroUpdated, ahrs.baro_last_update_utc || '');
    setInput(stAhrsError, ahrs.last_error || '');

    setInput(stAhrsOrientationSet, ahrs.orientation_set ? 'true' : 'false');
    setInput(stAhrsForwardAxis, ahrs.forward_axis == null ? '' : String(ahrs.forward_axis));

    const enabled = !!ahrs.enabled;
    if (btnAhrsLevel) btnAhrsLevel.disabled = !enabled;
    if (btnAhrsZeroDrift) btnAhrsZeroDrift.disabled = !enabled;
    if (btnAhrsOrientForward) btnAhrsOrientForward.disabled = !enabled;
    if (btnAhrsOrientDone) btnAhrsOrientDone.disabled = !enabled;
  }

  async function postAhrs(path) {
    if (ahrsMsg) ahrsMsg.textContent = 'Working…';
    try {
      const resp = await fetch(path, { method: 'POST' });
      const text = await resp.text();
      if (!resp.ok) throw new Error(text || `HTTP ${resp.status}`);
      if (ahrsMsg) ahrsMsg.textContent = 'OK.';
    } catch (e) {
      if (ahrsMsg) ahrsMsg.textContent = `Failed: ${String(e)}`;
    }
  }

  function fmtNum(x, digits = 1) {
    const n = Number(x);
    if (!Number.isFinite(n)) return '';
    return n.toFixed(digits);
  }

  function setAttitudeText(s) {
    const a = s?.attitude || {};
    setInput(attValid, a.valid ? 'true' : 'false');
    setInput(attRoll, a.roll_deg == null ? '' : fmtNum(a.roll_deg, 1));
    setInput(attPitch, a.pitch_deg == null ? '' : fmtNum(a.pitch_deg, 1));
    setInput(attHeading, a.heading_deg == null ? '' : fmtNum(a.heading_deg, 1));
    setInput(attUpdated, a.last_update_utc || '');
  }

  async function loadSettings() {
    saveMsg.textContent = '';
    try {
      const resp = await fetch('/api/settings', { cache: 'no-store' });
      if (!resp.ok) throw new Error(`settings ${resp.status}`);
      const p = await resp.json();

      // Populate scenario list (best-effort).
      try {
        const sresp = await fetch('/api/scenarios', { cache: 'no-store' });
        if (sresp.ok) {
          const sj = await sresp.json();
          const paths = Array.isArray(sj?.paths) ? sj.paths : [];

          if (setScenarioPath) {
            const current = p.scenario_path || '';

            // Preserve the placeholder option.
            const placeholder = setScenarioPath.querySelector('option[value=""]');
            setScenarioPath.innerHTML = '';
            if (placeholder) {
              setScenarioPath.appendChild(placeholder);
            } else {
              const opt = document.createElement('option');
              opt.value = '';
              opt.textContent = '(select a scenario)';
              setScenarioPath.appendChild(opt);
            }

            for (const path of paths) {
              const opt = document.createElement('option');
              opt.value = String(path);
              const parts = String(path).split('/');
              opt.textContent = parts[parts.length - 1] || String(path);
              setScenarioPath.appendChild(opt);
            }

            // If config points at a path not in the list, keep it selectable.
            if (current && !paths.includes(current)) {
              const opt = document.createElement('option');
              opt.value = current;
              opt.textContent = current;
              setScenarioPath.appendChild(opt);
            }
          }
        }
      } catch {
        // ignore
      }

      setGDL90Dest.value = p.gdl90_dest || '';
      if (setIntervalInput) setIntervalInput.value = p.interval || '';
      setTrafficEnable.checked = !!p.traffic_enable;
      setScenarioEnable.checked = !!p.scenario_enable;
      if (setScenarioPath) setScenarioPath.value = p.scenario_path || '';
      setScenarioStart.value = p.scenario_start_time_utc || '';
      setScenarioLoop.checked = !!p.scenario_loop;
    } catch (e) {
      saveMsg.textContent = `Settings unavailable (${String(e)})`;
    }
  }

  async function saveSettings() {
    saveMsg.textContent = 'Saving…';
    const payload = {
      gdl90_dest: setGDL90Dest.value,
      interval: setIntervalInput ? setIntervalInput.value : '',
      traffic_enable: !!setTrafficEnable.checked,
      scenario_enable: !!setScenarioEnable.checked,
      scenario_path: setScenarioPath.value,
      scenario_start_time_utc: setScenarioStart.value,
      scenario_loop: !!setScenarioLoop.checked,
    };
    try {
      const resp = await fetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const text = await resp.text();
      if (!resp.ok) throw new Error(text || `save ${resp.status}`);
      saveMsg.textContent = 'Saved and applied.';
    } catch (e) {
      saveMsg.textContent = `Save failed: ${String(e)}`;
    }
  }

  async function poll() {
    try {
      const resp = await fetch('/api/status', { cache: 'no-store' });
      if (!resp.ok) {
        if (subtitle) subtitle.textContent = 'Disconnected';
        return;
      }
      const s = await resp.json();
      setStatusText(s);
      setAttitudeText(s);
      if (subtitle) subtitle.textContent = 'Connected';
    } catch {
      if (subtitle) subtitle.textContent = 'Disconnected';
    }
  }

  // Initial view.
  const initial = (location.hash || '#map').slice(1);
  setView(['attitude', 'radar', 'map', 'status', 'settings', 'logs', 'about'].includes(initial) ? initial : 'map');
  logsRefresh?.addEventListener('click', loadLogs);
  logsTail?.addEventListener('change', loadLogs);

  loadSettings();
  settingsForm?.addEventListener('submit', (e) => {
    e.preventDefault();
    saveSettings();
  });

  poll();
  setInterval(poll, 1000);

  btnAhrsLevel?.addEventListener('click', () => postAhrs('/api/ahrs/level'));
  btnAhrsZeroDrift?.addEventListener('click', () => postAhrs('/api/ahrs/zero-drift'));
  btnAhrsOrientForward?.addEventListener('click', () => postAhrs('/api/ahrs/orient/forward'));
  btnAhrsOrientDone?.addEventListener('click', () => postAhrs('/api/ahrs/orient/done'));
})();
