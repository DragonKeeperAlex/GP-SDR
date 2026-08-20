const state = {
  status: null, profiles: [], events: [], signals: [], devices: [], decoders: [], mixer: [],
  integrations: null, setup: null, p25Status: null, spectrum: null, referenceResult: null,
  rangeSync: null, calibrations: [], mapper: null, remoteReceivers: [],
  selectedProfileID: null, selectedDecoderID: 'p25', editingProfile: null, activityTab: 'signals', view: 'live'
};
const serverToken = new URLSearchParams(location.search).get('token') || '';

const $ = selector => document.querySelector(selector);
const $$ = selector => [...document.querySelectorAll(selector)];
const encoder = new TextEncoder();
let toastTimer;
let setupPollTimer;
let lastWaterfallFrame = '';
const recordingPlayer = new Audio();
const liveAudio = { context:null, controller:null, gains:new Map(), nextTimes:new Map() };
let receiverApplyTimer, receiverApplying = false;
const displayPrefs = (()=>{try{return {fps:8,quality:.75,detail:512,smoothing:20,...JSON.parse(localStorage.getItem('gpsdr-display-v2')||'{}')}}catch(_){return {fps:8,quality:.75,detail:512,smoothing:20}}})();
const spectrumHistory = new WeakMap();

async function api(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (serverToken) headers['X-GP-SDR-Token'] = serverToken;
  const response = await fetch(path, { ...options, headers });
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try { message = (await response.json()).message || message; } catch (_) {}
    throw new Error(message);
  }
  if (response.status === 204) return null;
  return response.json();
}

function toast(message, error = false) {
  const element = $('#toast');
  element.textContent = message;
  element.className = `${error ? 'error ' : ''}show`;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => element.className = '', 2600);
}

function escapeHTML(value = '') {
  return String(value).replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));
}

function formatFrequency(hz) {
  if (hz >= 1e9) return `${(hz / 1e9).toFixed(6)} GHz`;
  if (hz >= 1e6) return `${(hz / 1e6).toFixed(4)} MHz`;
  if (hz >= 1e3) return `${(hz / 1e3).toFixed(3)} kHz`;
  return `${Math.round(hz)} Hz`;
}

function shortFrequency(hz) { return hz >= 1e6 ? `${(hz / 1e6).toFixed(4)}` : formatFrequency(hz); }
function timeAgo(dateString) {
  const seconds = Math.max(0, (Date.now() - new Date(dateString).getTime()) / 1000);
  if (seconds < 10) return 'now';
  if (seconds < 60) return `${Math.floor(seconds)}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}

function durationSince(dateString) {
  if (!dateString) return '—';
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(dateString).getTime()) / 1000));
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h`;
}

function setView(view) {
  state.view = view;
  $$('.nav-item').forEach(button => button.classList.toggle('active', button.dataset.view === view));
  $$('.view').forEach(element => element.classList.toggle('active', element.id === `view-${view}`));
  const copy = {
    live: ['Live', 'Receiver and channel mixer'],
    tuner: ['Tuner', 'Direct tuning, spectrum, and waterfall'],
    activity: ['Activity', 'Signals and transmission history'],
    mapper: ['Mapper', 'Wide-range activity survey'],
    profiles: ['Profiles', 'Scan ranges and channel sets'],
    decoders: ['Decoders', 'Dedicated decoder workspaces'],
    hardware: ['Hardware', 'Receivers and decoder tools'],
    settings: ['Settings', 'Server and local data']
  }[view];
  if (!copy) return;
  $('#view-title').textContent = copy[0];
  $('#view-subtitle').textContent = copy[1];
  if (view === 'tuner') { drawSpectrum(); drawWaterfall(); }
}

async function refreshAll() {
  try {
    const [status, profiles, events, signals, devices, decoders, mixer, integrations, setup, p25Status, spectrum, rangeSync, calibrations, mapper, remoteReceivers] = await Promise.all([
      api('/api/status'), api('/api/profiles'), api('/api/events?limit=300'), api('/api/signals?limit=1000'),
      api('/api/devices'), api('/api/decoders'), api('/api/mixer'), api('/api/integrations'), api('/api/setup'),
      api('/api/p25/status'), api('/api/spectrum'), api('/api/range-sync'), api('/api/calibrations'), api('/api/mapper'), api('/api/remote-receivers')
    ]);
    Object.assign(state, { status, profiles: profiles || [], events: events || [], signals: signals || [], devices: devices || [], decoders: decoders || [], mixer: mixer || [], integrations, setup, p25Status, spectrum, rangeSync, calibrations: calibrations || [], mapper, remoteReceivers: remoteReceivers || [] });
    if (!state.selectedProfileID || !profiles.some(profile => profile.id === state.selectedProfileID)) {
      state.selectedProfileID = status.activeProfileID || profiles[0]?.id || null;
    }
    render();
    clearTimeout(setupPollTimer);
    if (state.setup?.job?.state === 'installing') setupPollTimer = setTimeout(refreshAll, 1200);
  } catch (error) {
    $('#side-status').textContent = 'Offline';
    $('#side-dot').classList.remove('live');
    toast(error.message, true);
  }
}

function render() {
  renderStatus(); renderProfileSelect(); renderLatest(); renderMixer(); renderSignals();
  renderEvents(); renderProfiles(); renderHardware(); renderIntegrations(); renderRangeSync(); renderTuner(); renderDecoders(); renderMapper(); drawSpectrum(); drawWaterfall();
}

function renderMapper(){
  const body=$('#mapper-body'); if(!body)return; const connected=state.devices.filter(d=>d.connected&&d.kind!=='Simulator'), select=$('#mapper-device'); const sig=connected.map(d=>d.id).join(); if(select.dataset.signature!==sig){select.innerHTML=connected.map(d=>`<option value="${escapeHTML(d.id)}">${escapeHTML(d.name)}</option>`).join('')||'<option value="">No receiver</option>';select.dataset.signature=sig;}
  body.innerHTML=state.signals.map(s=>`<tr><td>${formatFrequency(s.frequencyHz)}</td><td>${escapeHTML(s.label||s.protocolName||'Unidentified')}</td><td>${escapeHTML(s.protocolName||s.modulation)}</td><td>${s.eventCount}</td><td>${s.strongestDBFS.toFixed(1)} dBFS</td><td>${timeAgo(s.lastSeen)}</td></tr>`).join('')||'<tr><td colspan="6">No mapped activity yet</td></tr>';
  $('#mapper-count').textContent=state.signals.length?`${state.signals.length} consolidated frequencies`:'No detected activity'; const running=state.status?.running&&state.status?.activeProfileID==='mapper-session'; $('#mapper-state').textContent=running?'Scanning':'Idle'; $('#mapper-start-button').disabled=running||!connected.length; $('#mapper-stop-button').disabled=!running;
  const m=state.mapper;if(m){if(!$('#mapper-sheet-form').contains(document.activeElement)){ $('#mapper-webhook').value=m.config.webhookURL||''; $('#mapper-secret').value=m.config.secret||''; $('#mapper-auto-upload').checked=!!m.config.autoUpload;} $('#mapper-upload-state').textContent=m.lastError?'Error':m.config.autoUpload?'Automatic':m.config.webhookURL?'Ready':'Off'; $('#mapper-upload-detail').textContent=m.lastError||`${m.uploadedRows||0} rows uploaded${m.lastUpload?' · '+timeAgo(m.lastUpload):''}`;}
}

function renderRangeSync() {
  const sync = state.rangeSync; if (!sync) return;
  const form = $('#range-sync-form');
  if (!form.contains(document.activeElement)) {
    $('#range-sync-url').value = sync.config.sheetURL || '';
    $('#range-sync-interval').value = String(sync.config.intervalMinutes || 360);
    $('#range-sync-enabled').checked = !!sync.config.enabled;
  }
  const badge = $('#range-sync-state');
  badge.textContent = sync.syncing ? 'Syncing' : sync.lastError ? 'Needs attention' : sync.lastSync ? 'Current' : sync.config.enabled ? 'Waiting' : 'Off';
  badge.className = `chip ${sync.lastError ? 'warning' : sync.lastSync && !sync.syncing ? 'ready' : ''}`;
  $('#range-sync-now').disabled = sync.syncing || !sync.config.sheetURL;
  const detail = $('#range-sync-detail');
  if (sync.syncing) detail.textContent = 'Downloading range updates…';
  else if (sync.lastError) detail.textContent = sync.lastError;
  else if (sync.lastSync) detail.textContent = `${sync.profileCount} profiles · ${sync.rangeCount} ranges · updated ${timeAgo(sync.lastSync)}${sync.usingCache ? ' · cached' : ''}`;
  else detail.textContent = sync.config.sheetURL ? 'Ready to sync.' : 'Paste a shared sheet link to begin.';
  detail.classList.toggle('error', !!sync.lastError);
}

function renderStatus() {
  const status = state.status;
  if (!status) return;
  $('#side-dot').classList.toggle('live', status.running);
  $('#side-status').textContent = status.running ? status.mode : 'Ready';
  $('#metric-devices').textContent = status.connectedDeviceCount;
  $('#metric-signals').textContent = state.signals.length;
  $('#metric-events').textContent = status.eventCount;
  $('#metric-session').textContent = durationSince(status.startedAt);
  $('#metric-mode').textContent = status.running ? status.activeProfileName || status.mode : 'idle';
	const address=$('#setting-address'); address.textContent=status.webAddress; address.href=status.webAddress;
  $('#setting-version').textContent = status.version;
  $('#demo-banner').classList.toggle('hidden', !(status.running && status.simulatorEnabled));
  const runtimeError = $('#runtime-error');
  runtimeError.textContent = status.lastError || '';
  runtimeError.classList.toggle('hidden', !status.lastError);
  const toggle = $('#survey-toggle');
  toggle.classList.toggle('running', status.running);
  toggle.querySelector('span:last-child').textContent = status.running ? 'Stop' : 'Start';
  toggle.title = status.running ? 'Stop the active survey' : 'Start the selected scan profile';
  $('#spectrum-label').textContent = status.running ? status.activeProfileName : 'Waiting for receiver';
}

