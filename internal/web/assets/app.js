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

  const btnAttAhrsLevel = document.getElementById('btn-att-ahrs-level');
  const btnAttAhrsZeroDrift = document.getElementById('btn-att-ahrs-zero-drift');
  const attAhrsMsg = document.getElementById('att-ahrs-msg');

  const attValid = document.getElementById('att-valid');
  const attRoll = document.getElementById('att-roll');
  const attPitch = document.getElementById('att-pitch');
  const attHeading = document.getElementById('att-heading');
  const attPalt = document.getElementById('att-palt');
  const attGps = document.getElementById('att-gps');
  const attG = document.getElementById('att-g');
  const attGMin = document.getElementById('att-gmin');
  const attGMax = document.getElementById('att-gmax');
  const attCanvas = document.getElementById('att-canvas');
  const attCtx = attCanvas ? attCanvas.getContext('2d') : null;

  const attTape = document.getElementById('att-hdg-tape');
  const attTapeCtx = attTape ? attTape.getContext('2d') : null;

  const attLeftTape = document.getElementById('att-tape-left');
  const attLeftTapeCtx = attLeftTape ? attLeftTape.getContext('2d') : null;
  const attRightTape = document.getElementById('att-tape-right');
  const attRightTapeCtx = attRightTape ? attRightTape.getContext('2d') : null;

  // Map UI.
  const mapLeafletEl = document.getElementById('map-leaflet');
  const mapMsg = document.getElementById('map-msg');
  const mapInfo = document.getElementById('map-info');

  let leafletMap = null;
  let ownshipMarker = null;
  let ownshipTrack = null;
  let followOwnship = true;
  let lastGoodLatLng = null;
  let trackPoints = [];
  let hasAutoCentered = false;
  const defaultOwnshipZoom = 12;

  function centerOnOwnship() {
    followOwnship = true;
    if (!leafletMap || !lastGoodLatLng) return;
    try {
      leafletMap.setView(lastGoodLatLng, defaultOwnshipZoom, { animate: false });
    } catch {
      // ignore
    }
  }

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
    const s = value == null ? '' : String(value);
    // Support both <input> and text elements (<span>/<div>).
    if ('value' in el) {
      el.value = s;
    } else {
      el.textContent = s;
    }
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

    document.body.classList.toggle('map-active', key === 'map');

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
    if (key === 'map') {
      initMapIfNeeded();
      // If the map was initialized while hidden, force Leaflet to compute sizes.
      try {
        leafletMap?.invalidateSize?.();
      } catch {
        // ignore
      }
    }
  }

  function setMapMessage(s) {
    if (!mapMsg) return;
    mapMsg.textContent = String(s || '');
  }

  function fmtDeg(x, digits) {
    const n = Number(x);
    if (!Number.isFinite(n)) return '--';
    return n.toFixed(digits);
  }

  function setMapHud(gps) {
    if (!mapInfo) return;
    if (!gps) {
      mapInfo.textContent = '--';
      return;
    }

    let gpsStatus = '--';
    if (gps.enabled) {
      if (!gps.valid) {
        gpsStatus = 'NO FIX';
      } else if (gps.fix_stale) {
        gpsStatus = 'STALE';
      } else {
        gpsStatus = 'OK';
      }
    }

    const lat = (gps.enabled && gps.valid) ? fmtDeg(gps.lat_deg, 5) : '--';
    const lon = (gps.enabled && gps.valid) ? fmtDeg(gps.lon_deg, 5) : '--';
    const alt = gps.alt_feet == null ? '--' : `${String(gps.alt_feet)}ft`;
    const gs = gps.ground_kt == null ? '--' : `${String(gps.ground_kt)}kt`;
    const trk = gps.track_deg == null ? '--' : `${fmtNum(gps.track_deg, 0)}°`;

    mapInfo.textContent = `GPS ${gpsStatus} | LAT ${lat} | LON ${lon} | ALT ${alt} | GS ${gs} | TRK ${trk}`;
  }

  function initMapIfNeeded() {
    if (leafletMap || !mapLeafletEl) return;

    if (!window.L || typeof window.L.map !== 'function') {
      setMapMessage('Map unavailable (Leaflet failed to load).');
      return;
    }

    setMapMessage('');

    leafletMap = window.L.map(mapLeafletEl, {
      zoomControl: true,
      attributionControl: true,
      preferCanvas: true,
    });

    // Start at a sensible world view.
    leafletMap.setView([0, 0], 2);

    // Stratux-style: keep it simple with OSM tiles.
    window.L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      crossOrigin: true,
      attribution: '&copy; OpenStreetMap contributors',
    }).addTo(leafletMap);

    const accent = cssVar('--accent') || '#7dd3fc';

    ownshipMarker = window.L.circleMarker([0, 0], {
      radius: 7,
      color: accent,
      weight: 2,
      fillColor: accent,
      fillOpacity: 0.35,
    }).addTo(leafletMap);

    ownshipTrack = window.L.polyline([], {
      color: accent,
      weight: 2,
      opacity: 0.75,
    }).addTo(leafletMap);

    // Add a Center control under Leaflet's +/- zoom control.
    try {
      const CenterControl = window.L.Control.extend({
        options: { position: 'topleft' },
        onAdd: function () {
          const container = window.L.DomUtil.create('div', 'leaflet-control map-center-control');
          const a = window.L.DomUtil.create('a', '', container);
          a.href = '#';
          a.title = 'Center';
          a.setAttribute('aria-label', 'Center on ownship');
          a.setAttribute('role', 'button');
          a.textContent = '⦿';

          window.L.DomEvent.disableClickPropagation(container);
          window.L.DomEvent.on(a, 'click', (e) => {
            window.L.DomEvent.preventDefault(e);
            centerOnOwnship();
          });
          return container;
        },
      });
      leafletMap.addControl(new CenterControl());
    } catch {
      // ignore
    }

    // If user pans/zooms, stop auto-follow until Center is pressed.
    leafletMap.on('dragstart', () => { followOwnship = false; });
    leafletMap.on('zoomstart', () => { followOwnship = false; });
  }

  function updateMapFromGPS(gps) {
    setMapHud(gps);
    if (!leafletMap || !ownshipMarker || !ownshipTrack) return;

    if (!gps || !gps.enabled) {
      setMapMessage('GPS disabled.');
      return;
    }
    if (!gps.valid) {
      setMapMessage('Waiting for GPS fix…');
      return;
    }

    const lat = Number(gps.lat_deg);
    const lon = Number(gps.lon_deg);
    if (!Number.isFinite(lat) || !Number.isFinite(lon)) {
      setMapMessage('Waiting for valid position…');
      return;
    }

    setMapMessage('');
    lastGoodLatLng = [lat, lon];

    try {
      ownshipMarker.setLatLng(lastGoodLatLng);
    } catch {
      // ignore
    }

    // Track: keep a small rolling buffer.
    const maxTrack = 600;
    const last = trackPoints.length ? trackPoints[trackPoints.length - 1] : null;
    const moved = !last || Math.abs(last[0] - lat) > 1e-6 || Math.abs(last[1] - lon) > 1e-6;
    if (moved) {
      trackPoints.push([lat, lon]);
      if (trackPoints.length > maxTrack) trackPoints = trackPoints.slice(trackPoints.length - maxTrack);
      try {
        ownshipTrack.setLatLngs(trackPoints);
      } catch {
        // ignore
      }
    }

    // Initial: center on ownship at the default zoom.
    if (!hasAutoCentered) {
      hasAutoCentered = true;
      try {
        centerOnOwnship();
      } catch {
        // ignore
      }
    }

    if (followOwnship) {
      try {
        const z = leafletMap.getZoom?.();
        leafletMap.setView(lastGoodLatLng, Number.isFinite(z) ? z : 12, { animate: false });
      } catch {
        // ignore
      }
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

    if (btnAttAhrsLevel) btnAttAhrsLevel.disabled = !enabled;
    if (btnAttAhrsZeroDrift) btnAttAhrsZeroDrift.disabled = !enabled;
  }

  let attAhrsBusyCount = 0;

  function setAttAhrsBusy(isBusy) {
    if (isBusy) {
      attAhrsBusyCount++;
    } else {
      attAhrsBusyCount = Math.max(0, attAhrsBusyCount - 1);
    }
    drawAttitude();
  }

  async function postAhrs(path, msgEl = ahrsMsg) {
    const isAttitudeUi = msgEl === attAhrsMsg;
    if (isAttitudeUi) {
      if (msgEl) msgEl.textContent = '';
      setAttAhrsBusy(true);
    } else {
      if (msgEl) msgEl.textContent = 'Working…';
    }
    try {
      const resp = await fetch(path, { method: 'POST' });
      const text = await resp.text();
      if (!resp.ok) throw new Error(text || `HTTP ${resp.status}`);
      if (!isAttitudeUi) {
        if (msgEl) msgEl.textContent = 'OK.';
      } else {
        if (msgEl) msgEl.textContent = '';
      }
    } catch (e) {
      if (msgEl) msgEl.textContent = `Failed: ${String(e)}`;
    } finally {
      if (isAttitudeUi) {
        setAttAhrsBusy(false);
      }
    }
  }

  function fmtNum(x, digits = 1) {
    const n = Number(x);
    if (!Number.isFinite(n)) return '';
    return n.toFixed(digits);
  }

  function clamp(x, lo, hi) {
    const n = Number(x);
    if (!Number.isFinite(n)) return lo;
    return Math.min(hi, Math.max(lo, n));
  }

  function hexToRgb(hex) {
    const s = String(hex || '').trim();
    if (!s.startsWith('#')) return null;
    const h = s.slice(1);
    if (h.length === 3) {
      const r = parseInt(h[0] + h[0], 16);
      const g = parseInt(h[1] + h[1], 16);
      const b = parseInt(h[2] + h[2], 16);
      if ([r, g, b].some((v) => Number.isNaN(v))) return null;
      return { r, g, b };
    }
    if (h.length === 6) {
      const r = parseInt(h.slice(0, 2), 16);
      const g = parseInt(h.slice(2, 4), 16);
      const b = parseInt(h.slice(4, 6), 16);
      if ([r, g, b].some((v) => Number.isNaN(v))) return null;
      return { r, g, b };
    }
    return null;
  }

  function cssVar(name) {
    try {
      return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    } catch {
      return '';
    }
  }

  function cssVarRGBA(name, alpha) {
    const v = cssVar(name);
    const rgb = hexToRgb(v);
    if (!rgb) return v || '';
    return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${String(alpha)})`;
  }

  let lastAttitude = null;
  let lastGps = null;

  function fitSideTapeCanvas(canvas, ctx) {
    if (!canvas || !ctx) return null;
    const rect = canvas.getBoundingClientRect();
    const cssW = Math.max(1, Math.floor(rect.width || 0));
    const cssH = Math.max(1, Math.floor(rect.height || 0));
    const dpr = Math.max(1, Math.floor((window.devicePixelRatio || 1) * 100) / 100);
    const pxW = Math.max(1, Math.floor(cssW * dpr));
    const pxH = Math.max(1, Math.floor(cssH * dpr));
    if (canvas.width !== pxW || canvas.height !== pxH) {
      canvas.width = pxW;
      canvas.height = pxH;
    }
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    return { w: cssW, h: cssH };
  }

  function fmtInt(n) {
    const x = Number(n);
    if (!Number.isFinite(x)) return '';
    const v = Math.round(x);
    try {
      return v.toLocaleString(undefined);
    } catch {
      return String(v);
    }
  }

  function isMultiple(value, step) {
    const v = Number(value);
    const s = Number(step);
    if (!Number.isFinite(v) || !Number.isFinite(s) || s === 0) return false;
    return Math.abs(v / s - Math.round(v / s)) < 1e-9;
  }

  function drawVerticalTape(canvas, ctx, opts) {
    if (!canvas || !ctx) return;
    const size = fitSideTapeCanvas(canvas, ctx);
    if (!size) return;
    const w = size.w;
    const h = size.h;

    const title = String(opts?.title || '');
    const side = (opts?.side === 'right') ? 'right' : 'left';
    const value = Number(opts?.value);
    const hasValue = Number.isFinite(value);
    const range = Number(opts?.range || 0);
    const halfRange = Number.isFinite(range) && range > 0 ? range / 2 : 0;
    const minorStep = Number(opts?.minorStep || 1);
    const majorStep = Number(opts?.majorStep || minorStep);
    const labelStep = Number(opts?.labelStep || majorStep);

    const surface = cssVar('--surface') || '#111111';
    const surface2 = cssVar('--surface2') || '#222222';
    const text = cssVar('--text') || '#ffffff';
    const muted = cssVar('--muted') || text;
    const accent = cssVar('--accent') || text;

    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = surface2;
    ctx.fillRect(0, 0, w, h);

    const pad = 6;
    const titlePx = Math.max(10, Math.floor(w * 0.22));
    const titleH = titlePx + 10;

    const top = pad + titleH;
    const bottom = h - pad;
    const tapeH = Math.max(1, bottom - top);
    const centerY = top + tapeH / 2;

    // Title.
    ctx.fillStyle = muted;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    ctx.font = `${titlePx}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
    ctx.fillText(title, w / 2, pad);

    // Rolling scale.
    if (hasValue && halfRange > 0 && Number.isFinite(minorStep) && minorStep > 0) {
      const pxPerUnit = tapeH / (halfRange * 2);

      const startVal = Math.floor((value - halfRange) / minorStep) * minorStep;
      const endVal = Math.ceil((value + halfRange) / minorStep) * minorStep;

      const tickX0 = side === 'left' ? w - pad : pad;
      const tickDir = side === 'left' ? -1 : 1;

      const minorLen = Math.max(8, Math.floor(w * 0.20));
      const majorLen = Math.max(12, Math.floor(w * 0.36));
      const labelPx = Math.max(11, Math.floor(w * 0.22));

      // Clip to tape area so labels/ticks don't overlap title.
      ctx.save();
      ctx.beginPath();
      ctx.rect(0, top, w, tapeH);
      ctx.clip();

      for (let v = startVal; v <= endVal + 1e-9; v += minorStep) {
        const y = centerY - (v - value) * pxPerUnit;
        if (y < top - 20 || y > bottom + 20) continue;

        const major = isMultiple(v, majorStep);
        const label = major && isMultiple(v, labelStep);
        const len = major ? majorLen : minorLen;

        ctx.strokeStyle = major ? text : muted;
        ctx.lineWidth = Math.max(1, w * 0.03);
        ctx.beginPath();
        ctx.moveTo(tickX0, y);
        ctx.lineTo(tickX0 + tickDir * len, y);
        ctx.stroke();

        if (label) {
          ctx.fillStyle = text;
          ctx.font = `${labelPx}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
          ctx.textBaseline = 'middle';
          if (side === 'left') {
            ctx.textAlign = 'right';
            ctx.fillText(fmtInt(v), tickX0 + tickDir * (len + 6), y);
          } else {
            ctx.textAlign = 'left';
            ctx.fillText(fmtInt(v), tickX0 + tickDir * (len + 6), y);
          }
        }
      }

      ctx.restore();
    }

    // Value window.
    const winH = clamp(Math.floor(tapeH * 0.20), 38, 72);
    const winW = Math.max(1, w - pad * 2);
    const winX = pad;
    const winY = centerY - winH / 2;

    ctx.fillStyle = cssVarRGBA('--surface', 0.92) || surface;
    ctx.fillRect(winX, winY, winW, winH);

    ctx.strokeStyle = accent;
    ctx.lineWidth = Math.max(1, w * 0.035);
    ctx.strokeRect(winX + 0.5, winY + 0.5, winW - 1, winH - 1);

    // Pointer triangle.
    const tri = Math.max(8, Math.floor(w * 0.18));
    ctx.fillStyle = accent;
    ctx.beginPath();
    if (side === 'left') {
      ctx.moveTo(w - 1, centerY);
      ctx.lineTo(w - 1 - tri, centerY - tri * 0.6);
      ctx.lineTo(w - 1 - tri, centerY + tri * 0.6);
    } else {
      ctx.moveTo(1, centerY);
      ctx.lineTo(1 + tri, centerY - tri * 0.6);
      ctx.lineTo(1 + tri, centerY + tri * 0.6);
    }
    ctx.closePath();
    ctx.fill();

    // Value text.
    ctx.fillStyle = text;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    const valPx = Math.max(16, Math.floor(w * 0.40));
    ctx.font = `${valPx}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
    ctx.fillText(hasValue ? fmtInt(value) : '---', w / 2, centerY);
  }

  function fitTapeCanvas() {
    if (!attTape || !attTapeCtx) return null;

    const rect = attTape.getBoundingClientRect();
    const cssW = Math.max(1, Math.floor(rect.width || 0));
    const cssH = Math.max(1, Math.floor(rect.height || 0));
    const dpr = Math.max(1, Math.floor((window.devicePixelRatio || 1) * 100) / 100);

    const pxW = Math.max(1, Math.floor(cssW * dpr));
    const pxH = Math.max(1, Math.floor(cssH * dpr));
    if (attTape.width !== pxW || attTape.height !== pxH) {
      attTape.width = pxW;
      attTape.height = pxH;
    }
    attTapeCtx.setTransform(dpr, 0, 0, dpr, 0, 0);
    return { w: cssW, h: cssH };
  }

  function wrap360(deg) {
    let x = Number(deg);
    if (!Number.isFinite(x)) return 0;
    x = x % 360;
    if (x < 0) x += 360;
    return x;
  }

  function headingLabel(deg) {
    const d = wrap360(deg);
    const n = Math.round(d / 10) * 10;
    const v = wrap360(n);
    if (v === 0) return 'N';
    if (v === 90) return 'E';
    if (v === 180) return 'S';
    if (v === 270) return 'W';
    return String(Math.round(v));
  }

  function drawHeadingTape() {
    if (!attTape || !attTapeCtx) return;
    const size = fitTapeCanvas();
    if (!size) return;

    const w = size.w;
    const h = size.h;
    const ctx = attTapeCtx;

    const surface2 = cssVar('--surface2') || '#222222';
    const text = cssVar('--text') || '#ffffff';
    const muted = cssVar('--muted') || text;
    const accent = cssVar('--accent') || text;

    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = surface2;
    ctx.fillRect(0, 0, w, h);

    const a = lastAttitude || {};
    const valid = !!a.valid;
    const hdg = valid && a.heading_deg != null ? wrap360(a.heading_deg) : null;

    const mid = w / 2;
    const pxPerDeg = w / 120; // show ~120° window
    const tickBase = h * 0.70;
    const longTick = h * 0.32;
    const shortTick = h * 0.20;
    const fontPx = Math.max(11, Math.floor(h * 0.26));
    ctx.font = `${fontPx}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';

    // Center marker.
    ctx.strokeStyle = accent;
    ctx.lineWidth = Math.max(2, h * 0.06);
    ctx.beginPath();
    ctx.moveTo(mid, 0);
    ctx.lineTo(mid, h);
    ctx.stroke();

    if (hdg == null) {
      ctx.fillStyle = text;
      ctx.fillText('HDG ---', mid, h * 0.45);
      return;
    }

    // Ticks +/- 60 degrees.
    ctx.strokeStyle = muted;
    ctx.lineWidth = Math.max(1, h * 0.04);
    for (let d = -60; d <= 60; d += 5) {
      const x = mid + d * pxPerDeg;
      const abs = Math.abs(d);
      const isMajor = abs % 10 === 0;
      const isLabel = abs % 30 === 0;
      const len = isMajor ? longTick : shortTick;
      ctx.beginPath();
      ctx.moveTo(x, tickBase);
      ctx.lineTo(x, tickBase - len);
      ctx.stroke();
      if (isLabel) {
        ctx.fillStyle = text;
        ctx.fillText(headingLabel(hdg + d), x, h * 0.28);
      }
    }

    // Center numeric heading.
    ctx.fillStyle = text;
    const hdgStr = String(Math.round(hdg)).padStart(3, '0');
    ctx.fillText(hdgStr, mid, h * 0.88);
  }

  function fitAttitudeCanvas() {
    if (!attCanvas || !attCtx) return null;

    const rect = attCanvas.getBoundingClientRect();
    const cssW = Math.max(1, Math.floor(rect.width || 0));
    const cssH = Math.max(1, Math.floor(rect.height || 0));
    const dpr = Math.max(1, Math.floor((window.devicePixelRatio || 1) * 100) / 100);

    const pxW = Math.max(1, Math.floor(cssW * dpr));
    const pxH = Math.max(1, Math.floor(cssH * dpr));
    if (attCanvas.width !== pxW || attCanvas.height !== pxH) {
      attCanvas.width = pxW;
      attCanvas.height = pxH;
    }
    attCtx.setTransform(dpr, 0, 0, dpr, 0, 0);
    return { w: cssW, h: cssH };
  }

  function drawAttitude() {
    if (!attCanvas || !attCtx) return;
    const size = fitAttitudeCanvas();
    if (!size) return;

    const w = size.w;
    const h = size.h;
    const ctx = attCtx;
    ctx.clearRect(0, 0, w, h);

    const bg = cssVar('--bg') || '#000000';
    const surface = cssVar('--surface') || '#111111';
    const surface2 = cssVar('--surface2') || '#222222';
    const text = cssVar('--text') || '#ffffff';
    const muted = cssVar('--muted') || text;
    const sky = cssVarRGBA('--accent', 0.22) || surface2;

    // Frame background (outside bezel).
    ctx.fillStyle = bg;
    ctx.fillRect(0, 0, w, h);

    const cx = w / 2;
    const cy = h / 2;
    const r = Math.max(10, Math.min(w, h) * 0.46);

    // Bezel.
    ctx.save();
    ctx.translate(cx, cy);
    ctx.beginPath();
    ctx.arc(0, 0, r + 6, 0, Math.PI * 2);
    ctx.fillStyle = surface;
    ctx.fill();
    ctx.restore();

    // Instrument face.
    ctx.save();
    ctx.translate(cx, cy);
    ctx.beginPath();
    ctx.arc(0, 0, r, 0, Math.PI * 2);
    ctx.clip();

    const a = lastAttitude || {};
    const valid = !!a.valid;
    const roll = valid && a.roll_deg != null ? clamp(a.roll_deg, -90, 90) : 0;
    const pitch = valid && a.pitch_deg != null ? clamp(a.pitch_deg, -45, 45) : 0;

    // Background rotates opposite aircraft roll.
    const rollRad = (-roll * Math.PI) / 180;
    const pxPerDeg = r / 40;

    ctx.save();
    ctx.rotate(rollRad);
    ctx.translate(0, pitch * pxPerDeg);

    // Sky / ground.
    ctx.fillStyle = sky;
    ctx.fillRect(-r * 2, -r * 2, r * 4, r * 2);
    ctx.fillStyle = surface2;
    ctx.fillRect(-r * 2, 0, r * 4, r * 2);

    // Horizon line.
    ctx.strokeStyle = text;
    ctx.lineWidth = Math.max(2, r * 0.012);
    ctx.beginPath();
    ctx.moveTo(-r * 2, 0);
    ctx.lineTo(r * 2, 0);
    ctx.stroke();

    // Pitch ladder.
    ctx.lineWidth = Math.max(1, r * 0.008);
    ctx.strokeStyle = text;
    ctx.fillStyle = text;
    const fontPx = Math.max(10, Math.floor(r * 0.10));
    ctx.font = `${fontPx}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
    ctx.textBaseline = 'middle';

    for (let deg = -30; deg <= 30; deg += 5) {
      if (deg === 0) continue;
      const y = -deg * pxPerDeg;
      const major = deg % 10 === 0;
      const halfLen = major ? r * 0.45 : r * 0.28;
      ctx.beginPath();
      ctx.moveTo(-halfLen, y);
      ctx.lineTo(halfLen, y);
      ctx.stroke();

      if (major) {
        const label = String(Math.abs(deg));
        ctx.fillText(label, halfLen + 8, y);
        const m = ctx.measureText(label);
        ctx.fillText(label, -halfLen - 8 - m.width, y);
      }
    }

    ctx.restore();

    // Center aircraft symbol.
    ctx.strokeStyle = text;
    ctx.lineWidth = Math.max(2, r * 0.014);
    ctx.lineCap = 'round';
    ctx.beginPath();
    ctx.moveTo(-r * 0.42, 0);
    ctx.lineTo(-r * 0.12, 0);
    ctx.moveTo(r * 0.12, 0);
    ctx.lineTo(r * 0.42, 0);
    ctx.moveTo(0, 0);
    ctx.lineTo(0, r * 0.10);
    ctx.stroke();

    // Roll scale (ticks at top arc).
    ctx.save();
    ctx.strokeStyle = muted;
    ctx.lineWidth = Math.max(1, r * 0.010);
    for (let deg = -60; deg <= 60; deg += 10) {
      const ang = (-Math.PI / 2) + (deg * Math.PI) / 180;
      const outer = r * 0.98;
      const inner = outer - (deg % 30 === 0 ? r * 0.10 : r * 0.06);
      ctx.beginPath();
      ctx.moveTo(Math.cos(ang) * inner, Math.sin(ang) * inner);
      ctx.lineTo(Math.cos(ang) * outer, Math.sin(ang) * outer);
      ctx.stroke();
    }

    // Bank pointer.
    ctx.fillStyle = text;
    ctx.beginPath();
    ctx.moveTo(0, -r * 1.02);
    ctx.lineTo(-r * 0.05, -r * 0.92);
    ctx.lineTo(r * 0.05, -r * 0.92);
    ctx.closePath();
    ctx.fill();
    ctx.restore();

    // If AHRS not valid, overlay a clear message.
    if (!valid) {
      ctx.fillStyle = cssVarRGBA('--surface', 0.85) || surface;
      ctx.fillRect(-r, -fontPx * 1.4, r * 2, fontPx * 2.0);
      ctx.fillStyle = text;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.font = `${Math.max(12, Math.floor(r * 0.12))}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
      ctx.fillText('WARMING UP / CALIBRATING', 0, 0);
      ctx.textAlign = 'start';
      ctx.textBaseline = 'alphabetic';
    }

    // Bezel border.
    ctx.restore();
    ctx.save();
    ctx.translate(cx, cy);
    ctx.strokeStyle = muted;
    ctx.lineWidth = Math.max(1, r * 0.010);
    ctx.beginPath();
    ctx.arc(0, 0, r, 0, Math.PI * 2);
    ctx.stroke();

    // AHRS action running (Set Level / Zero Drift): draw a big X.
    if (attAhrsBusyCount > 0) {
      const danger = cssVar('--danger') || text;
      ctx.strokeStyle = danger;
      ctx.lineWidth = Math.max(3, r * 0.025);
      ctx.lineCap = 'round';
      const x = r * 0.70;
      const y = r * 0.70;
      ctx.beginPath();
      ctx.moveTo(-x, -y);
      ctx.lineTo(x, y);
      ctx.moveTo(-x, y);
      ctx.lineTo(x, -y);
      ctx.stroke();
    }
    ctx.restore();

    // Separate bottom heading tape.
    drawHeadingTape();

    // Side tapes.
    const gps = lastGps || {};
    drawVerticalTape(attLeftTape, attLeftTapeCtx, {
      side: 'left',
      title: 'GS',
      value: gps.ground_kt,
      range: 40,     // +/- 20 kt window
      minorStep: 1,
      majorStep: 5,
      labelStep: 10,
    });

    drawVerticalTape(attRightTape, attRightTapeCtx, {
      side: 'right',
      title: 'GALT',
      value: gps.alt_feet,
      range: 1000,   // +/- 500 ft window
      minorStep: 20,
      majorStep: 100,
      labelStep: 500,
    });
  }

  function setAttitudeText(s) {
    const a = s?.attitude || {};
    const g = s?.gps || {};
    setInput(attValid, a.valid ? 'OK' : '--');
    setInput(attRoll, a.roll_deg == null ? '' : `${fmtNum(a.roll_deg, 1)}°`);
    setInput(attPitch, a.pitch_deg == null ? '' : `${fmtNum(a.pitch_deg, 1)}°`);
    setInput(attHeading, a.heading_deg == null ? '' : String(Math.round(Number(a.heading_deg))).padStart(3, '0'));
    setInput(attPalt, a.pressure_alt_ft == null ? '' : String(Math.round(Number(a.pressure_alt_ft))));

    setInput(attG, a.g_load == null ? '--' : fmtNum(a.g_load, 2));
    setInput(attGMin, a.g_min == null ? '--' : fmtNum(a.g_min, 2));
    setInput(attGMax, a.g_max == null ? '--' : fmtNum(a.g_max, 2));

    let gpsStatus = '--';
    if (g.enabled) {
      if (!g.valid) {
        gpsStatus = 'NO FIX';
      } else if (g.fix_stale) {
        gpsStatus = 'STALE';
      } else {
        gpsStatus = 'OK';
      }
    }
    setInput(attGps, gpsStatus);
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
      lastAttitude = s?.attitude || null;
      lastGps = s?.gps || null;
      drawAttitude();
      // Map updates are driven off the same poll.
      initMapIfNeeded();
      updateMapFromGPS(lastGps);
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
  // Redraw the instrument a bit faster than the status poll so it stays crisp on resize/theme changes.
  setInterval(drawAttitude, 100);
  window.addEventListener('resize', drawAttitude);

  btnAhrsLevel?.addEventListener('click', () => postAhrs('/api/ahrs/level', ahrsMsg));
  btnAhrsZeroDrift?.addEventListener('click', () => postAhrs('/api/ahrs/zero-drift', ahrsMsg));
  btnAhrsOrientForward?.addEventListener('click', () => postAhrs('/api/ahrs/orient/forward', ahrsMsg));
  btnAhrsOrientDone?.addEventListener('click', () => postAhrs('/api/ahrs/orient/done', ahrsMsg));

  btnAttAhrsLevel?.addEventListener('click', () => postAhrs('/api/ahrs/level', attAhrsMsg));
  btnAttAhrsZeroDrift?.addEventListener('click', () => postAhrs('/api/ahrs/zero-drift', attAhrsMsg));
})();
