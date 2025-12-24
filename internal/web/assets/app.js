(() => {
  const views = [
    { key: 'attitude', el: document.getElementById('view-attitude') },
    { key: 'radar', el: document.getElementById('view-radar') },
    { key: 'map', el: document.getElementById('view-map') },
    { key: 'status', el: document.getElementById('view-status') },
    { key: 'traffic', el: document.getElementById('view-traffic') },
    { key: 'weather', el: document.getElementById('view-weather') },
    { key: 'towers', el: document.getElementById('view-towers') },
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

  const stNetLocalAddrs = document.getElementById('st-net-local-addrs');
  const stNetClientsCount = document.getElementById('st-net-clients-count');
  const stNetClients = document.getElementById('st-net-clients');

  const stDiskTotal = document.getElementById('st-disk-total');
  const stDiskAvail = document.getElementById('st-disk-avail');
  const stDiskFree = document.getElementById('st-disk-free');
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
  const stFanBackend = document.getElementById('st-fan-backend');
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

  // Traffic / Weather pages.
  const trCount = document.getElementById('tr-count');
  const trList = document.getElementById('tr-list');
  const trAdsb1090 = document.getElementById('tr-adsb1090');
  const trUat978 = document.getElementById('tr-uat978');

  const wxRawEndpoint = document.getElementById('wx-raw-endpoint');
  const wxRawState = document.getElementById('wx-raw-state');
  const wxRawLines = document.getElementById('wx-raw-lines');
  const wxRawLastSeen = document.getElementById('wx-raw-last-seen');
  const wxRawError = document.getElementById('wx-raw-error');

  // Map UI.
  const mapLeafletEl = document.getElementById('map-leaflet');
  const mapMsg = document.getElementById('map-msg');
  const mapInfo = document.getElementById('map-info');
  const mapTrafficBox = document.getElementById('map-trafficbox');
  const mapTrafficBoxLines = document.getElementById('map-trafficbox-lines');

  // Radar UI (traffic radar).
  const radarCanvas = document.getElementById('radar-canvas');
  const radarCtx = radarCanvas ? radarCanvas.getContext('2d') : null;
  const radarMsg = document.getElementById('radar-msg');
  const radarHudHdg = document.getElementById('radar-hdg');
  const radarHudRange = document.getElementById('radar-range');
  const radarHudCount = document.getElementById('radar-count');
  const radarAlert = document.getElementById('radar-alert');
  const radarZoomOut = document.getElementById('radar-zoom-out');
  const radarZoomIn = document.getElementById('radar-zoom-in');
  const radarRangeValue = document.getElementById('radar-range-value');
  const radarAlertsToggle = document.getElementById('radar-alerts-toggle');
  const radarAlertRangeDown = document.getElementById('radar-alert-range-down');
  const radarAlertRangeUp = document.getElementById('radar-alert-range-up');
  const radarAlertRangeValue = document.getElementById('radar-alert-range-value');
  const radarAlertAltDown = document.getElementById('radar-alert-alt-down');
  const radarAlertAltUp = document.getElementById('radar-alert-alt-up');
  const radarAlertAltValue = document.getElementById('radar-alert-alt-value');

  let leafletMap = null;
  let ownshipMarker = null;
  let ownshipTrack = null;

  let lastTraffic = null;
  let radarRangeNm = 5;
  const radarRangesNm = [2, 5, 10, 20, 40];
  let lastRadarVisibleCount = 0;

  let radarAlertRangeNm = 2;
  let radarAlertAltBandFeet = 1000;
  // Alerts mode cycles: off -> both -> speech -> beep -> off
  let radarAlertsMode = 'both'; // 'off' | 'both' | 'speech' | 'beep'

  const radarAlertRangesNm = [0.5, 1, 2, 3, 5, 10, 15, 20, 30, 40];
  const radarAlertAltBandsFeet = [200, 500, 1000, 2000, 5000, 10000, 20000, 30000, 40000, 50000];

  const lsKeys = {
    radarRangeNm: 'stratuxng.radar.range_nm',
    radarAlertRangeNm: 'stratuxng.radar.alert_range_nm',
    radarAlertAltBandFeet: 'stratuxng.radar.alert_alt_band_ft',
    // Back-compat: older builds stored audio_mode + alerts_enabled separately.
    radarAudioMode: 'stratuxng.radar.audio_mode',
    radarAlertsEnabled: 'stratuxng.radar.alerts_enabled',
  };

  function lsGetNumber(key, fallback) {
    try {
      const v = localStorage.getItem(key);
      if (v == null) return fallback;
      const n = Number(v);
      return Number.isFinite(n) ? n : fallback;
    } catch {
      return fallback;
    }
  }

  function lsGetString(key, fallback) {
    try {
      const v = localStorage.getItem(key);
      return (v == null || v === '') ? fallback : String(v);
    } catch {
      return fallback;
    }
  }

  function lsSet(key, value) {
    try {
      localStorage.setItem(key, String(value));
    } catch {
      // ignore
    }
  }

  function applyRadarControlStateToUI() {
    if (radarRangeValue) radarRangeValue.textContent = `${radarRangeNm} nm`;
    if (radarAlertRangeValue) radarAlertRangeValue.textContent = `${radarAlertRangeNm} nm`;
    if (radarAlertAltValue) radarAlertAltValue.textContent = `±${radarAlertAltBandFeet} ft`;

    if (radarAlertsToggle) {
      const label = (() => {
        if (radarAlertsMode === 'off') return 'Off';
        if (radarAlertsMode === 'beep') return 'Beep';
        if (radarAlertsMode === 'speech') return 'Speech';
        return 'Speech/Beep';
      })();
      radarAlertsToggle.textContent = label;
      radarAlertsToggle.setAttribute('aria-pressed', radarAlertsMode === 'off' ? 'false' : 'true');
    }
  }

  // Audio alerts (beep + speech). Browsers require a user gesture to start audio.
  const audioState = {
    armed: false,
    ctx: null,
    lastBeepAt: 0,
    lastSpokenAt: 0,
    lastSpokenWho: '',
    lastAlertKey: '',
    prompted: false,

    // Speech throttling/queueing so alerts aren't constantly interrupted.
    speechBusy: false,
    speechQueuedText: '',
    speechLockWho: '',
    speechLockUntilMs: 0,
  };

  function armAudioOnce() {
    if (audioState.armed) return;
    try {
      const AC = window.AudioContext || window.webkitAudioContext;
      if (!AC) {
        audioState.armed = true; // mark as "done"; no audio supported
        return;
      }
      audioState.ctx = new AC();
      // Some browsers start suspended; resume best-effort.
      try { audioState.ctx.resume?.(); } catch { /* ignore */ }
      audioState.armed = true;
    } catch {
      // If audio init fails, don't keep retrying.
      audioState.armed = true;
    }
  }

  function playBeepPattern() {
    const ctx = audioState.ctx;
    if (!ctx || typeof ctx.createOscillator !== 'function') return;
    const now = ctx.currentTime;
    const mkBeep = (t0) => {
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = 'sine';
      osc.frequency.value = 880;
      gain.gain.setValueAtTime(0.0001, t0);
      gain.gain.exponentialRampToValueAtTime(0.15, t0 + 0.01);
      gain.gain.exponentialRampToValueAtTime(0.0001, t0 + 0.12);
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.start(t0);
      osc.stop(t0 + 0.14);
    };
    mkBeep(now);
    mkBeep(now + 0.18);
  }

  function speakTraffic(text) {
    const msg = String(text || '').trim();
    if (!msg) return;
    if (!('speechSynthesis' in window) || typeof window.SpeechSynthesisUtterance !== 'function') return;

    const synth = window.speechSynthesis;
    // If something is already speaking, queue exactly one message (replace any existing queued one).
    if (audioState.speechBusy || synth.speaking || synth.pending) {
      audioState.speechQueuedText = msg;
      return;
    }

    try {
      const u = new window.SpeechSynthesisUtterance(msg);
      u.rate = 1.0;
      u.pitch = 1.0;
      u.volume = 1.0;
      audioState.speechBusy = true;
      const done = () => {
        audioState.speechBusy = false;
        const next = String(audioState.speechQueuedText || '').trim();
        audioState.speechQueuedText = '';
        if (next) {
          // Speak the latest queued message.
          speakTraffic(next);
        }
      };
      u.onend = done;
      u.onerror = done;
      synth.speak(u);
    } catch {
      audioState.speechBusy = false;
    }
  }

  let trafficLayer = null;
  let trafficMarkers = new Map();
  let lastOwnshipAltFeet = null;
  let selectedTrafficIcao = null;
  let followOwnship = true;
  let lastGoodLatLng = null;
  let trackPoints = [];
  let hasAutoCentered = false;
  const defaultOwnshipZoom = 12;

  function setSelectedTrafficBox(lines) {
    if (!mapTrafficBox || !mapTrafficBoxLines) return;
    const list = Array.isArray(lines) ? lines : [];
    if (!list.length) {
      mapTrafficBox.classList.remove('active');
      mapTrafficBoxLines.innerHTML = '';
      return;
    }
    mapTrafficBoxLines.innerHTML = list
      .map((s) => `<div class="map-trafficbox-line">${escapeHtml(s)}</div>`)
      .join('');
    mapTrafficBox.classList.add('active');
  }

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

    document.body.classList.toggle('map-active', key === 'map' || key === 'radar');

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
    if (key === 'radar') drawRadar();
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

  function setRadarMessage(s) {
    if (!radarMsg) return;
    radarMsg.textContent = String(s || '');
  }

  function setRadarHud() {
    const hdg = getOwnshipHeadingDeg();
    if (radarHudHdg) radarHudHdg.textContent = Number.isFinite(hdg) ? `HDG ${fmtNum(hdg, 0)}°` : 'HDG --°';
    if (radarHudRange) radarHudRange.textContent = `RNG ${radarRangeNm} nm`;

    if (radarHudCount) radarHudCount.textContent = `${lastRadarVisibleCount} targets`;
  }

  function getOwnshipHeadingDeg() {
    const trk = Number(lastGps?.track_deg);
    if (Number.isFinite(trk)) return (trk + 360) % 360;
    const h = lastAttitude?.heading_deg;
    const hh = (typeof h === 'number') ? h : (h == null ? NaN : Number(h));
    if (Number.isFinite(hh)) return (hh + 360) % 360;
    return NaN;
  }

  function radarResizeCanvasIfNeeded() {
    if (!radarCanvas) return;
    const rect = radarCanvas.getBoundingClientRect();
    const dpr = Math.max(1, Math.min(3, window.devicePixelRatio || 1));
    const wantW = Math.max(1, Math.round(rect.width * dpr));
    const wantH = Math.max(1, Math.round(rect.height * dpr));
    if (radarCanvas.width !== wantW || radarCanvas.height !== wantH) {
      radarCanvas.width = wantW;
      radarCanvas.height = wantH;
    }
  }

  function drawRadar() {
    if (!radarCanvas || !radarCtx) return;
    const radarView = document.getElementById('view-radar');
    if (!radarView?.classList?.contains('active')) return;

    radarResizeCanvasIfNeeded();

    const w = radarCanvas.width;
    const h = radarCanvas.height;
    const ctx = radarCtx;
    ctx.clearRect(0, 0, w, h);

    const bg = cssVar('--surface2') || '#111';
    const text = cssVar('--text') || '#fff';
    const muted = cssVar('--muted') || '#bbb';
    const accent = cssVar('--accent') || '#7dd3fc';
    const danger = cssVar('--danger') || accent;
    const ring = 'rgba(127,127,140,0.35)';
    const grid = 'rgba(127,127,140,0.18)';

    ctx.fillStyle = bg;
    ctx.fillRect(0, 0, w, h);

    const cx = w / 2;
    const cy = h / 2;
    const radius = Math.max(10, Math.min(w, h) * 0.46);

    // Full range rings (Garmin-style full circles).
    ctx.strokeStyle = ring;
    ctx.lineWidth = Math.max(1, Math.round(Math.min(w, h) * 0.002));
    for (let i = 1; i <= 4; i++) {
      const r = (radius * i) / 4;
      ctx.beginPath();
      ctx.arc(cx, cy, r, 0, Math.PI * 2);
      ctx.stroke();
    }

    // Crosshair lines.
    ctx.strokeStyle = grid;
    ctx.beginPath();
    ctx.moveTo(cx - radius, cy);
    ctx.lineTo(cx + radius, cy);
    ctx.moveTo(cx, cy - radius);
    ctx.lineTo(cx, cy + radius);
    ctx.stroke();

    // Ring labels (top, inside the ring).
    ctx.fillStyle = muted;
    ctx.font = `${Math.max(11, Math.round(Math.min(w, h) * 0.02))}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    for (let i = 1; i <= 4; i++) {
      const r = (radius * i) / 4;
      const nm = (radarRangeNm * i) / 4;
      ctx.fillText(`${nm}nm`, cx, cy - r + 12);
    }

    // Heading indicator at top.
    ctx.fillStyle = text;
    ctx.font = `${Math.max(12, Math.round(Math.min(w, h) * 0.024))}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
    const hdg = getOwnshipHeadingDeg();
    const hdgText = Number.isFinite(hdg) ? `HDG ${fmtNum(hdg, 0)}°` : 'HDG --°';
    ctx.fillText(hdgText, cx, cy - radius - 14);

    // Ownship symbol (fixed heading-up: triangle pointing up).
    ctx.fillStyle = accent;
    ctx.strokeStyle = 'rgba(0,0,0,0.55)';
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(cx, cy - 10);
    ctx.lineTo(cx - 7, cy + 10);
    ctx.lineTo(cx + 7, cy + 10);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    const gps = lastGps;
    const gpsOK = !!(gps && gps.enabled && gps.valid && Number.isFinite(Number(gps.lat_deg)) && Number.isFinite(Number(gps.lon_deg)));
    if (!gpsOK) {
      setRadarMessage('GPS fix required for traffic radar.');
      setRadarHud();
      if (radarAlert) radarAlert.textContent = '';
      return;
    }
    setRadarMessage('');

    const ownLat = Number(gps.lat_deg);
    const ownLon = Number(gps.lon_deg);
    const ownAlt = (gps.alt_feet == null) ? null : Number(gps.alt_feet);
    const heading = Number.isFinite(hdg) ? hdg : 0;

    const toRad = (deg) => (deg * Math.PI) / 180;
    const toDeg = (rad) => (rad * 180) / Math.PI;
    const normDeg = (deg) => ((deg % 360) + 360) % 360;
    const haversineNm = (lat1, lon1, lat2, lon2) => {
      const Rm = 6371000;
      const dLat = toRad(lat2 - lat1);
      const dLon = toRad(lon2 - lon1);
      const a = Math.sin(dLat / 2) ** 2 + Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2;
      const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
      return (Rm * c) / 1852;
    };
    const bearingDeg = (lat1, lon1, lat2, lon2) => {
      const y = Math.sin(toRad(lon2 - lon1)) * Math.cos(toRad(lat2));
      const x = Math.cos(toRad(lat1)) * Math.sin(toRad(lat2)) - Math.sin(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.cos(toRad(lon2 - lon1));
      return normDeg(toDeg(Math.atan2(y, x)));
    };

    const list = Array.isArray(lastTraffic) ? lastTraffic : [];
    let closest = null;
    let visibleCount = 0;

    const drawTrafficArrow = (x, y, rotRad, fillStyle, strokeStyle) => {
      // Match the Map marker's SVG shape:
      //   M12 2 L20 22 L12 18 L4 22 Z
      // (scaled + rotated)
      const size = Math.max(8, Math.round(Math.min(w, h) * 0.022));
      ctx.save();
      ctx.translate(x, y);
      ctx.rotate(rotRad);
      ctx.beginPath();
      ctx.moveTo(0, -size);
      ctx.lineTo(size * 0.8, size);
      ctx.lineTo(0, size * 0.6);
      ctx.lineTo(-size * 0.8, size);
      ctx.closePath();
      ctx.fillStyle = fillStyle;
      ctx.globalAlpha = 0.92;
      ctx.fill();
      ctx.globalAlpha = 1;
      ctx.strokeStyle = strokeStyle;
      ctx.lineWidth = 2;
      ctx.stroke();
      ctx.restore();
    };

    // Draw targets.
    for (const t of list) {
      const lat = Number(t?.lat_deg);
      const lon = Number(t?.lon_deg);
      if (!Number.isFinite(lat) || !Number.isFinite(lon)) continue;

      const distNm = haversineNm(ownLat, ownLon, lat, lon);
      if (!Number.isFinite(distNm) || distNm <= 0) continue;
      if (distNm > radarRangeNm) continue;
      visibleCount += 1;

      const brg = bearingDeg(ownLat, ownLon, lat, lon);
      const rel = normDeg(brg - heading);
      const ang = toRad(rel);

      const r = (distNm / radarRangeNm) * radius;
      const x = cx + Math.sin(ang) * r;
      const y = cy - Math.cos(ang) * r;

      const age = Number(t?.age_sec);
      const stale = Number.isFinite(age) && age > 15;

      // Target symbol.
      const alt = Number(t?.alt_feet);
      const altOK = Number.isFinite(alt);
      const altDelta = (altOK && Number.isFinite(ownAlt)) ? (alt - ownAlt) : null;

      const isAlertCandidate = !stale && !t?.on_ground && altDelta != null && Math.abs(altDelta) <= radarAlertAltBandFeet;
      if (isAlertCandidate) {
        if (!closest || distNm < closest.distNm) {
          closest = {
            distNm,
            altDelta,
            relDeg: rel,
            vvelFpm: Number(t?.vvel_fpm),
            tail: String(t?.tail || '').trim(),
            icao: String(t?.icao || '').trim(),
          };
        }
      }

      // Icon + direction: match Map traffic marker.
      const trk = Number(t?.track_deg);
      const rotDeg = Number.isFinite(trk) ? normDeg(trk - heading) : rel;
      const rotRad = toRad(rotDeg);
      drawTrafficArrow(x, y, rotRad, stale ? muted : danger, 'rgba(0,0,0,0.85)');

      // Relative altitude (feet) + vertical trend arrow: match Map label semantics.
      const vs = Number(t?.vvel_fpm);
      const trendArrow = Number.isFinite(vs) ? (vs > 50 ? '▲' : (vs < -50 ? '▼' : '')) : '';
      const relAltFeet = (altDelta == null) ? null : Math.round(altDelta);
      const relAltFeetLabel = (relAltFeet == null) ? null : (relAltFeet >= 0 ? `+${relAltFeet} ft` : `${relAltFeet} ft`);
      const label = (relAltFeetLabel == null) ? null : (trendArrow ? `${relAltFeetLabel} ${trendArrow}` : relAltFeetLabel);
      if (label) {
        ctx.fillStyle = stale ? muted : text;
        ctx.font = `${Math.max(11, Math.round(Math.min(w, h) * 0.022))}px system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`;
        ctx.textAlign = 'left';
        ctx.textBaseline = 'middle';
        ctx.fillText(label, x + 12, y);
      }
    }

    lastRadarVisibleCount = visibleCount;

    // Alerts (visual only): show when closest traffic is very near.
    const alertsOn = radarAlertsMode !== 'off';
    if (radarAlert) {
      if (alertsOn && closest && closest.distNm <= radarAlertRangeNm) {
        const who = closest.tail || closest.icao || 'traffic';
        const dh = (closest.altDelta == null) ? '' : ` · ΔALT ${Math.round(closest.altDelta)}ft`;
        radarAlert.textContent = `TRAFFIC ${who} · ${fmtNum(closest.distNm, 1)}nm${dh}`;
      } else {
        radarAlert.textContent = '';
      }
    }

    // Audible alerts (beep + speech), gated on user gesture.
    if (alertsOn && closest && closest.distNm <= radarAlertRangeNm) {
      if (!audioState.armed || !audioState.ctx) {
        if (!audioState.prompted) {
          setRadarMessage('Tap once to enable audio alerts.');
          audioState.prompted = true;
        }
      } else {
        const nowMs = Date.now();
        const who = closest.tail || closest.icao || 'traffic';
        // Keep this fairly stable to avoid rapid "state changes" due to jitter.
        const alertKey = `${who}|${Math.round(closest.distNm * 10)}`;

        // Beep on first detection or periodically.
        const beepCooldownMs = 2000;
        const wantBeep = (radarAlertsMode === 'beep' || radarAlertsMode === 'both');
        if (wantBeep && (audioState.lastBeepAt === 0 || nowMs - audioState.lastBeepAt >= beepCooldownMs || audioState.lastAlertKey !== alertKey)) {
          try { audioState.ctx.resume?.(); } catch { /* ignore */ }
          playBeepPattern();
          audioState.lastBeepAt = nowMs;
        }

        // Speech: on change or every ~10s while alert persists.
        const speakCooldownMs = 10000;
        const wantSpeech = (radarAlertsMode === 'speech' || radarAlertsMode === 'both');
        const lockMs = 4500;
        const lockedOther = (audioState.speechLockWho && audioState.speechLockWho !== who && nowMs < audioState.speechLockUntilMs);
        if (wantSpeech && !lockedOther && (audioState.lastSpokenAt === 0 || nowMs - audioState.lastSpokenAt >= speakCooldownMs || audioState.lastSpokenWho !== who)) {
          // Convert relative bearing to clock position.
          let clock = Math.round((Number(closest.relDeg) || 0) / 30);
          clock = ((clock % 12) + 12) % 12;
          if (clock === 0) clock = 12;

          const distNm = Number(closest.distNm);
          const distSpoken = Number.isFinite(distNm) ? `${fmtNum(distNm, 1)} nautical miles` : '';

          const altDelta = Number(closest.altDelta);
          const altPart = Number.isFinite(altDelta)
            ? (() => {
              const rounded = Math.round(Math.abs(altDelta) / 100) * 100;
              if (rounded === 0) return 'same altitude';
              return `${rounded} feet ${altDelta > 0 ? 'above' : 'below'}`;
            })()
            : '';

          const vs = Number(closest.vvelFpm);
          const trend = Number.isFinite(vs) ? (vs > 50 ? 'climbing' : (vs < -50 ? 'descending' : 'level')) : '';

          const parts = ['Traffic', `${clock} o\'clock`];
          if (distSpoken) parts.push(distSpoken);
          if (altPart) parts.push(altPart);
          if (trend) parts.push(trend);

          speakTraffic(parts.join(', '));
          audioState.lastSpokenAt = nowMs;
          audioState.lastSpokenWho = who;
          audioState.speechLockWho = who;
          audioState.speechLockUntilMs = nowMs + lockMs;
        }

        audioState.lastAlertKey = alertKey;
      }
    } else {
      audioState.lastAlertKey = '';
    }

    setRadarHud();
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
    const iconOutline = 'rgba(0,0,0,0.85)';

    const ownshipIcon = window.L.divIcon({
      className: '',
      iconSize: [26, 26],
      iconAnchor: [13, 23],
      html: `
        <div class="ownship-pin" aria-hidden="true">
          <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 23s7-7.1 7-13a7 7 0 1 0-14 0c0 5.9 7 13 7 13z" fill="${accent}" fill-opacity="0.95" stroke="${iconOutline}" stroke-width="2.0" />
            <circle cx="12" cy="10" r="3.2" fill="${cssVar('--surface') || '#111'}" fill-opacity="0.95" />
          </svg>
        </div>
      `.trim(),
    });

    ownshipMarker = window.L.marker([0, 0], { icon: ownshipIcon, interactive: false }).addTo(leafletMap);

    ownshipTrack = window.L.polyline([], {
      color: accent,
      weight: 2,
      opacity: 0.75,
    }).addTo(leafletMap);

    trafficLayer = window.L.layerGroup().addTo(leafletMap);

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

    // Tap/click on empty map clears selected traffic.
    leafletMap.on('click', () => {
      selectedTrafficIcao = null;
      setSelectedTrafficBox([]);
    });
  }

  function updateMapTraffic(traffic) {
    if (!leafletMap || !trafficLayer) return;

    const list = Array.isArray(traffic) ? traffic : [];
    const want = new Set();

    const danger = cssVar('--danger') || cssVar('--accent') || '#ffffff';
    const iconOutline = 'rgba(0,0,0,0.85)';

    const detailsByIcao = new Map();

    const toRad = (deg) => (deg * Math.PI) / 180;
    const toDeg = (rad) => (rad * 180) / Math.PI;

    const haversineNm = (lat1, lon1, lat2, lon2) => {
      const Rm = 6371000;
      const dLat = toRad(lat2 - lat1);
      const dLon = toRad(lon2 - lon1);
      const a = Math.sin(dLat / 2) ** 2 + Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2;
      const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
      return (Rm * c) / 1852;
    };

    const bearingDeg = (lat1, lon1, lat2, lon2) => {
      const y = Math.sin(toRad(lon2 - lon1)) * Math.cos(toRad(lat2));
      const x = Math.cos(toRad(lat1)) * Math.sin(toRad(lat2)) - Math.sin(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.cos(toRad(lon2 - lon1));
      let brg = toDeg(Math.atan2(y, x));
      brg = (brg + 360) % 360;
      return brg;
    };

    for (const t of list) {
      const icao = String(t?.icao || '').trim();
      if (!icao) continue;
      const lat = Number(t?.lat_deg);
      const lon = Number(t?.lon_deg);
      if (!Number.isFinite(lat) || !Number.isFinite(lon)) continue;

      const trk = Number(t?.track_deg);
      const rot = Number.isFinite(trk) ? trk : 0;
      const tail = String(t?.tail || '').trim();

      const altFeet = Number(t?.alt_feet);
      const hasAlt = Number.isFinite(altFeet);
      const relAlt = (hasAlt && Number.isFinite(lastOwnshipAltFeet)) ? Math.round(altFeet - lastOwnshipAltFeet) : null;
      const relAltStr = relAlt == null ? '' : (relAlt >= 0 ? `+${relAlt} ft` : `${relAlt} ft`);
      const relAltFeetLabel = (relAlt == null) ? null : (relAlt >= 0 ? `+${relAlt} ft` : `${relAlt} ft`);

      const vs = Number(t?.vvel_fpm);
      const trendArrow = Number.isFinite(vs) ? (vs > 50 ? '▲' : (vs < -50 ? '▼' : '')) : '';
      const vsArrow = Number.isFinite(vs) ? (vs > 0 ? '▲' : (vs < 0 ? '▼' : '')) : '';
      const vsStr = Number.isFinite(vs) && vs !== 0
        ? `${Math.abs(Math.round(vs))}fpm${vsArrow ? ` ${vsArrow}` : ''}`
        : '';

      const gs = Number(t?.ground_kt);
      const gsStr = Number.isFinite(gs) ? `${Math.round(gs)}kt` : '';

      const trkStr = Number.isFinite(trk) ? `TRK ${fmtNum(trk, 0)}°` : '';

      const ageSec = Number(t?.age_sec);
      const ageStr = Number.isFinite(ageSec) ? `${fmtNum(ageSec, 0)}s` : '';

      let distStr = '';
      let brgStr = '';
      if (lastGoodLatLng) {
        const distNm = haversineNm(lastGoodLatLng[0], lastGoodLatLng[1], lat, lon);
        const brg = bearingDeg(lastGoodLatLng[0], lastGoodLatLng[1], lat, lon);
        if (Number.isFinite(distNm)) distStr = `${fmtNum(distNm, distNm < 10 ? 1 : 0)}nm`;
        if (Number.isFinite(brg)) brgStr = `${fmtNum(brg, 0)}°`;
      }

      const labelShort = tail || icao;
      const labelLong = tail ? `${tail} (${icao})` : icao;

      // Always-visible label: show name + relative altitude.
      const altLine = (relAltFeetLabel == null)
        ? '--'
        : (trendArrow ? `${relAltFeetLabel} ${trendArrow}` : relAltFeetLabel);
      const labelHtml = `
        <div>
          <div>${escapeHtml(labelShort)}</div>
          <div>${escapeHtml(altLine)}</div>
        </div>
      `.trim();

      const infoLines = [
        labelLong,
        relAltStr ? `Rel Alt: ${relAltStr}` : 'Rel Alt: --',
        (distStr || brgStr) ? `Pos: ${[distStr, brgStr].filter(Boolean).join(' ')}` : 'Pos: --',
        gsStr ? `GS: ${gsStr}` : 'GS: --',
        trkStr ? `${trkStr}` : 'TRK: --',
        vsStr ? `VS: ${vsStr}` : 'VS: --',
        ageStr ? `Age: ${ageStr}` : 'Age: --',
      ];
      detailsByIcao.set(icao, infoLines);

      want.add(icao);
      let marker = trafficMarkers.get(icao);
      if (!marker) {
        const icon = window.L.divIcon({
          className: '',
          iconSize: [22, 22],
          iconAnchor: [11, 11],
          html: `
            <div class="traffic-arrow" style="--rot:${rot}deg" aria-hidden="true">
              <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M12 2 L20 22 L12 18 L4 22 Z" fill="${danger}" fill-opacity="0.92" stroke="${iconOutline}" stroke-width="2.0" />
              </svg>
            </div>
          `.trim(),
        });
        marker = window.L.marker([lat, lon], { icon });
        marker.bindTooltip(labelHtml, {
          permanent: true,
          direction: 'top',
          offset: [0, -14],
          opacity: 0.95,
          className: 'traffic-label',
        });

        // Mobile-first: tap selects traffic and shows the fixed info box.
        marker.on('click', (e) => {
          try { window.L.DomEvent.stopPropagation(e); } catch { /* ignore */ }
          selectedTrafficIcao = (selectedTrafficIcao === icao) ? null : icao;
          const lines = selectedTrafficIcao ? (detailsByIcao.get(selectedTrafficIcao) || []) : [];
          setSelectedTrafficBox(lines);
        });

        marker.addTo(trafficLayer);
        trafficMarkers.set(icao, marker);
      } else {
        try {
          marker.setLatLng([lat, lon]);
          const el = marker.getElement?.();
          const arrow = el ? el.querySelector?.('.traffic-arrow') : null;
          if (arrow) {
            arrow.style.setProperty('--rot', `${rot}deg`);
          }
          marker.setTooltipContent?.(labelHtml);
        } catch {
          // ignore
        }
      }
    }

    // Keep selected traffic box in sync with latest data.
    if (selectedTrafficIcao) {
      const lines = detailsByIcao.get(selectedTrafficIcao) || [];
      setSelectedTrafficBox(lines);
      if (!lines.length) {
        selectedTrafficIcao = null;
      }
    }

    // Remove markers no longer present.
    for (const [icao, marker] of trafficMarkers.entries()) {
      if (want.has(icao)) continue;
      try {
        trafficLayer.removeLayer(marker);
      } catch {
        // ignore
      }
      trafficMarkers.delete(icao);
    }

    // If the selected one disappeared this tick, hide the box.
    if (selectedTrafficIcao && !want.has(selectedTrafficIcao)) {
      selectedTrafficIcao = null;
      setSelectedTrafficBox([]);
    }
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
    lastOwnshipAltFeet = (gps && gps.alt_feet != null) ? Number(gps.alt_feet) : null;
    if (!Number.isFinite(lastOwnshipAltFeet)) lastOwnshipAltFeet = null;

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

    // Network/Disk are optional and OS-dependent.
    const netw = s?.network || null;
    const disk = s?.disk || null;

    const netErr = netw?.last_error ? ` (${netw.last_error})` : '';
    const addrs = Array.isArray(netw?.local_addrs) ? netw.local_addrs : [];
    setInput(stNetLocalAddrs, addrs.length ? addrs.join(' | ') : (netw ? `-` + netErr : ''));

    const clients = Array.isArray(netw?.clients) ? netw.clients : [];
    const cc = Number.isFinite(Number(netw?.clients_count)) ? Number(netw.clients_count) : clients.length;
    setInput(stNetClientsCount, netw ? `${cc}${netErr}` : '');

    if (stNetClients) {
      if (!netw) {
        stNetClients.value = '';
      } else if (!clients.length) {
        stNetClients.value = netErr ? netErr.trim() : '';
      } else {
        stNetClients.value = clients
          .map((c) => {
            const host = (c?.hostname || '').trim();
            const ip = (c?.ip || '').trim();
            if (host && ip) return `${host}  ${ip}`;
            return ip || host || '';
          })
          .filter(Boolean)
          .join('\n');
      }
    }

    const diskErr = disk?.last_error ? ` (${disk.last_error})` : '';
    setInput(stDiskTotal, disk ? `${fmtBytes(disk.root_total_bytes)}${diskErr}` : '');
    setInput(stDiskAvail, disk ? `${fmtBytes(disk.root_avail_bytes)}${diskErr}` : '');
    setInput(stDiskFree, disk ? `${fmtBytes(disk.root_free_bytes)}${diskErr}` : '');

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
    setInput(stFanBackend, fan.backend || '');
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

    // Traffic page.
    const traffic = Array.isArray(s?.traffic) ? s.traffic : [];
    setInput(trCount, String(traffic.length));
    if (trList) {
      trList.value = traffic
        .map((t) => {
          const id = String(t?.tail || t?.icao || '').trim();
          const alt = t?.alt_feet == null ? '--' : `${String(t.alt_feet)}ft`;
          const gs = t?.ground_kt == null ? '--' : `${String(t.ground_kt)}kt`;
          const trk = t?.track_deg == null ? '--' : `${fmtNum(t.track_deg, 0)}°`;
          const age = t?.age_sec == null ? '' : ` age ${fmtNum(t.age_sec, 1)}s`;
          const flags = `${t?.on_ground ? ' GND' : ''}${t?.extrapolated ? ' XTRP' : ''}`;
          return `${id || '--'}  ${alt} ${gs} ${trk}${age}${flags}`.trim();
        })
        .filter(Boolean)
        .join('\n');
    }

    function fmtJSON(obj) {
      try {
        return obj ? JSON.stringify(obj, null, 2) : '';
      } catch {
        return '';
      }
    }

    if (trAdsb1090) trAdsb1090.value = fmtJSON(s?.adsb1090);
    if (trUat978) trUat978.value = fmtJSON(s?.uat978);

    // Weather page (relay health, not decoded products).
    const uat = s?.uat978 || {};
    const raw = uat?.raw_stream || {};
    setInput(wxRawEndpoint, uat?.raw_endpoint || '');
    setInput(wxRawState, raw?.state || '');
    setInput(wxRawLines, raw?.lines == null ? '' : String(raw.lines));
    setInput(wxRawLastSeen, raw?.last_seen_utc || '');
    setInput(wxRawError, raw?.last_error || '');
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

  function fmtBytes(x) {
    const n = Number(x);
    if (!Number.isFinite(n) || n < 0) return '';
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
    let v = n;
    let i = 0;
    while (v >= 1024 && i < units.length - 1) {
      v /= 1024;
      i++;
    }
    const digits = i === 0 ? 0 : v >= 10 ? 1 : 2;
    return `${v.toFixed(digits)} ${units[i]}`;
  }

  function escapeHtml(s) {
    return String(s)
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#39;');
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
      lastTraffic = s?.traffic || null;
      drawAttitude();
      // Map updates are driven off the same poll.
      initMapIfNeeded();
      updateMapFromGPS(lastGps);
      updateMapTraffic(s?.traffic);
      // Radar updates (heading-up traffic view).
      drawRadar();
      if (subtitle) subtitle.textContent = 'Connected';
    } catch {
      if (subtitle) subtitle.textContent = 'Disconnected';
    }
  }

  // Initial view.
  const initial = (location.hash || '#status').slice(1);
  setView(['attitude', 'radar', 'map', 'status', 'traffic', 'weather', 'towers', 'settings', 'logs', 'about'].includes(initial) ? initial : 'status');
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
  setInterval(drawRadar, 100);
  window.addEventListener('resize', drawAttitude);
  window.addEventListener('resize', drawRadar);

  btnAhrsLevel?.addEventListener('click', () => postAhrs('/api/ahrs/level', ahrsMsg));
  btnAhrsZeroDrift?.addEventListener('click', () => postAhrs('/api/ahrs/zero-drift', ahrsMsg));
  btnAhrsOrientForward?.addEventListener('click', () => postAhrs('/api/ahrs/orient/forward', ahrsMsg));
  btnAhrsOrientDone?.addEventListener('click', () => postAhrs('/api/ahrs/orient/done', ahrsMsg));

  btnAttAhrsLevel?.addEventListener('click', () => postAhrs('/api/ahrs/level', attAhrsMsg));
  btnAttAhrsZeroDrift?.addEventListener('click', () => postAhrs('/api/ahrs/zero-drift', attAhrsMsg));

  function setRadarRange(nm) {
    const n = Number(nm);
    if (!Number.isFinite(n) || n <= 0) return;
    radarRangeNm = n;
    lsSet(lsKeys.radarRangeNm, radarRangeNm);
    applyRadarControlStateToUI();
    drawRadar();
  }

  function setRadarAlertsMode(mode) {
    const m = String(mode || '').toLowerCase();
    if (!['off', 'both', 'speech', 'beep'].includes(m)) return;
    radarAlertsMode = m;
    // Persist using the existing audio_mode key for continuity.
    lsSet(lsKeys.radarAudioMode, radarAlertsMode);
    // Back-compat: keep the old enabled flag consistent.
    lsSet(lsKeys.radarAlertsEnabled, radarAlertsMode === 'off' ? '0' : '1');
    applyRadarControlStateToUI();
    drawRadar();
  }

  function cycleRadarAlertsMode() {
    // Required cycle: off -> both -> speech -> beep -> off
    const next = (radarAlertsMode === 'off')
      ? 'both'
      : (radarAlertsMode === 'both')
        ? 'speech'
        : (radarAlertsMode === 'speech')
          ? 'beep'
          : 'off';
    setRadarAlertsMode(next);
  }

  function setRadarAlertRange(nm) {
    const n = Number(nm);
    if (!Number.isFinite(n) || n <= 0) return;
    radarAlertRangeNm = n;
    lsSet(lsKeys.radarAlertRangeNm, radarAlertRangeNm);
    applyRadarControlStateToUI();
    drawRadar();
  }

  function setRadarAlertAltBand(feet) {
    const n = Number(feet);
    if (!Number.isFinite(n) || n <= 0) return;
    radarAlertAltBandFeet = n;
    lsSet(lsKeys.radarAlertAltBandFeet, radarAlertAltBandFeet);
    applyRadarControlStateToUI();
    drawRadar();
  }

  function stepValue(list, current, dir) {
    const idx = list.indexOf(current);
    const i = idx >= 0 ? idx : 0;
    const next = i + (dir < 0 ? -1 : 1);
    const clamped = Math.max(0, Math.min(list.length - 1, next));
    return list[clamped];
  }

  radarZoomOut?.addEventListener('click', () => {
    const idx = Math.max(0, radarRangesNm.indexOf(radarRangeNm));
    setRadarRange(radarRangesNm[Math.max(0, idx - 1)]);
  });
  radarZoomIn?.addEventListener('click', () => {
    const idx = Math.max(0, radarRangesNm.indexOf(radarRangeNm));
    setRadarRange(radarRangesNm[Math.min(radarRangesNm.length - 1, idx + 1)]);
  });
  radarAlertsToggle?.addEventListener('click', cycleRadarAlertsMode);

  radarAlertRangeDown?.addEventListener('click', () => setRadarAlertRange(stepValue(radarAlertRangesNm, radarAlertRangeNm, -1)));
  radarAlertRangeUp?.addEventListener('click', () => setRadarAlertRange(stepValue(radarAlertRangesNm, radarAlertRangeNm, +1)));
  radarAlertAltDown?.addEventListener('click', () => setRadarAlertAltBand(stepValue(radarAlertAltBandsFeet, radarAlertAltBandFeet, -1)));
  radarAlertAltUp?.addEventListener('click', () => setRadarAlertAltBand(stepValue(radarAlertAltBandsFeet, radarAlertAltBandFeet, +1)));

  // Load persisted radar settings.
  radarRangeNm = lsGetNumber(lsKeys.radarRangeNm, radarRangeNm);
  if (!radarRangesNm.includes(radarRangeNm)) radarRangeNm = 5;
  radarAlertRangeNm = lsGetNumber(lsKeys.radarAlertRangeNm, radarAlertRangeNm);
  if (!radarAlertRangesNm.includes(radarAlertRangeNm)) radarAlertRangeNm = 2;
  radarAlertAltBandFeet = lsGetNumber(lsKeys.radarAlertAltBandFeet, radarAlertAltBandFeet);
  if (!radarAlertAltBandsFeet.includes(radarAlertAltBandFeet)) radarAlertAltBandFeet = 1000;
  // Default required: BOTH.
  // Back-compat: if alerts_enabled was explicitly off, honor it.
  const enabledFlag = lsGetString(lsKeys.radarAlertsEnabled, '1');
  if (enabledFlag === '0') {
    radarAlertsMode = 'off';
  } else {
    const storedMode = lsGetString(lsKeys.radarAudioMode, 'both');
    radarAlertsMode = ['off', 'both', 'speech', 'beep'].includes(storedMode) ? storedMode : 'both';
    // If storedMode is 'off' but enabledFlag isn't, still honor 'off'.
  }
  applyRadarControlStateToUI();

  // Arm audio on first user gesture.
  window.addEventListener('pointerdown', armAudioOnce, { once: true });
  window.addEventListener('keydown', armAudioOnce, { once: true });
})();