function renderProfileSelect() {
  const select = $('#active-profile');
  const signature = state.profiles.map(p => p.id).join();
  if (select.dataset.signature !== signature) {
    select.innerHTML = state.profiles.map(profile => `<option value="${profile.id}">${escapeHTML(profile.name)}</option>`).join('');
    select.dataset.signature = signature;
  }
  select.value = state.status?.activeProfileID || state.selectedProfileID || '';
  select.disabled = !!state.status?.running;
}

function renderLatest() {
  const root = $('#latest-event');
  const event = state.events[0];
  if (!event) { root.className = 'empty-state compact'; root.textContent = 'No activity yet'; return; }
  root.className = 'latest';
  root.innerHTML = `
    <div class="latest-frequency">${formatFrequency(event.frequencyHz)}</div>
    <div class="latest-meta">
      <span class="chip">${escapeHTML(event.label || 'Unlabeled')}</span>
      <span class="chip">${escapeHTML(event.protocolName || event.modulation)}</span>
      <span class="chip">${event.signalDBFS.toFixed(0)} dBFS</span>
      ${event.simulated ? '<span class="chip demo">DEMO</span>' : ''}
    </div>
    <div class="latest-transcript">${escapeHTML(event.transcript || 'No transcript')}</div>
    ${event.audioPath ? `<button class="play-event" data-event-id="${event.id}" title="Play or pause this recording">▶ Recording</button>` : ''}`;
}

function renderMixer() {
  const root = $('#mixer-list');
  if (!state.mixer.length) { root.className = 'mixer-list empty-state compact'; root.textContent = 'Start a channel profile'; return; }
  root.className = 'mixer-list';
  root.innerHTML = mixerRows(state.mixer);
  applyMixerGains();
}

function mixerRows(items) {
  return items.map(item => {
    const detail = item.talkgroupID ? `TG ${item.talkgroupID} · ${item.systemName || 'P25'}${item.encrypted ? ' · encrypted' : ''}` : `${shortFrequency(item.channel.frequencyHz)} MHz`;
    return `
    <div class="mixer-row ${item.active ? 'active' : ''}" data-mixer-id="${item.id}">
      <div class="channel-name" title="${escapeHTML(item.channel.name)} · ${escapeHTML(detail)}"><strong>${escapeHTML(item.channel.name)}${item.discovered ? ' <span class="discovered-mark">new</span>' : ''}</strong><small>${escapeHTML(detail)}</small></div>
      <div class="level-meter" title="Current audio level"><i style="width:${Math.round(item.level * 100)}%"></i></div>
      <button class="mini-toggle mixer-mute ${item.muted ? 'on' : ''}" title="Mute this ${item.talkgroupID ? 'talkgroup' : 'channel'}">M</button>
      <button class="mini-toggle mixer-solo ${item.solo ? 'on' : ''}" title="Hear only this ${item.talkgroupID ? 'talkgroup' : 'channel'}">S</button>
      <input class="mixer-volume" type="range" min="0" max="1" step="0.05" value="${item.volume}" title="Channel volume" aria-label="Volume for ${escapeHTML(item.channel.name)}">
    </div>`;
  }).join('');
}

function renderSignals() {
  const filter = ($('#signal-search').value || '').toLowerCase();
  const rows = state.signals.filter(signal => [formatFrequency(signal.frequencyHz), signal.label, signal.protocolName, signal.modulation]
    .some(value => String(value || '').toLowerCase().includes(filter)));
  $('#signals-body').innerHTML = rows.length ? rows.map(signal => `
    <tr><td class="frequency">${formatFrequency(signal.frequencyHz)}</td>
    <td>${escapeHTML(signal.label || 'Unlabeled')}</td>
    <td><span class="chip">${escapeHTML(signal.protocolName || signal.modulation)}</span></td>
    <td>${signal.eventCount}</td><td>${timeAgo(signal.lastSeen)}</td>
    <td><button class="row-action share-signal" data-frequency="${signal.frequencyHz}" data-label="${escapeHTML(signal.label || '')}" data-mode="${escapeHTML(signal.modulation)}" title="Share this frequency as a profile">⇧</button></td></tr>`).join('')
    : '<tr><td colspan="6" class="empty-state compact">No discovered signals</td></tr>';
}

function renderEvents() {
  $('#events-body').innerHTML = state.events.length ? state.events.map(event => `
    <tr><td>${new Date(event.startedAt).toLocaleTimeString([], {hour:'2-digit', minute:'2-digit', second:'2-digit'})}${event.simulated ? ' · Demo' : ''}</td>
    <td class="frequency">${formatFrequency(event.frequencyHz)}</td><td>${escapeHTML(event.protocolName || event.modulation)}</td>
    <td>${event.signalDBFS.toFixed(0)} dBFS</td><td>${event.durationSeconds.toFixed(1)}s</td>
    <td class="transcript" title="${escapeHTML(event.transcript || '')}">${escapeHTML(event.transcript || '—')}</td>
    <td>${event.audioPath ? `<button class="row-action play-event" data-event-id="${event.id}" title="Play or pause this recording">▶</button>` : ''}</td></tr>`).join('')
    : '<tr><td colspan="7" class="empty-state compact">No events logged</td></tr>';
}

function renderProfiles() {
  $('#profile-grid').innerHTML = state.profiles.map(profile => `
    <article class="profile-card ${profile.id === state.selectedProfileID ? 'selected' : ''}" data-profile-id="${profile.id}">
      <div class="card-top"><div><h3>${escapeHTML(profile.name)}</h3><p>${escapeHTML(profile.summary || 'Custom scan configuration')}</p></div>${profile.builtIn ? '<span class="chip">Built-in</span>' : ''}</div>
      <div class="profile-stats"><div><span>Ranges</span><strong>${profile.ranges.length}</strong></div><div><span>Channels</span><strong>${profile.channels.length}</strong></div><div><span>P25</span><strong>${(profile.p25Systems || []).length}</strong></div><div><span>Receivers</span><strong>${profile.deviceAssignments.length}</strong></div></div>
      <div class="card-actions">
        <button class="select-profile" title="Use this profile">Use</button>
        <button class="${profile.builtIn ? 'duplicate-profile' : 'edit-profile'}" title="${profile.builtIn ? 'Make an editable copy' : 'Edit this profile'}">${profile.builtIn ? 'Duplicate' : 'Edit'}</button>
        <button class="export-profile" title="Download this profile for sharing">Export</button>
      </div>
    </article>`).join('');
}

function renderHardware() {
  $('#device-grid').innerHTML = state.devices.map(device => `
    <article class="hardware-card"><div class="hardware-title"><i class="${device.connected ? 'ready' : device.available ? 'optional' : ''}"></i><h3>${escapeHTML(device.name)}</h3></div>
		<p>${escapeHTML(device.connected && state.status?.running ? `Streaming · ${state.status.mode}` : device.note || 'Connected and ready for assignment.')}</p>
      <div class="hardware-detail">${device.kind === 'HackRF' ? 'LNA 0–40 dB · VGA 0–62 dB · RF amp · antenna power · 2–20 MS/s' : device.kind === 'RTL-SDR' ? 'Tuner AGC/manual gain · PPM correction · 0.225–3.2 MS/s' : 'SoapySDR gain · PPM and device-specific controls'}<br>${escapeHTML(device.driver)}${device.serial ? ` · ${escapeHTML(device.serial)}` : ''}${device.helperArchitecture ? ` · ${escapeHTML(device.helperArchitecture)}` : ''}</div>
      <footer><span>${escapeHTML(device.kind)}</span><span>${device.connected ? 'Connected' : device.available ? 'Driver ready' : 'Driver needed'}</span></footer>
      ${device.connected ? calibrationControls(device) : ''}
      ${device.connected || device.available ? '' : setupActions(device.kind === 'HackRF' ? 'hackrf' : device.kind === 'RTL-SDR' ? 'rtlsdr' : 'soapysdr')}</article>`).join('');
  $('#decoder-grid').innerHTML = state.decoders.map(decoder => `
    <article class="hardware-card"><div class="hardware-title"><i class="${decoder.state}"></i><h3>${escapeHTML(decoder.name)}</h3></div>
      <p>${escapeHTML(decoder.note)}</p><footer><span>${escapeHTML(decoder.standards.join(' · '))}</span><span>${decoder.state === 'ready' ? 'Ready' : 'Optional'}</span></footer>
      <div class="card-actions decoder-card-actions"><button class="open-decoder" data-decoder-id="${escapeHTML(decoder.id)}" title="Open the ${escapeHTML(decoder.name)} workspace">Open</button></div>
      ${decoder.state === 'ready' || decoder.id === 'analog' ? '' : setupActions(decoder.id)}</article>`).join('');
  renderSetupJob();
  const remoteList=$('#remote-list'); if(remoteList) remoteList.innerHTML=state.remoteReceivers.map(item=>`<div class="remote-row"><span><strong>${escapeHTML(item.name)}</strong><small>${escapeHTML(item.host)}:${item.port}</small></span><button class="remove-remote" data-remote-id="${escapeHTML(item.id)}" title="Remove this remote receiver">Remove</button></div>`).join('')||'<span class="empty-state compact">No remote receivers saved</span>';
}

function calibrationControls(device) {
  const c = device.calibration;
  const reference = c?.referenceHz ? c.referenceHz / 1e6 : state.spectrum?.centerFrequencyHz ? state.spectrum.centerFrequencyHz / 1e6 : 98.1;
  const status = c ? `${c.ppmCorrection >= 0 ? '+' : ''}${c.ppmCorrection} PPM · Q ${c.iqGain.toFixed(3)} · ${c.iqPhase >= 0 ? '+' : ''}${c.iqPhase.toFixed(1)}° · ${Math.round(c.confidence*100)}%` : 'Not calibrated';
  return `<div class="calibration-box" data-calibration-device="${escapeHTML(device.id)}">
    <div><strong>Calibration</strong><span>${escapeHTML(status)}</span></div>
    <label>Reference · MHz<input class="calibration-reference" type="number" min="0.001" step="0.000001" value="${reference}" title="Known active carrier used to measure frequency and I/Q error"></label>
    <div class="calibration-actions"><button class="auto-calibration" title="Measure, save, and apply calibration for this receiver">Auto calibrate</button><button class="edit-calibration" title="Save the displayed tuner PPM and I/Q controls for this receiver">Save current</button>${c ? '<button class="reset-calibration" title="Remove saved calibration">Reset</button>' : ''}</div>
  </div>`;
}

function renderTuner() {
  const select = $('#tuner-device');
	const liveSelect = $('#live-radio-device');
  if (!select) return;
  const connected = state.devices.filter(device => device.connected && device.kind !== 'Simulator');
  const signature = connected.map(device => device.id).join();
  if (select.dataset.signature !== signature) {
    select.innerHTML = connected.length ? connected.map(device => `<option value="${escapeHTML(device.id)}">${escapeHTML(device.name)}</option>`).join('') : '<option value="">No receiver</option>';
		if (liveSelect) liveSelect.innerHTML = select.innerHTML;
    select.dataset.signature = signature;
  }
  const tuning = state.status?.running && state.status?.activeProfileID === 'quick-tune';
	const receiving = state.mixer.some(item => item.active);
	for (const prefix of ['live','tuner']) { const light=$('#'+prefix+'-signal-light')?.parentElement; if(light) light.classList.toggle('active',receiving); const label=$('#'+prefix+'-signal-text'); if(label) label.textContent=receiving?'Signal':'No signal'; }
  $('#tuner-start').disabled = !connected.length || tuning;
  $('#tuner-stop').disabled = !tuning;
  $('#tuner-status').textContent = tuning ? `${state.status.mode} · ${state.status.activeProfileName}` : connected.length ? 'Ready to tune.' : 'Connect a receiver, then refresh Hardware.';
  const snapshot = state.spectrum;
  if (snapshot?.binsDBFS?.length) {
    $('#tuner-center').textContent = formatFrequency(snapshot.centerFrequencyHz);
    $('#tuner-span').textContent = `${(snapshot.sampleRateHz / 1e6).toFixed(2)} MS/s · ${snapshot.binsDBFS.length} bins`;
    $('#waterfall-start').textContent = formatFrequency(snapshot.startFrequencyHz);
    $('#waterfall-mid').textContent = formatFrequency(snapshot.centerFrequencyHz);
    $('#waterfall-end').textContent = formatFrequency(snapshot.endFrequencyHz);
		$('#live-waterfall-start').textContent = formatFrequency(snapshot.startFrequencyHz);
		$('#live-waterfall-mid').textContent = formatFrequency(snapshot.centerFrequencyHz);
		$('#live-waterfall-end').textContent = formatFrequency(snapshot.endFrequencyHz);
  }
}

function tunerRequest() {
  return {deviceID:$('#tuner-device').value,frequencyHz:Number($('#tuner-frequency').value)*1e6,mode:$('#tuner-mode').value,
    bandwidthHz:Number($('#tuner-bandwidth').value)*1000,sampleRateHz:Number($('#tuner-rate').value),gainDB:Number($('#tuner-gain').value),
    lnaGainDB:Number($('#tuner-lna').value),vgaGainDB:Number($('#tuner-vga').value),ppmCorrection:Number($('#tuner-ppm').value),
    ampEnabled:$('#tuner-amp').checked,antennaPower:$('#tuner-bias').checked,iqDCRemoval:$('#tuner-dc').checked,
    iqGain:Number($('#tuner-iq-gain').value),iqPhase:Number($('#tuner-iq-phase').value),iqSwap:$('#tuner-iq-swap').checked,
    autoGain:$('#tuner-agc').checked,squelchDB:Number($('#tuner-squelch').value),monitorOpen:$('#tuner-monitor-open').checked,
    noiseReduction:$('#tuner-noise').value,useCalibration:$('#tuner-use-calibration').checked};
}

function decoderMatchesEvent(decoder, event) {
  const identity = `${event.protocolName || ''} ${event.modulation || ''}`.toLowerCase();
  if (decoder.id === 'analog') return !event.protocolName && ['am','nfm','wfm','fm'].includes(String(event.modulation || '').toLowerCase());
  if (decoder.id === 'p25') return identity.includes('p25');
  const keys = {
    'dsd-fme':['dmr','nxdn','d-star','ysf','m17'], 'rtl-433':['sensor','tpms','weather'], dump1090:['ads-b','mode s'],
    'multimon-ng':['pocsag','flex','mdc1200','dtmf'], acarsdec:['acars'], ais:['ais']
  }[decoder.id] || decoder.standards.map(value => value.toLowerCase());
  return keys.some(key => identity.includes(key));
}

function renderDecoders() {
  const nav = $('#decoder-nav'), detail = $('#decoder-detail');
  if (!nav || !detail) return;
  if (!state.decoders.some(item => item.id === state.selectedDecoderID)) state.selectedDecoderID = state.decoders[0]?.id || 'analog';
  nav.innerHTML = state.decoders.map(decoder => `<button class="decoder-nav-item ${decoder.id === state.selectedDecoderID ? 'active' : ''}" data-decoder-id="${escapeHTML(decoder.id)}">
    <i class="${escapeHTML(decoder.state)}"></i><span><strong>${escapeHTML(decoder.name)}</strong><small>${escapeHTML(decoder.standards.slice(0,2).join(' · '))}</small></span></button>`).join('');
  const decoder = state.decoders.find(item => item.id === state.selectedDecoderID);
  if (!decoder) { detail.innerHTML = '<div class="empty-state">No decoders found</div>'; return; }
  const events = state.events.filter(event => decoderMatchesEvent(decoder,event)).slice(0,12);
  const relevantProfiles = state.profiles.filter(profile => decoder.id === 'p25' ? (profile.p25Systems || []).length :
    profile.channels.some(channel => decoder.id === 'analog' ? ['am','nfm','wfm','fm','auto'].includes(channel.mode) : channel.decoder === decoder.id));
  const setup = decoder.state === 'ready' || decoder.id === 'analog' ? '' : setupActions(decoder.id);
  const p25 = decoder.id === 'p25' ? renderP25DecoderWorkspace() : '';
  detail.innerHTML = `<article class="panel decoder-hero">
    <div><div class="decoder-heading"><i class="${escapeHTML(decoder.state)}"></i><h2>${escapeHTML(decoder.name)}</h2><span class="chip">${decoder.state === 'ready' ? 'Ready' : 'Setup'}</span></div>
    <p>${escapeHTML(decoder.note)}</p><div class="decoder-standards">${decoder.standards.map(item=>`<span>${escapeHTML(item)}</span>`).join('')}</div></div>${setup}</article>
    ${p25}
    <div class="decoder-columns">
      <article class="panel"><div class="panel-head"><div><h2>Profiles</h2><span>Configurations using this decoder</span></div></div><div class="decoder-profile-list">${relevantProfiles.length ? relevantProfiles.map(profile=>`<button class="decoder-profile" data-profile-id="${escapeHTML(profile.id)}"><span><strong>${escapeHTML(profile.name)}</strong><small>${escapeHTML(profile.summary || '')}</small></span><b>Use</b></button>`).join('') : '<div class="empty-state compact">No matching profiles</div>'}</div></article>
      <article class="panel"><div class="panel-head"><div><h2>Recent activity</h2><span>Events identified by this decoder</span></div></div><div class="decoder-event-list">${events.length ? events.map(event=>`<div><span><strong>${escapeHTML(event.label || formatFrequency(event.frequencyHz))}</strong><small>${formatFrequency(event.frequencyHz)} · ${timeAgo(event.startedAt)}</small></span><b>${escapeHTML(event.protocolName || event.modulation)}</b></div>`).join('') : '<div class="empty-state compact">No decoded activity yet</div>'}</div></article>
    </div>`;
}

function renderP25DecoderWorkspace() {
  const status = state.p25Status || {};
  const talkgroups = state.mixer.filter(item => item.talkgroupID);
  const active = talkgroups.filter(item => item.active).length;
  const connected = state.devices.filter(item=>item.connected);
  const calibrated = connected.filter(item=>item.calibration).length;
  return `<article class="panel p25-overview"><div class="p25-metrics"><div><span>Engine</span><strong>${escapeHTML(status.engine || 'Bundled')}</strong></div><div><span>Reception</span><strong>${escapeHTML(status.reception || status.state || 'setup')}</strong></div><div><span>Talkgroups</span><strong>${talkgroups.length}</strong></div><div><span>Calibration</span><strong>${calibrated}/${connected.length}</strong></div></div><p class="hardware-detail">${escapeHTML(status.note || '')}${calibrated ? ' · Saved PPM, gain, and front-end calibration applied; P25 IQ tracking remains automatic.' : ' · Calibrate the receiver on the Hardware page for best results.'}</p>
    <div class="panel-head"><div><h2>Talkgroup mixer</h2><span>Mute, solo, and set volume independently</span></div><button class="decoder-mute-all icon-button" title="Mute or unmute every P25 talkgroup">M</button></div>
    <div class="mixer-list p25-mixer">${talkgroups.length ? mixerRows(talkgroups) : '<div class="empty-state compact">Start a P25 profile to load talkgroups</div>'}</div></article>`;
}

function renderIntegrations() {
  if (!state.integrations) return;
  const names = { dsp:'Live DSP', soapySDR:'SoapySDR', p25:'P25 trunking', transcription:'Transcription', radioReference:'RadioReference' };
  $('#integration-grid').innerHTML = Object.entries(names).map(([key, name]) => {
    const item = state.integrations[key] || {};
    const ready = item.state === 'ready' || item.state === 'running';
    const setupIDs = {soapySDR:'soapysdr',p25:'p25',transcription:'transcription',radioReference:'radioreference'};
    return `<article class="hardware-card"><div class="hardware-title"><i class="${ready ? 'ready' : item.state === 'error' ? '' : 'optional'}"></i><h3>${name}</h3></div>
      <p>${escapeHTML(item.note || 'Not configured.')}</p><footer><span class="integration-state">${escapeHTML(item.state || 'setup')}</span><span>${ready ? 'Ready' : 'Optional'}</span></footer>
      ${ready || !setupIDs[key] ? '' : setupActions(setupIDs[key])}</article>`;
  }).join('');
  renderSetupJob();
}

function setupComponent(id) { return state.setup?.components?.find(component => component.id === id); }

function setupActions(id) {
  const component = setupComponent(id); if (!component || component.state === 'ready') return '';
  const busy = state.setup?.job?.state === 'installing';
  return `<div class="setup-actions">
    ${component.installable ? `<button class="primary install-component" data-component-id="${escapeHTML(id)}" ${busy ? 'disabled' : ''} title="Install ${escapeHTML(component.name)} with the detected package manager">${busy && state.setup.job.componentID === id ? 'Installing…' : 'Install'}</button>` : ''}
    <button class="guide-component" data-component-id="${escapeHTML(id)}" title="Show setup instructions">How to</button>
  </div>`;
}

function renderSetupJob() {
  const root = $('#setup-job'); if (!root) return;
  const job = state.setup?.job;
  root.classList.toggle('hidden', !job);
  if (!job) return;
  root.className = `setup-job ${job.state === 'error' ? 'error' : ''}`;
  root.innerHTML = `<strong>${escapeHTML(setupComponent(job.componentID)?.name || job.componentID)}</strong><span>${escapeHTML(job.message)}</span>`;
}

function openSetupGuide(id) {
  const component = setupComponent(id); if (!component) return;
  $('#setup-dialog-title').textContent = component.name;
  $('#setup-guide').textContent = component.guide;
  const command = $('#setup-command'); command.textContent = component.command || ''; command.classList.toggle('hidden', !component.command);
  const link = $('#setup-link'); link.href = component.guideURL || '#'; link.classList.toggle('hidden', !component.guideURL);
  $('#setup-dialog').showModal();
}

function applyMixerGains() {
  const solo = state.mixer.some(item => item.solo);
  for (const item of state.mixer) {
    const gain = liveAudio.gains.get(item.id);
    if (!gain) continue;
    gain.gain.setTargetAtTime(item.muted || (solo && !item.solo) ? 0 : item.volume, liveAudio.context.currentTime, .015);
  }
}

function channelGain(channelID) {
  let gain = liveAudio.gains.get(channelID);
  if (!gain) {
    gain = liveAudio.context.createGain(); gain.connect(liveAudio.context.destination);
    liveAudio.gains.set(channelID, gain); applyMixerGains();
  }
  return gain;
}

function scheduleAudioFrame(channelID, sampleRate, pcm) {
  if (!liveAudio.context || !pcm.length || sampleRate < 8000) return;
  const buffer = liveAudio.context.createBuffer(1, pcm.length, sampleRate);
  const output = buffer.getChannelData(0);
  for (let index=0; index<pcm.length; index++) output[index] = pcm[index] / 32768;
  const source = liveAudio.context.createBufferSource(); source.buffer = buffer; source.connect(channelGain(channelID));
  const now = liveAudio.context.currentTime, previous = liveAudio.nextTimes.get(channelID) || now;
	const start = previous < now - .02 || previous > now + .12 ? now + .015 : previous;
  source.start(start); liveAudio.nextTimes.set(channelID, start + buffer.duration);
}

async function startLiveAudio() {
  const AudioContextClass = window.AudioContext || window.webkitAudioContext;
  if (!AudioContextClass) return;
  if (!liveAudio.context) liveAudio.context = new AudioContextClass({latencyHint:'interactive'});
  await liveAudio.context.resume();
	const audioState=$('#audio-state'); if(audioState) audioState.textContent=liveAudio.context.state==='running'?'Audio ready':'Audio blocked';
  if (liveAudio.controller) return;
  const controller = new AbortController(); liveAudio.controller = controller;
  const headers = serverToken ? {'X-GP-SDR-Token':serverToken} : {};
  try {
    const response = await fetch('/api/live-audio', {headers, signal:controller.signal});
    if (!response.ok || !response.body) throw new Error('Live audio stream is unavailable');
    const reader = response.body.getReader(); let pending = new Uint8Array(0);
    while (true) {
      const {value,done} = await reader.read(); if (done) break;
      const joined = new Uint8Array(pending.length + value.length); joined.set(pending); joined.set(value,pending.length); pending=joined;
      while (pending.length >= 10) {
        const header = new DataView(pending.buffer,pending.byteOffset,pending.byteLength);
        const idLength=header.getUint16(0,true), sampleRate=header.getUint32(2,true), count=header.getUint32(6,true);
        const packetLength=10+idLength+count*2; if(count>10000000) throw new Error('Invalid live audio frame'); if(pending.length<packetLength) break;
        const channelID=new TextDecoder().decode(pending.slice(10,10+idLength)); const pcm=new Int16Array(count);
        const samples=new DataView(pending.buffer,pending.byteOffset+10+idLength,count*2);
        for(let index=0;index<count;index++) pcm[index]=samples.getInt16(index*2,true);
        scheduleAudioFrame(channelID,sampleRate,pcm); pending=pending.slice(packetLength);
      }
    }
  } catch (error) { if (error.name !== 'AbortError') toast(error.message,true); }
  finally { if (liveAudio.controller === controller) liveAudio.controller = null; }
}

function stopLiveAudio() {
  liveAudio.controller?.abort(); liveAudio.controller=null; liveAudio.nextTimes.clear();
}

function drawSpectrumCanvas(canvas) {
  if (!canvas || !canvas.isConnected || canvas.offsetParent === null) return;
  const ratio = window.devicePixelRatio || 1;
  const width = Math.max(600, canvas.clientWidth);
  const height = canvas.clientHeight;
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return;
  if (canvas.width !== width * ratio || canvas.height !== height * ratio) {
    canvas.width = width * ratio; canvas.height = height * ratio;
  }
  const ctx = canvas.getContext('2d'); ctx.setTransform(ratio,0,0,ratio,0,0); ctx.clearRect(0,0,width,height);
  ctx.strokeStyle = '#1b222b'; ctx.lineWidth = 1;
  for (let x = 0; x <= width; x += width / 10) { ctx.beginPath(); ctx.moveTo(x,0); ctx.lineTo(x,height); ctx.stroke(); }
  const gridStepY = height / 5;
  for (let y = 0; y <= height; y += gridStepY) { ctx.beginPath(); ctx.moveTo(0,y); ctx.lineTo(width,y); ctx.stroke(); }
	const sourceBins = state.spectrum?.binsDBFS || [];
	const desired=Math.min(displayPrefs.detail,sourceBins.length), stride=Math.max(1,Math.floor(sourceBins.length/Math.max(1,desired)));
	let bins=sourceBins.filter((_,index)=>index%stride===0).slice(0,desired);
	const old=spectrumHistory.get(canvas); if(old?.length===bins.length && displayPrefs.smoothing>0){const weight=displayPrefs.smoothing/100;bins=bins.map((value,index)=>value*(1-weight)+old[index]*weight);} spectrumHistory.set(canvas,bins);
	const points = bins.length || 180;
  ctx.beginPath();
  for (let i = 0; i < points; i++) {
    const x = i / (points - 1) * width;
    const db = bins.length ? bins[i] : -110;
    const normalized = Math.max(0, Math.min(1, (db + 120) / 100));
    const y = height - 8 - normalized * (height - 18);
    i ? ctx.lineTo(x,y) : ctx.moveTo(x,y);
  }
  const gradient = ctx.createLinearGradient(0,0,width,0); gradient.addColorStop(0,'#3f9e86'); gradient.addColorStop(.5,'#55e0b6'); gradient.addColorStop(1,'#5a98df');
  ctx.strokeStyle = state.status?.running ? gradient : '#39434f'; ctx.lineWidth = 1.4; ctx.stroke();
  ctx.lineTo(width,height); ctx.lineTo(0,height); ctx.closePath();
  const fill = ctx.createLinearGradient(0,0,0,height); fill.addColorStop(0,'rgba(73,213,170,.2)'); fill.addColorStop(1,'rgba(73,213,170,0)'); ctx.fillStyle = fill; ctx.fill();
}

function drawSpectrum() {
  drawSpectrumCanvas($('#spectrum'));
  drawSpectrumCanvas($('#tuner-spectrum'));
}

function waterfallColor(db) {
  const value = Math.max(0, Math.min(1, (db + 115) / 95));
  if (value < .25) return [3, 10 + value * 80, 24 + value * 100];
  if (value < .55) return [15, 45 + value * 210, 105 + value * 150];
  if (value < .8) return [45 + value * 190, 210, 150 - value * 70];
  return [255, 232 - value * 80, 120 - value * 90];
}

function drawWaterfall() {
	for(const canvas of [$('#live-waterfall'),$('#waterfall')]) drawWaterfallCanvas(canvas);
}

function drawWaterfallCanvas(canvas) {
	const snapshot = state.spectrum;
	if (!canvas || canvas.offsetParent === null || !snapshot?.binsDBFS?.length || snapshot.capturedAt === canvas.dataset.lastFrame) return;
	const ratio = (window.devicePixelRatio || 1)*displayPrefs.quality, width = Math.max(450, Math.floor(canvas.clientWidth * ratio)), height = Math.max(1, Math.floor(canvas.clientHeight * ratio));
	if (canvas.width !== width || canvas.height !== height) { canvas.width = width; canvas.height = height; canvas.dataset.lastFrame = ''; }
  const ctx = canvas.getContext('2d');
	const rowHeight=Math.max(2,Math.ceil(ratio*2));
	ctx.drawImage(canvas, 0, 0, width, height - rowHeight, 0, rowHeight, width, height - rowHeight);
	const image = ctx.createImageData(width, rowHeight);
  for (let x=0; x<width; x++) {
    const index = Math.min(snapshot.binsDBFS.length-1, Math.floor(x / width * snapshot.binsDBFS.length));
    const [red,green,blue] = waterfallColor(snapshot.binsDBFS[index]);
    for (let row=0; row<image.height; row++) { const offset=(row*width+x)*4; image.data[offset]=red; image.data[offset+1]=green; image.data[offset+2]=blue; image.data[offset+3]=255; }
  }
	ctx.putImageData(image,0,0); canvas.dataset.lastFrame = snapshot.capturedAt;
}

function emptyProfile() {
  return { schemaVersion: 1, id: crypto.randomUUID(), name: '', summary: '', ranges: [], channels: [],
    deviceAssignments: [{id: crypto.randomUUID(), deviceID: null, role: 'automatic', target: null}],
    p25Systems: [], settings: {noiseMarginDB:8,revisitSeconds:20,recordAudio:true,recordIQForUnknown:true,transcribeVoice:false,maxRecordingDays:30}, builtIn:false };
}

function openProfileEditor(profile = emptyProfile()) {
  state.editingProfile = structuredClone(profile);
  $('#profile-dialog-title').textContent = profile.name ? 'Edit profile' : 'New profile';
  $('#profile-name').value = profile.name; $('#profile-summary').value = profile.summary || '';
  state.editingProfile.p25Systems ||= [];
  $('#record-audio').checked = profile.settings?.recordAudio !== false;
  $('#transcribe-voice').checked = !!profile.settings?.transcribeVoice;
  renderEditorRows(); $('#profile-dialog').showModal();
}

function renderEditorRows() {
  const profile = state.editingProfile;
  $('#ranges-editor').innerHTML = profile.ranges.length ? profile.ranges.map(range => `
    <div class="editor-row" data-id="${range.id}"><input data-key="name" value="${escapeHTML(range.name)}" placeholder="Name" title="Range name">
      <input data-key="startHz" type="number" value="${range.startHz / 1e6}" step="0.0001" placeholder="Start MHz" title="Start frequency in MHz">
      <input data-key="endHz" type="number" value="${range.endHz / 1e6}" step="0.0001" placeholder="End MHz" title="End frequency in MHz">
      <input data-key="stepHz" type="number" value="${range.stepHz / 1000}" step="0.001" placeholder="Step kHz" title="Channel step in kHz">
      <select data-key="preferredMode" title="Preferred modulation"><option value="auto">Auto</option><option value="am">AM</option><option value="nfm">NFM</option><option value="wfm">WFM</option><option value="digital">Digital</option></select>
      <button type="button" class="remove-row" title="Remove range">×</button></div>`).join('') : '<div class="editor-empty">No sweep ranges</div>';
  profile.ranges.forEach(range => { const row = $(`#ranges-editor [data-id="${range.id}"]`); if (row) row.querySelector('[data-key="preferredMode"]').value = range.preferredMode; });
  $('#channels-editor').innerHTML = profile.channels.length ? profile.channels.map(channel => `
    <div class="editor-row channel" data-id="${channel.id}"><input data-key="name" value="${escapeHTML(channel.name)}" placeholder="Name" title="Channel name">
      <input data-key="frequencyHz" type="number" value="${channel.frequencyHz / 1e6}" step="0.0001" placeholder="MHz" title="Frequency in MHz">
      <select data-key="mode" title="Modulation"><option value="auto">Auto</option><option value="am">AM</option><option value="nfm">NFM</option><option value="wfm">WFM</option><option value="digital">Digital</option><option value="p25">P25</option></select>
      <input data-key="decoder" value="${escapeHTML(channel.decoder || '')}" placeholder="Decoder" title="Optional decoder name">
      <button type="button" class="remove-row" title="Remove channel">×</button></div>`).join('') : '<div class="editor-empty">No fixed channels</div>';
  profile.channels.forEach(channel => { const row = $(`#channels-editor [data-id="${channel.id}"]`); if (row) row.querySelector('[data-key="mode"]').value = channel.mode; });
  $('#assignments-editor').innerHTML = profile.deviceAssignments.map(item => `
    <div class="editor-row assignment" data-id="${item.id}"><select data-key="role" title="Receiver role"><option value="automatic">Automatic</option><option value="discovery">Discovery sweep</option><option value="control">Trunk control</option><option value="voice">Trunk voice</option><option value="channelBank">Wideband bank</option><option value="parked">Parked channel</option></select>
      <select data-key="deviceID" title="Pin this role to a receiver"><option value="">Any compatible receiver</option>${state.devices.filter(d=>d.available).map(d=>`<option value="${d.id}">${escapeHTML(d.name)}</option>`).join('')}</select>
      <button type="button" class="remove-row" title="Remove receiver role">×</button></div>`).join('');
  profile.deviceAssignments.forEach(item => { const row = $(`#assignments-editor [data-id="${item.id}"]`); if (row) { row.querySelector('[data-key="role"]').value = item.role; row.querySelector('[data-key="deviceID"]').value = item.deviceID || ''; } });
  $('#p25-editor').innerHTML = profile.p25Systems.length ? profile.p25Systems.map(system => `
    <div class="editor-row p25" data-id="${system.id}">
      <input data-key="name" value="${escapeHTML(system.name)}" placeholder="System name" title="P25 system name">
      <input data-key="controlChannelsHz" value="${system.controlChannelsHz.map(frequency => (frequency/1e6).toFixed(6)).join(', ')}" placeholder="Control MHz" title="Comma-separated control channels in MHz">
      <input data-key="nac" value="${escapeHTML(system.nac || '')}" placeholder="NAC" title="Network access code, such as 0x123">
      <input data-key="wacn" value="${escapeHTML(system.wacn || '')}" placeholder="WACN" title="Wide area communications network ID">
      <input data-key="systemID" value="${escapeHTML(system.systemID || '')}" placeholder="SysID" title="P25 system ID">
      <input data-key="talkgroups" value="${escapeHTML((system.talkgroups || []).map(talkgroup=>`${talkgroup.id}:${talkgroup.name}:${talkgroup.mode || (talkgroup.encrypted ? 'DE' : 'D')}`).join(', '))}" placeholder="TGID:Name:Mode" title="Optional talkgroups, such as 101:Dispatch:D, 202:Tactical:DE; E marks encrypted">
      <label class="compact-check" title="Control channel carries Phase 2 TDMA signaling"><input data-key="tdmaControl" type="checkbox" ${system.tdmaControl ? 'checked' : ''}> TDMA CC</label>
      <button type="button" class="remove-row" title="Remove P25 system">×</button>
    </div>`).join('') : '<div class="editor-empty">No P25 systems</div>';
}

function collectEditor() {
  const profile = state.editingProfile;
  profile.name = $('#profile-name').value.trim(); profile.summary = $('#profile-summary').value.trim();
  profile.ranges = $$('#ranges-editor .editor-row').map(row => ({
    id: row.dataset.id, name: row.querySelector('[data-key="name"]').value.trim() || 'Range',
    startHz: Number(row.querySelector('[data-key="startHz"]').value) * 1e6,
    endHz: Number(row.querySelector('[data-key="endHz"]').value) * 1e6,
    stepHz: Number(row.querySelector('[data-key="stepHz"]').value) * 1e3,
    dwellMilliseconds: 180, preferredMode: row.querySelector('[data-key="preferredMode"]').value, enabled: true
  }));
  profile.channels = $$('#channels-editor .editor-row').map(row => ({
    id: row.dataset.id, name: row.querySelector('[data-key="name"]').value.trim() || 'Channel',
    frequencyHz: Number(row.querySelector('[data-key="frequencyHz"]').value) * 1e6,
    bandwidthHz: row.querySelector('[data-key="mode"]').value === 'wfm' ? 180000 : 12500,
    mode: row.querySelector('[data-key="mode"]').value,
    decoder: row.querySelector('[data-key="decoder"]').value.trim() || null, enabled: true, priority: 5
  }));
  profile.deviceAssignments = $$('#assignments-editor .editor-row').map(row => ({
    id: row.dataset.id, role: row.querySelector('[data-key="role"]').value,
    deviceID: row.querySelector('[data-key="deviceID"]').value || null, target: null
  }));
  profile.p25Systems = $$('#p25-editor .editor-row').map(row => ({
    id: row.dataset.id, name: row.querySelector('[data-key="name"]').value.trim() || 'P25 system',
    controlChannelsHz: row.querySelector('[data-key="controlChannelsHz"]').value.split(',').map(value=>Number(value.trim())*1e6).filter(value=>Number.isFinite(value)&&value>0),
    nac: row.querySelector('[data-key="nac"]').value.trim(), wacn: row.querySelector('[data-key="wacn"]').value.trim(),
    systemID: row.querySelector('[data-key="systemID"]').value.trim(), tdmaControl:row.querySelector('[data-key="tdmaControl"]').checked,
    talkgroups: row.querySelector('[data-key="talkgroups"]').value.split(',').map(value=>{
      const parts=value.split(':'), id=Number(parts.shift()?.trim());
      if(!Number.isInteger(id)||id<1)return null;
      let mode=(parts.length>1?parts.pop().trim():'D').toUpperCase()||'D';
      const name=parts.join(':').trim()||`Talkgroup ${id}`;
      return {id,name,mode,encrypted:mode.includes('E'),enabled:true};
    }).filter(Boolean), enabled:true
  }));
  profile.settings.recordAudio = $('#record-audio').checked;
  profile.settings.transcribeVoice = $('#transcribe-voice').checked;
  return profile;
}

function downloadJSON(value, filename) {
  const url = URL.createObjectURL(new Blob([JSON.stringify(value, null, 2)], {type:'application/json'}));
  const link = document.createElement('a'); link.href = url; link.download = filename; link.click(); URL.revokeObjectURL(url);
}

function shareFrequency(frequencyHz, label, modulation) {
  const profile = emptyProfile();
  profile.name = label || `${shortFrequency(frequencyHz)} MHz`;
  profile.summary = 'Shared GP-SDR channel';
  profile.ranges = [];
  const modeMap = {AM:'am',NFM:'nfm',WFM:'wfm',Digital:'digital',Unknown:'auto',Burst:'auto'};
  profile.channels = [{id:crypto.randomUUID(),name:profile.name,frequencyHz,bandwidthHz:12500,mode:modeMap[modulation]||'auto',decoder:null,enabled:true,priority:5}];
  downloadJSON(profile, `GP-SDR-${shortFrequency(frequencyHz)}MHz.json`);
  toast('Channel profile exported');
}

function referenceRange() {
  const selected = $('#reference-range').value;
  return selected === 'custom' ? Number($('#custom-range').value) : Number(selected);
}

function renderReferenceResults() {
  const result = state.referenceResult;
  if (!result) return;
  const channels = result.channels || [], systems = result.p25Systems || [];
  $('#reference-status').textContent = `${result.location.city || 'ZIP ' + result.location.zip} · ${result.radiusMiles} miles · ${result.counties.length} counties`;
  $('#reference-results').classList.remove('hidden');
  $('#reference-results').innerHTML = `
    <section class="reference-group"><h3>Channels · ${channels.length}</h3><div class="reference-list">
      ${channels.length ? channels.map((channel,index)=>`<label class="reference-item"><input type="checkbox" data-reference-kind="channel" value="${index}" checked>
        <span><strong>${escapeHTML(channel.name || 'Unlabeled')}</strong><small>${escapeHTML(channel.county)} · ${escapeHTML(channel.mode.toUpperCase())}</small></span><span>${shortFrequency(channel.frequencyHz)}</span></label>`).join('') : '<div class="editor-empty">No conventional channels</div>'}
    </div></section>
    <section class="reference-group"><h3>P25 systems · ${systems.length}</h3><div class="reference-list">
      ${systems.length ? systems.map((system,index)=>`<label class="reference-item"><input type="checkbox" data-reference-kind="p25" value="${index}" checked>
        <span><strong>${escapeHTML(system.name)}</strong><small>${(system.talkgroups || []).length} talkgroups · encrypted disabled</small></span><span>${system.controlChannelsHz.length} CC</span></label>`).join('') : '<div class="editor-empty">No nearby P25 sites</div>'}
    </div></section>`;
  $('#reference-import').disabled = channels.length + systems.length === 0;
}

async function importReferenceSelection() {
  const result = state.referenceResult;
  if (!result) return;
  const channelIndexes = $$('[data-reference-kind="channel"]:checked').map(input=>Number(input.value));
  const systemIndexes = $$('[data-reference-kind="p25"]:checked').map(input=>Number(input.value));
  if (!channelIndexes.length && !systemIndexes.length) return toast('Select at least one item', true);
  const baseName = `${result.location.city || result.location.zip} · ${result.radiusMiles} mi`;
  const profiles = [];
  if (channelIndexes.length) {
    const profile = emptyProfile();
    profile.name = baseName + ' · Channels';
    profile.summary = 'RadioReference location import';
    profile.channels = channelIndexes.map(index => {
      const channel = result.channels[index];
      return {id:crypto.randomUUID(),name:channel.name||channel.description||'Channel',frequencyHz:channel.frequencyHz,
        bandwidthHz:channel.mode==='wfm'?180000:12500,mode:channel.mode,decoder:null,enabled:true,priority:5};
    });
    profiles.push(profile);
  }
  if (systemIndexes.length) {
    for (const index of systemIndexes) {
      const system = result.p25Systems[index];
      const profile = emptyProfile();
      profile.name = `${baseName} · ${system.name}`;
      profile.summary = 'RadioReference P25 location import';
      profile.channels = [];
      profile.deviceAssignments = [
        {id:crypto.randomUUID(),deviceID:null,role:'control',target:'P25 control'},
        {id:crypto.randomUUID(),deviceID:null,role:'voice',target:'P25 voice'}
      ];
      profile.p25Systems = [{id:crypto.randomUUID(),name:system.name,controlChannelsHz:system.controlChannelsHz,nac:system.nac||'',
        wacn:system.wacn||'',systemID:system.systemID||'',tdmaControl:false,talkgroups:system.talkgroups||[],enabled:true}];
      profiles.push(profile);
    }
  }
  try {
    for (const profile of profiles) await api('/api/profiles',{method:'POST',body:JSON.stringify(profile)});
    $('#reference-dialog').close(); toast(profiles.length === 1 ? 'Profile imported' : `${profiles.length} profiles imported`); await refreshAll();
  } catch (error) { toast(error.message,true); }
}

document.addEventListener('click', async event => {
  const dialogCancel = event.target.closest('dialog button[value="cancel"]');
  if (dialogCancel) { event.preventDefault(); dialogCancel.closest('dialog').close(); return; }
  const guide = event.target.closest('.guide-component');
  if (guide) { openSetupGuide(guide.dataset.componentId); return; }
  const install = event.target.closest('.install-component');
  if (install) {
    install.disabled = true;
    try {
      await api('/api/setup/install',{method:'POST',body:JSON.stringify({componentID:install.dataset.componentId})});
      toast('Installation started'); await refreshAll();
    } catch (error) { toast(error.message,true); await refreshAll(); }
    return;
  }
  const nav = event.target.closest('.nav-item'); if (nav) return setView(nav.dataset.view);
  const removeRemote=event.target.closest('.remove-remote'); if(removeRemote){try{await api('/api/remote-receivers?id='+encodeURIComponent(removeRemote.dataset.remoteId),{method:'DELETE'});toast('Remote receiver removed');await refreshAll();}catch(error){toast(error.message,true);}return;}
  const decoderLink = event.target.closest('[data-decoder-id]');
  if (decoderLink) {
    state.selectedDecoderID = decoderLink.dataset.decoderId; setView('decoders'); renderDecoders();
    history.replaceState(null,'',`#decoder/${encodeURIComponent(state.selectedDecoderID)}`); return;
  }
  const tab = event.target.closest('[data-activity-tab]'); if (tab) {
    state.activityTab = tab.dataset.activityTab; $$('.segmented button').forEach(b=>b.classList.toggle('active',b===tab));
    $('#activity-signals').classList.toggle('hidden',state.activityTab!=='signals'); $('#activity-events').classList.toggle('hidden',state.activityTab!=='events'); return;
  }
  const calibrationBox = event.target.closest('[data-calibration-device]');
  if (calibrationBox) {
    const deviceID = calibrationBox.dataset.calibrationDevice;
    const device = state.devices.find(item => item.id === deviceID);
    const referenceHz = Number(calibrationBox.querySelector('.calibration-reference').value) * 1e6;
    const button = event.target.closest('button');
    if (!button) return;
    button.disabled = true;
    try {
      if (button.classList.contains('auto-calibration')) {
        button.textContent = 'Measuring…';
        const result = await api('/api/calibrations/auto',{method:'POST',body:JSON.stringify({deviceID,referenceHz,sampleRateHz:2400000,lnaGainDB:Number($('#tuner-lna').value),vgaGainDB:Number($('#tuner-vga').value)})});
        toast(`Calibration saved · ${result.ppmCorrection >= 0 ? '+' : ''}${result.ppmCorrection} PPM`);
      } else if (button.classList.contains('edit-calibration')) {
        await api('/api/calibrations',{method:'PUT',body:JSON.stringify({deviceID,deviceKind:device.kind,serial:device.serial||'',referenceHz,ppmCorrection:Number($('#tuner-ppm').value),iqGain:Number($('#tuner-iq-gain').value),iqPhase:Number($('#tuner-iq-phase').value),iqSwap:$('#tuner-iq-swap').checked,dcRemoval:$('#tuner-dc').checked,lnaGainDB:Number($('#tuner-lna').value),vgaGainDB:Number($('#tuner-vga').value),ampEnabled:$('#tuner-amp').checked,confidence:1,signalToNoiseDB:0,source:'manual'})});
        toast('Calibration saved and enabled');
      } else if (button.classList.contains('reset-calibration')) {
        await api('/api/calibrations?deviceID='+encodeURIComponent(deviceID),{method:'DELETE'});
        toast('Calibration reset');
      }
      await refreshAll();
    } catch (error) { toast(error.message,true); button.disabled=false; if(button.classList.contains('auto-calibration'))button.textContent='Auto calibrate'; }
    return;
  }
  const card = event.target.closest('[data-profile-id]');
  try {
    if (event.target.closest('.decoder-profile')) { state.selectedProfileID = card.dataset.profileId; renderProfileSelect(); renderProfiles(); toast('Profile selected'); return; }
    if (event.target.closest('.select-profile')) { state.selectedProfileID = card.dataset.profileId; renderProfiles(); renderProfileSelect(); return; }
    if (event.target.closest('.duplicate-profile')) { await api(`/api/profiles/duplicate?id=${card.dataset.profileId}`,{method:'POST'}); toast('Editable copy created'); return refreshAll(); }
    if (event.target.closest('.edit-profile')) { return openProfileEditor(state.profiles.find(p=>p.id===card.dataset.profileId)); }
    if (event.target.closest('.export-profile')) { window.location.href = `/api/profiles/export?id=${card.dataset.profileId}${serverToken ? `&token=${encodeURIComponent(serverToken)}` : ''}`; return; }
    const share = event.target.closest('.share-signal'); if (share) return shareFrequency(Number(share.dataset.frequency),share.dataset.label,share.dataset.mode);
    const play = event.target.closest('.play-event'); if (play) {
      if (recordingPlayer.dataset.eventId === play.dataset.eventId && !recordingPlayer.paused) recordingPlayer.pause();
      else {
        recordingPlayer.dataset.eventId = play.dataset.eventId;
        recordingPlayer.src = '/api/audio?id=' + encodeURIComponent(play.dataset.eventId) + (serverToken ? '&token=' + encodeURIComponent(serverToken) : '');
        await recordingPlayer.play();
      }
      return;
    }
    const mixerRow = event.target.closest('[data-mixer-id]');
    if (mixerRow && (event.target.closest('.mixer-mute') || event.target.closest('.mixer-solo'))) {
      const item = state.mixer.find(i=>i.id===mixerRow.dataset.mixerId); const isMute=!!event.target.closest('.mixer-mute');
      await api('/api/mixer',{method:'POST',body:JSON.stringify({id:item.id,[isMute?'muted':'solo']:!item[isMute?'muted':'solo']})}); return refreshAll();
    }
    if (event.target.closest('.decoder-mute-all')) {
      const talkgroups = state.mixer.filter(item=>item.talkgroupID), mute = talkgroups.some(item=>!item.muted);
      await Promise.all(talkgroups.map(item=>api('/api/mixer',{method:'POST',body:JSON.stringify({id:item.id,muted:mute})})));
      return refreshAll();
    }
  } catch (error) { toast(error.message,true); }
});

$('#active-profile').addEventListener('change', event => { state.selectedProfileID = event.target.value; renderProfiles(); });
$('#survey-toggle').addEventListener('click', async () => {
  try {
    if (state.status?.running) { await api('/api/control/stop',{method:'POST',body:'{}'}); stopLiveAudio(); }
    else if (state.selectedProfileID) { void startLiveAudio(); await api('/api/control/start',{method:'POST',body:JSON.stringify({profileID:state.selectedProfileID})}); }
    await refreshAll();
  } catch (error) { stopLiveAudio(); toast(error.message,true); }
});
$('#tuner-frequency').addEventListener('input', event => $('#tuner-readout').textContent = Number(event.target.value || 0).toFixed(4));
$('#tuner-mode').addEventListener('change', event => {
	$('#tuner-bandwidth').value = event.target.value === 'wfm' ? '180' : event.target.value === 'am' ? '10' : '12.5';
	if(event.target.value==='wfm'){ $('#tuner-lna').value='32'; $('#tuner-vga').value='24'; $('#tuner-squelch').value='4'; $('#tuner-agc').checked=true; $('#tuner-monitor-open').checked=true; }
	queueReceiverControls();
});

async function applyReceiverControls() {
	if(receiverApplying) return;
	receiverApplying=true; const panel=$('.receiver-controls-panel'), button=$('#live-apply-radio'); panel?.classList.add('applying'); panel?.classList.remove('applied'); button.disabled=true; button.textContent='Applying…'; $('#audio-state').textContent='Applying settings';
	try { await startLiveAudio(); await api('/api/tuner/start',{method:'POST',body:JSON.stringify(tunerRequest())}); button.textContent='Applied'; panel?.classList.add('applied'); $('#audio-state').textContent='Settings applied'; $('#tuner-status').textContent='Settings applied'; }
	catch(error) { button.textContent='Retry'; $('#audio-state').textContent='Apply failed'; $('#tuner-status').textContent='Apply failed'; toast(error.message,true); }
	finally { receiverApplying=false; button.disabled=false; panel?.classList.remove('applying'); }
}

function queueReceiverControls() { if(!state.status?.running || state.status?.activeProfileID!=='quick-tune') return; clearTimeout(receiverApplyTimer); $('#live-apply-radio').textContent='Pending…'; $('#audio-state').textContent='Settings pending'; $('#tuner-status').textContent='Applying settings…'; receiverApplyTimer=setTimeout(applyReceiverControls,250); }

$$('#view-live .receiver-control-grid input, #view-live .receiver-control-grid select').forEach(control=>control.addEventListener(control.type==='number'?'input':'change',()=>{
	const map={'live-radio-device':'tuner-device','live-lna':'tuner-lna','live-vga':'tuner-vga','live-ppm':'tuner-ppm','live-iq-gain':'tuner-iq-gain','live-iq-phase':'tuner-iq-phase','live-squelch':'tuner-squelch','live-amp':'tuner-amp','live-bias':'tuner-bias','live-dc':'tuner-dc','live-iq-swap':'tuner-iq-swap','live-agc':'tuner-agc','live-monitor-open':'tuner-monitor-open','live-use-calibration':'tuner-use-calibration'};
	const target=$('#'+map[control.id]); if(target){if(control.type==='checkbox')target.checked=control.checked;else target.value=control.value;} queueReceiverControls();
}));
$$('#view-tuner .advanced-radio input, #view-tuner .advanced-radio select, #tuner-gain, #tuner-rate').forEach(control=>control.addEventListener(control.type==='number'?'input':'change',queueReceiverControls));
$('#tuner-form').addEventListener('submit', async event => {
  event.preventDefault();
	const request = tunerRequest();
  try { void startLiveAudio(); await api('/api/tuner/start',{method:'POST',body:JSON.stringify(request)}); toast('Tuner started'); await refreshAll(); }
  catch(error) { stopLiveAudio(); toast(error.message,true); }
});
$('#live-apply-radio').addEventListener('click', async () => {
  [['live-radio-device','tuner-device'],['live-lna','tuner-lna'],['live-vga','tuner-vga'],['live-ppm','tuner-ppm'],['live-iq-gain','tuner-iq-gain'],['live-iq-phase','tuner-iq-phase'],['live-squelch','tuner-squelch']].forEach(([from,to])=>$('#'+to).value=$('#'+from).value);
  [['live-amp','tuner-amp'],['live-bias','tuner-bias'],['live-dc','tuner-dc'],['live-iq-swap','tuner-iq-swap'],['live-agc','tuner-agc'],['live-monitor-open','tuner-monitor-open'],['live-use-calibration','tuner-use-calibration']].forEach(([from,to])=>$('#'+to).checked=$('#'+from).checked);
	try { await applyReceiverControls(); }
  catch(error) { stopLiveAudio(); toast(error.message,true); }
});
$('#audio-monitor').addEventListener('click', async () => {
  try { await startLiveAudio(); toast(liveAudio.context?.state === 'running' ? 'Audio monitor ready' : 'Audio output is blocked'); }
  catch(error) { toast(error.message,true); }
});
$('#tuner-stop').addEventListener('click', async () => {
  try { await api('/api/tuner/stop',{method:'POST',body:'{}'}); stopLiveAudio(); await refreshAll(); }
  catch(error) { toast(error.message,true); }
});
$('#signal-search').addEventListener('input', renderSignals);
$('#new-profile').addEventListener('click', () => openProfileEditor());
$('#reference-button').addEventListener('click', () => $('#reference-dialog').showModal());
$('#reference-range').addEventListener('change', event => $('#custom-range-wrap').classList.toggle('hidden', event.target.value !== 'custom'));
$('#reference-form').addEventListener('submit', async event => {
  if (event.submitter?.value === 'cancel') return;
  event.preventDefault();
  const zipCode = $('#reference-zip').value.trim(), radius = referenceRange();
  if (!/^[0-9]{5}$/.test(zipCode) || !Number.isFinite(radius) || radius < 1 || radius > 100) return toast('Enter a ZIP and a range from 1 to 100 miles', true);
  $('#reference-status').textContent = 'Searching…'; $('#reference-results').classList.add('hidden'); $('#reference-import').disabled = true;
  try {
    state.referenceResult = await api('/api/radioreference/nearby?zip=' + encodeURIComponent(zipCode) + '&radius=' + encodeURIComponent(radius));
    renderReferenceResults();
  } catch (error) { $('#reference-status').textContent = error.message; toast(error.message,true); }
});
$('#reference-import').addEventListener('click', importReferenceSelection);
$('#import-button').addEventListener('click', () => $('#import-profile').click());
$('#import-profile').addEventListener('change', async event => {
  const files = [...event.target.files]; if (!files.length) return;
  let imported = 0;
  try {
    for (const file of files) {
      const isJSON = file.name.toLowerCase().endsWith('.json');
      const endpoint = isJSON ? '/api/profiles/import' : '/api/profiles/import-channels?filename=' + encodeURIComponent(file.name);
      await api(endpoint,{method:'POST',body:await file.text()}); imported++;
    }
    toast(imported === 1 ? 'Import complete' : `${imported} channel banks imported`); await refreshAll();
  } catch(error){toast(`${imported ? `${imported} imported · ` : ''}${error.message}`,true);} event.target.value='';
});
$('#refresh-hardware').addEventListener('click', async () => {
  try { await api('/api/devices/refresh',{method:'POST',body:'{}'}); toast('Hardware refreshed'); await refreshAll(); }
  catch(error){toast(error.message,true);}
});
$('#remote-form').addEventListener('submit',async event=>{event.preventDefault();try{await api('/api/remote-receivers',{method:'PUT',body:JSON.stringify({name:$('#remote-name').value.trim(),host:$('#remote-host').value.trim(),port:Number($('#remote-port').value),enabled:true})});toast('Remote receiver saved');$('#remote-host').value='';await refreshAll();}catch(error){toast(error.message,true);}});
$('#mapper-form').addEventListener('submit',async event=>{
  event.preventDefault(); const start=Number($('#mapper-start').value)*1e6,end=Number($('#mapper-end').value)*1e6,deviceID=$('#mapper-device').value,mode=$('#mapper-mode').value;let step=Number($('#mapper-step').value)*1000;
  if(!deviceID||!Number.isFinite(start)||!Number.isFinite(end)||end<=start||step<=0)return toast('Select a receiver and enter a valid range',true);
  if((end-start)/step>20000){step=Math.ceil((end-start)/20000/1000)*1000;toast(`Step increased to ${step/1000} kHz for a responsive scan`);}
  const profile={schemaVersion:1,id:'mapper-session',name:'Mapper Session',summary:'Temporary wide-range activity map',ranges:[{id:'mapper-range',name:'Mapper range',startHz:start,endHz:end,stepHz:step,dwellMilliseconds:120,preferredMode:mode,enabled:true}],channels:[],deviceAssignments:[{id:'mapper-device',deviceID,role:'survey',target:'Mapper'}],p25Systems:[],settings:{noiseMarginDB:8,revisitSeconds:10,recordAudio:false,recordIQForUnknown:false,transcribeVoice:false,maxRecordingDays:30},builtIn:false};
  try{if(state.status?.running)await api('/api/control/stop',{method:'POST',body:'{}'});await api('/api/profiles',{method:'POST',body:JSON.stringify(profile)});await api('/api/control/start',{method:'POST',body:JSON.stringify({profileID:'mapper-session'})});toast('Mapper started');await refreshAll();}catch(error){toast(error.message,true);}
});
$('#mapper-stop-button').addEventListener('click',async()=>{try{await api('/api/control/stop',{method:'POST',body:'{}'});toast('Mapper stopped');await refreshAll();}catch(error){toast(error.message,true);}});
$('#mapper-sheet-form').addEventListener('submit',async event=>{event.preventDefault();try{state.mapper=await api('/api/mapper',{method:'PUT',body:JSON.stringify({webhookURL:$('#mapper-webhook').value.trim(),secret:$('#mapper-secret').value,autoUpload:$('#mapper-auto-upload').checked})});renderMapper();toast('Mapper upload settings saved');}catch(error){toast(error.message,true);}});
$('#mapper-upload-now').addEventListener('click',async()=>{try{state.mapper=await api('/api/mapper/upload',{method:'POST',body:'{}'});renderMapper();if(state.mapper.lastError)toast(state.mapper.lastError,true);else toast('New Mapper activity uploaded');}catch(error){toast(error.message,true);}});
$('#range-sync-form').addEventListener('submit', async event => {
  event.preventDefault();
  const config = {sheetURL:$('#range-sync-url').value.trim(),intervalMinutes:Number($('#range-sync-interval').value),enabled:$('#range-sync-enabled').checked};
  try { state.rangeSync = await api('/api/range-sync',{method:'PUT',body:JSON.stringify(config)}); renderRangeSync(); toast('Range sync saved'); setTimeout(refreshAll,800); }
  catch(error){toast(error.message,true);}
});
$('#range-sync-now').addEventListener('click', async () => {
  try { state.rangeSync = await api('/api/range-sync/now',{method:'POST',body:'{}'}); renderRangeSync(); if(state.rangeSync.lastError) toast(state.rangeSync.lastError,true); else { toast('Ranges updated'); await refreshAll(); } }
  catch(error){toast(error.message,true);}
});
$('#mute-all').addEventListener('click', async () => {
  const mute = state.mixer.some(item=>!item.muted); await Promise.all(state.mixer.map(item=>api('/api/mixer',{method:'POST',body:JSON.stringify({id:item.id,muted:mute})}))); refreshAll();
});
document.addEventListener('change', async event => {
  const slider=event.target.closest('.mixer-volume'); if(!slider)return; const row=slider.closest('[data-mixer-id]');
  try{await api('/api/mixer',{method:'POST',body:JSON.stringify({id:row.dataset.mixerId,volume:Number(slider.value)})});}catch(error){toast(error.message,true);}
});

$('#add-range').addEventListener('click',()=>{state.editingProfile.ranges.push({id:crypto.randomUUID(),name:'New range',startHz:144e6,endHz:148e6,stepHz:12500,dwellMilliseconds:180,preferredMode:'auto',enabled:true});renderEditorRows();});
$('#add-channel').addEventListener('click',()=>{state.editingProfile.channels.push({id:crypto.randomUUID(),name:'New channel',frequencyHz:462.55e6,bandwidthHz:12500,mode:'nfm',decoder:null,enabled:true,priority:5});renderEditorRows();});
$('#add-assignment').addEventListener('click',()=>{state.editingProfile.deviceAssignments.push({id:crypto.randomUUID(),deviceID:null,role:'automatic',target:null});renderEditorRows();});
$('#add-p25').addEventListener('click',()=>{state.editingProfile.p25Systems.push({id:crypto.randomUUID(),name:'P25 system',controlChannelsHz:[],nac:'',wacn:'',systemID:'',tdmaControl:false,talkgroups:[],enabled:true});renderEditorRows();});
$('#profile-dialog').addEventListener('click',event=>{
  const remove=event.target.closest('.remove-row');if(!remove)return;const row=remove.closest('.editor-row');
  for(const key of ['ranges','channels','deviceAssignments','p25Systems']){const index=state.editingProfile[key].findIndex(item=>item.id===row.dataset.id);if(index>=0){state.editingProfile[key].splice(index,1);break;}}renderEditorRows();
});
$('#profile-form').addEventListener('submit',async event=>{
  if(event.submitter?.value==='cancel')return;event.preventDefault();
  try{const profile=collectEditor();await api('/api/profiles',{method:'POST',body:JSON.stringify(profile)});$('#profile-dialog').close();toast('Profile saved');await refreshAll();}
  catch(error){toast(error.message,true);}
});

window.addEventListener('resize',()=>{drawSpectrum();drawWaterfall();});
const decoderHash = location.hash.match(/^#decoder\/(.+)$/);
if (decoderHash) { state.selectedDecoderID = decodeURIComponent(decoderHash[1]); setView('decoders'); }
const interfaceMode=localStorage.getItem('gpsdr-interface-mode')||'beginner';
$('#interface-mode').value=interfaceMode;document.body.classList.toggle('advanced-mode',interfaceMode==='advanced');
$('#interface-mode').addEventListener('change',event=>{const advanced=event.target.value==='advanced';document.body.classList.toggle('advanced-mode',advanced);localStorage.setItem('gpsdr-interface-mode',event.target.value);toast(advanced?'Advanced controls enabled':'Automatic controls enabled');});
refreshAll();
setInterval(async()=>{
  if(document.hidden)return;
	try{const [status,mixer,p25Status]=await Promise.all([api('/api/status'),api('/api/mixer'),api('/api/p25/status')]);Object.assign(state,{status,mixer,p25Status});renderStatus();renderMixer();renderTuner();if(state.view==='decoders')renderDecoders();}catch(_){ }
},750);
setInterval(async()=>{
	if(document.hidden)return;
	try{const [events,signals]=await Promise.all([api('/api/events?limit=150'),api('/api/signals?limit=400')]);Object.assign(state,{events,signals});renderLatest();if(state.view==='activity'){renderSignals();renderEvents();}}catch(_){ }
},5000);
async function pollSpectrum() {
	if(!document.hidden && state.status?.running && (state.view==='live'||state.view==='tuner')){try{state.spectrum=await api('/api/spectrum?bins='+displayPrefs.detail);renderTuner();drawSpectrum();drawWaterfall();}catch(_){ }}
	setTimeout(pollSpectrum,Math.max(40,1000/displayPrefs.fps));
}
pollSpectrum();

function saveDisplayPrefs(){localStorage.setItem('gpsdr-display-v2',JSON.stringify(displayPrefs));$('#display-smoothing-value').textContent=displayPrefs.smoothing+'%';$$('#live-waterfall,#waterfall').forEach(canvas=>{canvas.dataset.lastFrame='';canvas.width=0;});drawSpectrum();drawWaterfall();}
for(const [id,key,number] of [['display-fps','fps',true],['display-quality','quality',true],['display-detail','detail',true],['display-smoothing','smoothing',true]]){const control=$('#'+id);control.value=displayPrefs[key];control.addEventListener('input',()=>{displayPrefs[key]=number?Number(control.value):control.value;saveDisplayPrefs();});}
saveDisplayPrefs();
