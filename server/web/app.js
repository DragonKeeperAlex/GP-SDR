const state = {
  status: null, profiles: [], events: [], signals: [], devices: [], decoders: [], mixer: [],
  integrations: null, setup: null, p25Status: null, spectrum: null, referenceResult: null,
  rangeSync: null, localDatabase: null, calibrations: [], characterization: null, mapper: null, mapperProgress: null, remoteReceivers: [],
  selectedProfileID: null, selectedDecoderID: 'p25', editingProfile: null, activityTab: 'signals', view: 'live',
  p25Order: localStorage.getItem('gpsdr-p25-order') || 'recent',
  mixerOrder: localStorage.getItem('gpsdr-mixer-order') || 'active'
};
const serverToken = new URLSearchParams(location.search).get('token') || '';

const $ = selector => document.querySelector(selector);
const $$ = selector => [...document.querySelectorAll(selector)];
const encoder = new TextEncoder();
let toastTimer;
let setupPollTimer;
let lastWaterfallFrame = '';
const masterAudio = (()=>{try{return {volume:.8,muted:false,...JSON.parse(localStorage.getItem('gpsdr-master-audio-v1')||'{}')}}catch(_){return {volume:.8,muted:false}}})();
const recordingPlayer = new Audio();
const liveAudio = { context:null, controller:null, masterGain:null, gains:new Map(), panners:new Map(), nextTimes:new Map() };
let receiverApplyTimer, receiverApplying = false;
const displayPrefs = (()=>{try{return {fps:8,quality:.75,detail:512,smoothing:20,peakHold:false,markers:true,floor:-120,ceiling:-20,...JSON.parse(localStorage.getItem('gpsdr-display-v2')||'{}')}}catch(_){return {fps:8,quality:.75,detail:512,smoothing:20,peakHold:false,markers:true,floor:-120,ceiling:-20}}})();
const spectrumHistory = new WeakMap();
const spectrumPeaks = new WeakMap();
const spectrumCursors = new WeakMap();
let tunerHistory=(()=>{try{return JSON.parse(localStorage.getItem('gpsdr-tuner-history')||'[]')}catch(_){return []}})();
let mapperResultsCollapsed=localStorage.getItem('gpsdr-mapper-results-collapsed')==='true';

function renderTunerHistory(){const select=$('#tuner-history');if(!select)return;select.innerHTML=tunerHistory.length?tunerHistory.map(item=>`<option value="${item.frequencyHz}">${escapeHTML(item.label)}</option>`).join(''):'<option value="">No recent frequencies</option>';select.value='';}
function rememberTunerFrequency(request){const label=`${(request.frequencyHz/1e6).toFixed(6)} MHz · ${String(request.mode).toUpperCase()}`;tunerHistory=[{frequencyHz:request.frequencyHz,mode:request.mode,bandwidthHz:request.bandwidthHz,label},...tunerHistory.filter(item=>Math.abs(item.frequencyHz-request.frequencyHz)>1)].slice(0,12);localStorage.setItem('gpsdr-tuner-history',JSON.stringify(tunerHistory));renderTunerHistory();}
const expandedMapperFrequencies = new Set();

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

function confirmAction({title='Confirm action',message,confirmLabel='Continue'}={}) {
  const dialog=$('#confirm-dialog'),accept=$('#confirm-dialog-accept');
  if(!dialog||typeof dialog.showModal!=='function')return Promise.resolve(false);
  if(dialog.open)dialog.close('cancel');
  $('#confirm-dialog-title').textContent=title;
  $('#confirm-dialog-message').textContent=message||'Continue with this action?';
  accept.textContent=confirmLabel;
  dialog.returnValue='cancel';
  return new Promise(resolve=>{
    dialog.addEventListener('close',()=>resolve(dialog.returnValue==='confirm'),{once:true});
    dialog.showModal();
    requestAnimationFrame(()=>$('#confirm-dialog-cancel').focus());
  });
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

function compactDuration(totalSeconds) {
  const seconds = Math.max(0, Math.floor(Number(totalSeconds) || 0));
  const days = Math.floor(seconds / 86400), hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60), remainder = seconds % 60;
  if (days) return `${days}d ${hours}h`;
  if (hours) return `${hours}h ${minutes}m`;
  if (minutes) return `${minutes}m ${remainder}s`;
  return `${remainder}s`;
}

function formatBytes(value) {
  let bytes=Math.max(0,Number(value)||0),unit='B';
  for(const next of ['KB','MB','GB','TB']){if(bytes<1024)break;bytes/=1024;unit=next;}
  return `${bytes>=10||unit==='B'?bytes.toFixed(0):bytes.toFixed(1)} ${unit}`;
}

function durationBetween(start, end = null) {
  if (!start) return '—';
  return compactDuration((new Date(end || Date.now()).getTime() - new Date(start).getTime()) / 1000);
}

function setView(view) {
  state.view = view;
	document.body.dataset.view=view;
  localStorage.setItem('gpsdr-last-view', view);
  $$('.nav-item').forEach(button => {
    const active = button.dataset.view === view;
    button.classList.toggle('active', active);
    if (active) button.setAttribute('aria-current', 'page'); else button.removeAttribute('aria-current');
  });
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
  document.title = `${copy[0]} · GP-SDR`;
  if (view === 'tuner' || view === 'mapper') { drawSpectrum(); drawWaterfall(); if(view==='mapper')renderMapperRF(); }
}

async function refreshAll() {
  try {
    const [status, profiles, events, signals, devices, decoders, mixer, integrations, setup, p25Status, spectrum, rangeSync, localDatabase, calibrations, characterization, mapper, mapperProgress, remoteReceivers] = await Promise.all([
      api('/api/status'), api('/api/profiles'), api('/api/events?limit=300'), api('/api/signals?limit=1000'),
      api('/api/devices'), api('/api/decoders'), api('/api/mixer'), api('/api/integrations'), api('/api/setup'),
      api('/api/p25/status'), api('/api/spectrum'), api('/api/range-sync'), api('/api/local-database'), api('/api/calibrations'), api('/api/calibrations/characterization'), api('/api/mapper'), api('/api/mapper/progress'), api('/api/remote-receivers')
    ]);
    Object.assign(state, { status, profiles: profiles || [], events: events || [], signals: signals || [], devices: devices || [], decoders: decoders || [], mixer: mixer || [], integrations, setup, p25Status, spectrum, rangeSync, localDatabase, calibrations: calibrations || [], characterization, mapper, mapperProgress, remoteReceivers: remoteReceivers || [] });
    if (!state.selectedProfileID || !profiles.some(profile => profile.id === state.selectedProfileID)) {
      state.selectedProfileID = status.activeProfileID || profiles[0]?.id || null;
    }
    render();
    document.body.classList.remove('server-offline');
    $('#side-status').title = `Updated ${new Date().toLocaleTimeString()}`;
    clearTimeout(setupPollTimer);
    if (state.setup?.job?.state === 'installing') setupPollTimer = setTimeout(refreshAll, 1200);
  } catch (error) {
    $('#side-status').textContent = 'Offline';
    $('#side-status').title = error.message;
    $('#side-dot').classList.remove('live');
    document.body.classList.add('server-offline');
    toast(error.message, true);
  }
}

function render() {
  renderStatus(); renderProfileSelect(); renderLatest(); renderMixer(); renderSignals();
  renderEvents(); renderProfiles(); renderHardware(); renderCharacterization(); renderIntegrations(); renderRadioReferenceSettings(); renderRangeSync(); renderLocalDatabase(); renderTuner(); renderDecoders(); renderMapper(); renderMissingComponents(); drawSpectrum(); drawWaterfall();
}

function renderMissingComponents() {
  const dialog=$('#missing-components-dialog'), list=$('#missing-components-list');
  if(!dialog||!list||!state.setup?.components)return;
  const missing=state.setup.components.filter(item=>item.state!=='ready'&&item.id!=='radioreference');
  if(!missing.length){if(dialog.open)dialog.close();return;}
  const signature=`${state.status?.version||''}:`+missing.map(item=>item.id).sort().join(',');
  list.innerHTML=missing.map(item=>`<div class="missing-component-row"><span><strong>${escapeHTML(item.name)}</strong><small>${escapeHTML(item.guide||item.note||'Optional feature')}</small></span>${setupActions(item.id)}</div>`).join('');
  dialog.dataset.signature=signature;
  if(!dialog.open&&localStorage.getItem('gpsdr-ignored-components')!==signature&&sessionStorage.getItem('gpsdr-dismissed-components')!==signature) requestAnimationFrame(()=>{if(!dialog.open)dialog.showModal();});
}

function renderLocalDatabase(){const status=state.localDatabase;if(!status)return;const badge=$('#local-db-state');badge.textContent=status.scanning?'Scanning':status.lastError?'Needs attention':status.lastScan?'Ready':status.folder?'Waiting':'Not configured';badge.className=`chip ${status.lastError?'warning':status.lastScan&&!status.scanning?'ready':''}`;$('#local-db-path').textContent=status.folder||'No folder selected.';$('#local-db-path').title=status.folder||'';if(!$('#local-db-form').contains(document.activeElement))$('#local-db-folder').value=status.folder||'';$('#local-db-detail').textContent=status.scanning?'Scanning supported files…':status.lastError||`${status.files||0} files · ${status.profiles||0} profiles · ${status.channels||0} channels${status.skippedFiles?` · ${status.skippedFiles} skipped`:''}`;$('#local-db-scan').disabled=!status.folder||status.scanning||!status.canManage;$('#local-db-choose').disabled=status.scanning||!status.canManage;if(!status.canManage)$('#local-db-detail').textContent='Open the native GP-SDR app on the server to choose its local folder.';}

window.setLocalDatabaseFolder=async function(folder){try{state.localDatabase=await api('/api/local-database',{method:'PUT',body:JSON.stringify({folder})});renderLocalDatabase();toast('Local database folder saved');setTimeout(refreshAll,800);}catch(error){toast(error.message,true);}};

function setMapperLocation(latitude,longitude){const button=$('#mapper-current-location');$('#mapper-latitude').value=Number(latitude).toFixed(6);$('#mapper-longitude').value=Number(longitude).toFixed(6);$('#mapper-location').checked=true;button.textContent='Location added';button.dataset.locationSettings='';toast('Current location added');}
function mapperLocationError(message,settings=false){const button=$('#mapper-current-location');button.textContent=settings?'Open Location Settings':'Use current location';button.dataset.locationSettings=settings?'true':'';toast(message||'Could not read location',true);}
window.gpsdrNativeLocationResult=function(result){if(result?.status==='success')setMapperLocation(result.latitude,result.longitude);else mapperLocationError(result?.message,result?.settings);};

function renderRadioReferenceSettings() {
  const status=state.integrations?.radioReference, form=$('#rr-credentials-form'); if(!status||!form)return;
  const ready=status.state==='ready'; $('#rr-state').textContent=ready?'Ready':'Setup'; $('#rr-state').classList.toggle('ready',ready);
  const account=status.accountHint?` · ${status.accountHint}`:''; $('#rr-detail').textContent=`${status.credentialSource||'Not configured'}${account}${status.appKeyConfigured?' · app key saved':''}`;
  const local=status.canManage!==false; form.querySelectorAll('input,button').forEach(control=>control.disabled=!local); if(!local)$('#rr-detail').textContent='Open Settings in the native GP-SDR app to change credentials.';
}

function mapperFullyIdentified(record){const classification=String(record.protocolName||record.modulation||'').trim().toLowerCase();return !!record.identificationVerified&&!!String(record.name||'').trim()&&!!classification&&!['unknown','auto'].includes(classification)&&!classification.includes('candidate')&&!classification.includes('likely');}

function renderMapper(){
  const body=$('#mapper-body'); if(!body)return; const connected=state.devices.filter(d=>d.connected&&d.available),select=$('#mapper-device'),selected=select.value; const sig=connected.map(d=>d.id).join(); if(select.dataset.signature!==sig){select.innerHTML=connected.map(d=>`<option value="${escapeHTML(d.id)}">${escapeHTML(d.name)} · ${escapeHTML(d.kind)}</option>`).join('')||'<option value="">No receiver</option>';select.dataset.signature=sig;if(connected.some(d=>d.id===selected))select.value=selected;}
  const m=state.mapper||{},jobs=m.jobs||[],jobNames=new Map(jobs.map(job=>[job.id,job.name])),deviceNames=new Map(state.devices.map(device=>[device.id,device.name]));
  const jobFilter=$('#mapper-filter-job'),deviceFilter=$('#mapper-filter-device'),oldJob=jobFilter.value,oldDevice=deviceFilter.value;jobFilter.innerHTML='<option value="">All jobs</option>'+jobs.map(job=>`<option value="${escapeHTML(job.id)}">${escapeHTML(job.name)}</option>`).join('');deviceFilter.innerHTML='<option value="">All receivers</option>'+connected.map(device=>`<option value="${escapeHTML(device.id)}">${escapeHTML(device.name)}</option>`).join('');jobFilter.value=oldJob;deviceFilter.value=oldDevice;
  const allRecords=m.records||[],typeFilter=$('#mapper-filter-type'),oldType=typeFilter.value,types=[...new Set(allRecords.map(record=>record.protocolName||record.modulation).filter(Boolean))].sort((a,b)=>a.localeCompare(b));typeFilter.innerHTML='<option value="">All types</option>'+types.map(type=>`<option value="${escapeHTML(type)}">${escapeHTML(type)}</option>`).join('');typeFilter.value=types.includes(oldType)?oldType:'';
  const query=$('#mapper-filter-search').value.trim().toLowerCase(),resultState=$('#mapper-filter-state').value,repeatedOnly=$('#mapper-filter-repeated').checked,now=Date.now();
  const records=allRecords.filter(record=>{const identified=!!(record.name||record.protocolName||record.identificationSource),verified=mapperFullyIdentified(record),haystack=[record.frequencyHz,formatFrequency(record.frequencyHz),record.name,record.protocolName,record.modulation,record.identificationSource,record.verificationReason,record.candidateDecoder,record.detectionEvidence,record.analysisSummary,record.lastTranscript,...(record.callsigns||[])].filter(Boolean).join(' ').toLowerCase();return (!jobFilter.value||(record.jobIDs||[]).includes(jobFilter.value))&&(!deviceFilter.value||(record.deviceIDs||[]).includes(deviceFilter.value))&&(!typeFilter.value||(record.protocolName||record.modulation)===typeFilter.value)&&(!query||haystack.includes(query))&&(!repeatedOnly||Number(record.hits)>1)&&(!resultState||(resultState==='verified'&&verified)||(resultState==='identified'&&identified)||(resultState==='unidentified'&&!identified)||(resultState==='decoded'&&record.candidateDecoder)||(resultState==='transcribed'&&record.lastTranscript)||(resultState==='callsign'&&(record.callsigns||[]).length)||(resultState==='recent'&&now-new Date(record.lastSeen).getTime()<=3600000)||(resultState==='frequent'&&Number(record.hits)>=10)||(resultState==='repeat'&&Number(record.hits)>1)||(resultState==='oneoff'&&Number(record.hits)===1));});
  const sort=$('#mapper-sort').value,identity=record=>String(record.name||record.protocolName||record.modulation||'').toLowerCase(),snr=record=>Number(record.strongestDBFS)-Number(record.noiseDBFS);records.sort((a,b)=>sort==='frequency'?a.frequencyHz-b.frequencyHz:sort==='frequency-desc'?b.frequencyHz-a.frequencyHz:sort==='hits'?Number(b.hits)-Number(a.hits)||new Date(b.lastSeen)-new Date(a.lastSeen):sort==='checks'?Number(b.checks)-Number(a.checks):sort==='occupancy'?Number(b.occupancy)-Number(a.occupancy)||Number(b.hits)-Number(a.hits):sort==='strongest'?Number(b.strongestDBFS)-Number(a.strongestDBFS):sort==='snr'?snr(b)-snr(a):sort==='confidence'?Number(b.confidence)-Number(a.confidence)||new Date(b.lastSeen)-new Date(a.lastSeen):sort==='receivers'?(b.deviceIDs||[]).length-(a.deviceIDs||[]).length:sort==='first-seen'?new Date(a.firstSeen)-new Date(b.firstSeen):sort==='identity'?identity(a).localeCompare(identity(b)):new Date(b.lastSeen)-new Date(a.lastSeen));
  // Keep the table responsive with large long-running surveys.  The complete
  // record set remains available for filtering and export; only the visible
  // table rows are capped to avoid creating thousands of DOM nodes at once.
  const visibleRecords=records.slice(0,250),rowHTML=visibleRecords.map(s=>{const key=String(s.frequencyHz),expanded=expandedMapperFrequencies.has(key),verified=mapperFullyIdentified(s),sendBlocked=!!m.config?.uploadVerifiedOnly&&!verified,sources=[...(s.jobIDs||[]).map(id=>jobNames.get(id)||id),...(s.deviceIDs||[]).map(id=>deviceNames.get(id)||id)];return `<tr class="mapper-result-row ${expanded?'expanded':''}" data-mapper-frequency="${key}"><td><button class="mapper-frequency-button" type="button" aria-expanded="${expanded}" title="Show identification and activity details"><i>›</i>${formatFrequency(s.frequencyHz)}</button></td><td>${escapeHTML(s.name||s.protocolName||'Unidentified')}${verified?'<small class="verified-identity">✓ Successfully identified</small>':s.location?.label?`<small>${escapeHTML(s.location.label)}</small>`:''}</td><td>${escapeHTML(s.protocolName||s.modulation||'Unknown')}</td><td>${s.hits} / ${s.checks} · ${(100*(s.occupancy||0)).toFixed(1)}%</td><td><span class="mapper-result-source">${escapeHTML(sources.join(' · ')||'Legacy Mapper')}</span></td><td>${timeAgo(s.lastSeen)}</td><td><button class="mapper-send-one" data-frequency-hz="${s.frequencyHz}" ${sendBlocked?'disabled':''} title="${sendBlocked?'Requires a nearby authoritative match or valid decoder frames':'Add this observation to the spreadsheet Additions Queue'}">Queue</button></td></tr><tr class="mapper-detail-row ${expanded?'':'hidden'}" data-mapper-detail="${key}"><td colspan="7">${mapperDetailHTML(s)}</td></tr>`;}).join('');
  const moreRows=records.length>visibleRecords.length?`<tr class="mapper-results-more"><td colspan="7">Showing ${visibleRecords.length.toLocaleString()} of ${records.length.toLocaleString()} results · narrow the filters to view more</td></tr>`:'';
  body.innerHTML=rowHTML+moreRows||'<tr><td colspan="7">No mapped activity matches these filters</td></tr>';
  $('#mapper-count').textContent=allRecords.length?(records.length===allRecords.length?`${allRecords.length} active frequencies`:`${records.length} of ${allRecords.length} frequencies`):'No detected activity';const activeJobs=jobs.filter(job=>job.state==='running'||job.state==='stopping');$('#mapper-state').textContent=activeJobs.length?`${activeJobs.length} active`:'Idle';$('#mapper-state').className=`chip ${activeJobs.length?'ready':''}`;$('#mapper-start-button').disabled=!connected.length;$('#mapper-stop-button').disabled=!activeJobs.length;
	if(!$('#mapper-form').contains(document.activeElement))$('#mapper-job-grid').innerHTML=jobs.map(mapperJobHTML).join('')||'<div class="empty-state compact">Create one job per SDR. Each receiver can run its own range and workflow.</div>';
  if(m.config&&!$('#mapper-sheet-form').contains(document.activeElement)){ $('#mapper-sheet-url').value=m.config.sheetURL||''; $('#mapper-webhook').value=m.config.webhookURL||''; $('#mapper-contributor').value=m.config.contributor||'GP-SDR'; $('#mapper-secret').value=m.config.secret||''; $('#mapper-auto-upload').checked=!!m.config.autoUpload;$('#mapper-upload-verified').checked=!!m.config.uploadVerifiedOnly;$('#mapper-upload-state').textContent=m.lastError?'Error':m.config.autoUpload?'Automatic':m.config.webhookURL?'Ready':'Off';$('#mapper-upload-detail').textContent=m.lastError||`Additions Queue · ${m.uploadedRows||0} rows sent · ${m.verifiedRecords||0} fully identified${m.lastUpload?' · '+timeAgo(m.lastUpload):''}`;}
  setMapperResultsCollapsed(mapperResultsCollapsed,false);updateMapperWorkflow();renderMapperProgress();renderMapperRF();
}

function mapperWorkflowLabel(mode){return mode==='decipher'?'Identify':mode==='discovery'?'Discovery':'Map';}
function mapperWorkflowValue(){return $('#mapper-workflow input:checked')?.value||'adaptive';}
function setMapperWorkflow(value){const normalized=['adaptive','discovery','decipher'].includes(value)?value:'adaptive',input=$(`#mapper-workflow input[value="${normalized}"]`);if(input)input.checked=true;}
function mapperBatchReadout(progress={}){const frequencies=progress.currentFrequenciesHz||[];if(!frequencies.length)return progress.currentFrequencyHz?formatFrequency(progress.currentFrequencyHz):'—';if(frequencies.length===1)return formatFrequency(frequencies[0]);return `${formatFrequency(frequencies[0])} +${frequencies.length-1}`;}
function mapperTuningLabel(tuning={}){if(!tuning.mode)return 'RF settings pending';const parts=[tuning.mode==='auto'?'Auto RF':tuning.mode==='saved'?'Saved RF':'Manual RF'];if(Number.isFinite(tuning.lnaGainDB))parts.push(`LNA ${tuning.lnaGainDB}`);if(Number.isFinite(tuning.vgaGainDB))parts.push(`VGA ${tuning.vgaGainDB}`);parts.push(`amp ${tuning.ampEnabled?'on':'off'}`);if(tuning.noiseMarginDB)parts.push(`${Number(tuning.noiseMarginDB).toFixed(1)} dB threshold`);if(tuning.decision)parts.push(tuning.decision);return parts.join(' · ');}
function mapperJobHTML(job){const p=job.progress||{},total=Number(p.totalTargets)||0,index=Number.isInteger(p.currentIndex)?p.currentIndex:-1,totalBatches=Number(p.totalBatches)||0,batch=Number.isInteger(p.currentBatch)?p.currentBatch:-1,percent=totalBatches&&batch>=0?Math.min(100,Math.max(0,((batch+1)/totalBatches)*100)):total&&index>=0?Math.min(100,Math.max(0,((index+1)/total)*100)):0,running=job.state==='running'||job.state==='stopping',device=state.devices.find(item=>item.id===job.config.deviceID),eta=p.estimatedPassEndAt?compactDuration(Math.max(0,(new Date(p.estimatedPassEndAt)-Date.now())/1000)):'Calculating',channels=p.monitoredChannels||job.config.concurrentChannels||(job.config.mode==='decipher'?4:16),range=job.config.mode==='decipher'?' · found frequencies':` · ${(job.config.startHz/1e6).toFixed(3)}–${(job.config.endHz/1e6).toFixed(3)} MHz`,decoder=job.config.preferredDecoder&&job.config.preferredDecoder!=='auto'?job.config.preferredDecoder.toUpperCase():'Auto decoders';return `<article class="mapper-job-card ${escapeHTML(job.state)}" data-job-id="${escapeHTML(job.id)}"><div class="mapper-job-head"><h3>${escapeHTML(job.name)}</h3><span class="chip ${running?'ready':job.state==='error'?'warning':''}">${escapeHTML(job.state||'idle')}</span></div><p>${escapeHTML(device?.name||job.config.deviceID)} · ${mapperWorkflowLabel(job.config.mode)}${range} · up to ${channels} at once · ${escapeHTML(decoder)}</p><strong class="mapper-job-frequency">${mapperBatchReadout(p)}</strong><p title="Current receiver gain and overload decision">${escapeHTML(mapperTuningLabel(p.tuning))}</p><div class="mapper-job-progress"><i style="width:${percent}%"></i></div><div class="mapper-job-stats"><span>${Number(p.checksCompleted||0).toLocaleString()} checks</span><span>${p.passesCompleted||0} passes</span><span>${running?`ETA ${eta}`:'Ready'}</span></div>${job.lastError?`<p class="mapper-job-error">${escapeHTML(job.lastError)}</p>`:''}<div class="mapper-job-actions"><button data-job-action="${running?'stop':'start'}">${running?'Stop':'Start'}</button><button data-job-action="edit" ${running?'disabled':''}>Edit</button><button data-job-action="duplicate">Duplicate</button><button data-job-action="export">Export</button><button data-job-action="delete" ${running?'disabled':''}>Delete</button></div></article>`;}

function mapperPeakHours(hourly=[]){
  return hourly.map((count,hour)=>({hour,count:Number(count)||0})).filter(item=>item.count>0).sort((a,b)=>b.count-a.count||a.hour-b.hour).slice(0,3).map(item=>`${String(item.hour).padStart(2,'0')}:00–${String(item.hour).padStart(2,'0')}:59 (${item.count})`).join(' · ')||'Not enough activity yet';
}

function mapperDetailHTML(record){
  const identity=record.name||record.protocolName||'Not identified';
  const source=record.identificationSource||'No RadioReference, local database, or saved-profile match';
  const callsigns=(record.callsigns||[]).join(', ')||'None decoded';
  const location=record.location?(record.location.label||`${Number(record.location.latitude).toFixed(3)}, ${Number(record.location.longitude).toFixed(3)}`):'Not recorded';
  const analysis=record.analysisSummary||(record.identificationSource?`Matched ${record.identificationSource}. The receiver classified this as ${record.protocolName||record.modulation||'an unknown signal'} with ${Math.round((record.confidence||0)*100)}% confidence.`:`Observed RF activity, but no authoritative database match is available yet. Modulation and identity remain evidence-based estimates.`);
	const analysisEvidence=(record.analysisEvidence||[]).join(' · ')||(record.analysisEngine?`Engine: ${record.analysisEngine}`:'No local waveform evidence saved yet');
  const decoder=record.candidateDecoder?`${record.candidateDecoder} · ${record.detectionStatus||'candidate'}`:'No protocol decoder selected';
  const decoderDetail=record.candidateDecoder?(record.decoderReady?'Decoder installed; awaiting a valid frame or message':'Decoder setup is required before protocol confirmation'):'No decoder target matched this frequency';
  const verification=mapperFullyIdentified(record)?`Verified · ${record.verificationReason||'authoritative evidence'}`:'Not fully identified';
  const distance=record.referenceDistanceMiles==null?'':` · ${Number(record.referenceDistanceMiles).toFixed(1)} mi from reference area`;
  return `<div class="mapper-detail-grid"><div><span>Identified as</span><strong>${escapeHTML(identity)}</strong><small>${escapeHTML(source)}</small></div><div><span>Verification</span><strong>${escapeHTML(verification)}</strong><small>${escapeHTML((record.identificationSource||'No authoritative match')+distance)}</small></div><div><span>Peak activity hours</span><strong>${escapeHTML(mapperPeakHours(record.hourlyHits))}</strong><small>${escapeHTML(record.activityTimeZone||'Receiver local time')}</small></div><div><span>Signal evidence</span><strong>${record.hits} ${record.hits===1?'hit':'hits'} · ${(100*(record.occupancy||0)).toFixed(1)}% successful checks</strong><small>Discovery ${record.discoveryHits||0}/${record.discoveryChecks||0} · Identify ${record.identifyHits||0}/${record.identifyChecks||0} · ${Number(record.strongestDBFS).toFixed(1)} dBFS peak</small></div><div><span>Classification</span><strong>${escapeHTML(record.protocolName||record.modulation||'Unknown')}</strong><small>${escapeHTML(record.modulation||'Unknown modulation')} · ${Math.round((record.confidence||0)*100)}% confidence</small></div><div><span>Decoder evidence</span><strong>${escapeHTML(decoder)}</strong><small>${escapeHTML(record.detectionEvidence||decoderDetail)}</small></div><div><span>Callsigns</span><strong>${escapeHTML(callsigns)}</strong><small>Decoded or transcript-derived identifiers</small></div><div><span>Observed</span><strong>${timeAgo(record.lastSeen)==='now'?'Now':`${timeAgo(record.lastSeen)} ago`}</strong><small>${escapeHTML(location)}</small></div><div class="mapper-detail-wide"><span>Local automatic analysis</span><strong>${escapeHTML(analysis)}</strong><small>${escapeHTML(analysisEvidence)}</small></div><div class="mapper-detail-wide"><span>Offline transcription</span><strong>${record.lastTranscript?escapeHTML(record.lastTranscript):'No speech transcript is available yet.'}</strong><small>${record.lastTranscript?'Processed locally; no API key or cloud service used.':'Enable Dictate voice in Identify mode and install the offline model.'}</small></div></div>`;
}

function setMapperListenInput(seconds){
  const options=[86400,3600,60,1],unit=options.find(value=>seconds>=value&&seconds%value===0)||1;
  $('#mapper-listen-unit').value=String(unit); $('#mapper-listen-value').value=Math.max(1,seconds/unit);
}

function setMapperDwellInput(milliseconds){
  const bounded=Math.max(100,Math.min(7*86400000,Number(milliseconds)||500)),options=[86400000,3600000,60000,1000],unit=options.find(value=>bounded>=value&&bounded%value===0)||1000;
  $('#mapper-dwell-unit').value=String(unit);$('#mapper-dwell-value').value=Math.max(.1,bounded/unit);
}

function mapperDwellMilliseconds(){return Math.round(Number($('#mapper-dwell-value').value)*Number($('#mapper-dwell-unit').value));}

function mapperActiveJobs(){return (state.mapper?.jobs||[]).filter(job=>job.progress?.running);}
function mapperSpectrumJob(){const active=mapperActiveJobs(),snapshot=state.spectrum;if(!active.length)return null;if(!snapshot)return active[0];return active.find(job=>(job.progress?.currentFrequenciesHz||[]).some(frequency=>frequency>=snapshot.startFrequencyHz&&frequency<=snapshot.endFrequencyHz))||active[0];}

function setMapperResultsCollapsed(collapsed,persist=true){mapperResultsCollapsed=!!collapsed;const panel=$('#mapper-results-panel'),content=$('#mapper-results-content'),button=$('#mapper-results-toggle');if(!panel||!content||!button)return;panel.classList.toggle('collapsed',mapperResultsCollapsed);content.hidden=mapperResultsCollapsed;button.textContent=mapperResultsCollapsed?'Expand':'Collapse';button.setAttribute('aria-expanded',String(!mapperResultsCollapsed));if(persist)localStorage.setItem('gpsdr-mapper-results-collapsed',String(mapperResultsCollapsed));}

function renderMapperRF(){const snapshot=state.spectrum,job=mapperSpectrumJob(),running=mapperActiveJobs().length>0,badge=$('#mapper-rf-state');if(!badge)return;badge.textContent=running?'Sampling':snapshot?.capturedAt?'Last capture':'Waiting';badge.className=`chip ${running?'ready':''}`;if(snapshot?.binsDBFS?.length){const start=formatFrequency(snapshot.startFrequencyHz),center=formatFrequency(snapshot.centerFrequencyHz),end=formatFrequency(snapshot.endFrequencyHz),age=timeAgo(snapshot.capturedAt);for(const prefix of ['mapper-spectrum','mapper-waterfall']){$('#'+prefix+'-start').textContent=start;$('#'+prefix+'-mid').textContent=center;$('#'+prefix+'-end').textContent=end;}const receiver=state.devices.find(device=>device.id===job?.config?.deviceID);$('#mapper-spectrum-caption').textContent=`${(snapshot.sampleRateHz/1e6).toFixed(2)} MS/s · ${snapshot.binsDBFS.length} bins · ${age==='now'?'live':`${age} old`}`;$('#mapper-rf-detail').textContent=job?`${job.name} · ${receiver?.name||job.config.deviceID} · highlighted lines are channels in this batch`:'Most recent sampled receiver bandwidth';}else{$('#mapper-spectrum-caption').textContent='No capture yet';$('#mapper-rf-detail').textContent=running?'Waiting for the first IQ frame':'Spectrum and waterfall update while Mapper is sampling';}drawSpectrumCanvas($('#mapper-spectrum'));drawWaterfallCanvas($('#mapper-waterfall'));}

function renderMapperProgress(){
  const jobs=state.mapper?.jobs||[],active=jobs.filter(job=>job.progress?.running),selected=mapperSpectrumJob()||active[0]||jobs[0],p=selected?.progress||state.mapperProgress||{},records=state.mapper?.records||[],running=active.length>0;
  const total=Number(p.totalTargets)||0,index=Number.isInteger(p.currentIndex)?p.currentIndex:-1,totalBatches=Number(p.totalBatches)||0,batch=Number.isInteger(p.currentBatch)?p.currentBatch:-1;
  const percent=totalBatches&&batch>=0?Math.min(100,Math.max(0,((batch+1)/totalBatches)*100)):total&&index>=0?Math.min(100,Math.max(0,((index+1)/total)*100)):0;
  $('#mapper-current-frequency').textContent=mapperBatchReadout(p);
  $('#mapper-current-detail').textContent=p.currentFrequencyHz?`${p.currentLabel||'Scanning'} · ${p.monitoredChannels||1} at once · batch ${Math.max(1,batch+1)} of ${totalBatches||total}`:(running?'Preparing receiver':'Waiting to start');
  const workflow=mapperWorkflowLabel(p.mode||selected?.config?.mode),receiver=state.devices.find(device=>device.id===selected?.config?.deviceID),frequencies=p.currentFrequenciesHz||[];
  $('#mapper-operation').textContent=running?workflow:selected?.state==='error'?'Error':'Idle';
  $('#mapper-operation-detail').textContent=running?`${selected?.name||'Mapper job'} · pass ${(p.passesCompleted||0)+1} · batch ${Math.max(1,batch+1)} of ${totalBatches||'—'}`:selected?.lastError||'Create or start a Mapper job';
  const snapshot=state.spectrum,tuning=p.tuning||{};$('#mapper-capture').textContent=snapshot?.sampleRateHz?`${(snapshot.sampleRateHz/1e6).toFixed(2)} MS/s`:'—';$('#mapper-capture-detail').textContent=snapshot?.centerFrequencyHz?`${receiver?.name||selected?.config?.deviceID||'Receiver'} · centered ${formatFrequency(snapshot.centerFrequencyHz)} · ${mapperTuningLabel(tuning)}`:'No sampled band yet';
  $('#mapper-channel-list').innerHTML=frequencies.length?frequencies.map((frequency,index)=>`<span title="Software VFO ${index+1} of ${frequencies.length}">${formatFrequency(frequency)}</span>`).join(''):`<small>${running?'Preparing the next receiver batch':'Waiting for a receiver batch'}</small>`;
  $('#mapper-pass-progress').textContent=String(active.length);
  $('#mapper-pass-count').textContent=`${new Set(active.map(job=>job.config.deviceID)).size} receivers assigned`;
  const checks=jobs.reduce((sum,job)=>sum+Number(job.progress?.checksCompleted||0),0),passes=jobs.reduce((sum,job)=>sum+Number(job.progress?.passesCompleted||0),0);$('#mapper-checks').textContent=checks.toLocaleString();
  $('#mapper-passes').textContent=`${passes} completed ${passes===1?'pass':'passes'}`;
  const hits=records.reduce((sum,item)=>sum+(Number(item.hits)||0),0);
  $('#mapper-hits').textContent=hits.toLocaleString();
  const identified=records.filter(mapperFullyIdentified).length;$('#mapper-identified').textContent=identified.toLocaleString();$('#mapper-identified-detail').textContent=`${records.length?Math.round(identified/records.length*100):0}% of active frequencies`;
  const activityAge=p.lastActivityAt?timeAgo(p.lastActivityAt):'';$('#mapper-last-activity').textContent=activityAge?(activityAge==='now'?'Activity now':`Last activity ${activityAge} ago`):'No activity yet';
  $('#mapper-elapsed').textContent=durationBetween(p.startedAt,running?null:p.stoppedAt);
	const eta=p.estimatedPassEndAt?new Date(p.estimatedPassEndAt):null,remaining=eta?Math.max(0,(eta.getTime()-Date.now())/1000):0;
	$('#mapper-eta').textContent=running&&eta?compactDuration(remaining):running?'Calculating':'—';
	$('#mapper-eta-time').textContent=running&&eta?`Estimated ${eta.toLocaleTimeString([], {hour:'numeric',minute:'2-digit'})}`:running?'Available after the first channel':'No active pass';
  if(running&&p.mode==='decipher'&&p.targetEndsAt){const remaining=Math.max(0,(new Date(p.targetEndsAt).getTime()-Date.now())/1000);$('#mapper-channel-time').textContent=`Next channel in ${compactDuration(remaining)}`;}
  else if(running){const checkAge=p.lastCheckAt?timeAgo(p.lastCheckAt):'';$('#mapper-channel-time').textContent=checkAge?(checkAge==='now'?'Checking now':`Last check ${checkAge} ago`):'Starting first check';}
  else{$('#mapper-channel-time').textContent=p.stoppedAt?'Session stopped':'—';}
  const track=$('.mapper-progress-track');track.setAttribute('aria-valuenow',String(Math.round(percent)));$('#mapper-progress-bar').style.width=`${percent}%`;
  renderMapperRF();
}

function updateMapperWorkflow(){const workflow=mapperWorkflowValue(),identify=workflow==='decipher',adaptive=workflow==='adaptive',detail=$('#mapper-workflow-detail');$$('.mapper-discovery-control').forEach(control=>control.classList.toggle('hidden',identify));$$('.mapper-identify-control').forEach(control=>control.classList.toggle('hidden',!identify));$('#mapper-step').disabled=identify;$('#mapper-dwell-value').disabled=identify;$('#mapper-dwell-unit').disabled=identify;$('#mapper-listen-wrap').classList.toggle('hidden',!identify);$('#mapper-transcribe-wrap').classList.toggle('hidden',!(identify||adaptive));$('#mapper-start-caption').textContent=identify?'Filter start · MHz':'Start · MHz';$('#mapper-end-caption').textContent=identify?'Filter end · MHz':'End · MHz';$('#mapper-job-name').placeholder=identify?'Known-channel identification':adaptive?'General-purpose map':'UHF discovery';$('#mapper-start-button').textContent=identify?'Start identify':adaptive?'Start mapping':'Start discovery';detail.classList.toggle('identify',identify);detail.innerHTML=identify?'<strong>Identify</strong> revisits only frequencies meeting the selected hit, successful-check, age, and range rules. New Identify activity still increments the combined history; Discovery-only eligibility prevents one-off hits from promoting themselves.':adaptive?'<strong>Map</strong> repeatedly sweeps the selected range, detects activity, classifies and decodes it locally, records speech when enabled, and manages IQ evidence after analysis.':'<strong>Discovery</strong> checks multiple frequency steps from each IQ capture and records active frequencies. Auto uses up to sixteen channels at once.';updateMapperGainControls();}

function updateMapperGainControls(){const manual=$('#mapper-gain-mode').value==='manual',manualMargin=$('#mapper-sensitivity').value==='manual';$$('.mapper-manual-gain').forEach(control=>control.classList.toggle('hidden',!manual));$$('.mapper-manual-margin').forEach(control=>control.classList.toggle('hidden',!manualMargin));}

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
  const health=$('#health-notices'), notices=status.healthNotices||[];
  health.textContent=notices.map(item=>item.message).join(' ');
  health.classList.toggle('hidden',!notices.length);
  health.classList.toggle('error',notices.some(item=>item.level==='warning'));
  const storage=status.storage||{};
  $('#storage-total').textContent=formatBytes(storage.totalBytes);
  $('#storage-recordings').textContent=formatBytes(storage.recordingBytes);
  $('#storage-iq').textContent=formatBytes(storage.iqBytes);
  $('#storage-iq-pending').textContent=formatBytes(storage.iqPendingBytes);
  $('#storage-iq-retained').textContent=formatBytes(storage.iqRetainedBytes);
  $('#storage-iq-quarantine').textContent=formatBytes(storage.iqQuarantineBytes);
  $('#storage-journal').textContent=formatBytes((storage.journalBytes||0)+(storage.profileBytes||0));
  const storageForm=$('#storage-policy-form'),policy=storage.policy||{};if(storageForm&&!storageForm.contains(document.activeElement)){const gb=1024**3;$('#storage-auto-cleanup').checked=!!policy.autoCleanup;$('#storage-auto-remove-rejected').checked=policy.autoRemoveQuarantine!==false;$('#storage-rejected-hours').value=String(policy.quarantineRetentionHours||24);$('#storage-max-days').value=String(policy.maxCaptureDays??30);$('#storage-recording-cap').value=String(Math.round(Number(policy.recordingCapBytes||0)/gb));$('#storage-iq-cap').value=String(Math.round(Number(policy.iqCapBytes||0)/gb));}
  $('#storage-clean-now').disabled=!!storage.cleanupRunning;
  const cleanup=storage.lastCleanup||{},cleanupText=cleanup.completedAt?` · last cleanup ${timeAgo(cleanup.completedAt)} ago${cleanup.bytesFreed?` · freed ${formatBytes(cleanup.bytesFreed)}`:''}`:'';
  $('#storage-checked').textContent=storage.checkedAt?`Measured ${timeAgo(storage.checkedAt)} ago · GP-SDR-owned folders only${cleanupText}`:'Calculating local storage…';
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
    ${event.analysis?`<div class="latest-analysis">${escapeHTML(event.analysis.summary||'Local signal analysis complete')}</div>`:''}
    <div class="latest-transcript">${escapeHTML(event.transcript || 'No transcript')}${event.callsigns?.length?` · ${escapeHTML(event.callsigns.join(', '))}`:''}</div>
    ${event.audioPath ? `<button class="play-event" data-event-id="${event.id}" title="Play or pause this recording">▶ Recording</button>` : ''}`;
}

function renderMixer() {
  const root = $('#mixer-list');
  if (!state.mixer.length) { root.className = 'mixer-list empty-state compact'; root.textContent = 'Start a channel profile'; return; }
  const query = ($('#mixer-search')?.value || '').trim().toLowerCase();
  const filtered = state.mixer.filter(item => {
    if (!query) return true;
    const detail = item.talkgroupID ? `tg ${item.talkgroupID} ${item.systemName || 'p25'}` : formatFrequency(item.channel.frequencyHz);
    return [item.channel.name, detail, item.channel.mode, item.channel.decoder].some(value => String(value || '').toLowerCase().includes(query));
  }).sort((left, right) => {
    if (state.mixerOrder === 'recent') return new Date(right.lastHeardAt || 0) - new Date(left.lastHeardAt || 0) || (right.eventCount || 0) - (left.eventCount || 0);
    if (state.mixerOrder === 'heard') return (right.eventCount || 0) - (left.eventCount || 0) || Number(right.active) - Number(left.active);
    if (state.mixerOrder === 'frequency') return (left.channel.frequencyHz || 0) - (right.channel.frequencyHz || 0);
    if (state.mixerOrder === 'name') return String(left.channel.name || '').localeCompare(String(right.channel.name || ''));
    return Number(right.active) - Number(left.active) || new Date(right.lastHeardAt || 0) - new Date(left.lastHeardAt || 0) || (right.eventCount || 0) - (left.eventCount || 0);
  });
  root.className = 'mixer-list';
  root.innerHTML = filtered.length ? mixerRows(filtered) : '<div class="empty-state compact">No matching channels</div>';
  applyMixerGains();
}

function mixerRows(items) {
  return items.map(item => {
    const activity = item.talkgroupID ? `${item.eventCount || 0} received${item.lastHeardAt ? ` · ${timeAgo(item.lastHeardAt)} ago` : ''}` : '';
    const detail = item.talkgroupID ? `TG ${item.talkgroupID} · ${item.systemName || 'P25'}${item.encrypted ? ' · encrypted' : ''}${activity ? ` · ${activity}` : ''}` : `${shortFrequency(item.channel.frequencyHz)} MHz`;
    return `
    <div class="mixer-row ${item.active ? 'active' : ''}" data-mixer-id="${item.id}">
      <div class="channel-name" title="${escapeHTML(item.channel.name)} · ${escapeHTML(detail)}"><strong>${escapeHTML(item.channel.name)}${item.discovered ? ' <span class="discovered-mark">new</span>' : ''}</strong><small>${escapeHTML(detail)}</small></div>
      <div class="level-meter" title="Current audio level"><i style="width:${Math.round(item.level * 100)}%"></i></div>
      <button class="mini-toggle mixer-mute ${item.muted ? 'on' : ''}" title="Mute this ${item.talkgroupID ? 'talkgroup' : 'channel'}">M</button>
      <button class="mini-toggle mixer-solo ${item.solo ? 'on' : ''}" title="Hear only this ${item.talkgroupID ? 'talkgroup' : 'channel'}">S</button>
      ${item.talkgroupID?'<span class="p25-native-audio" title="P25 audio uses the selected system output; use mute or solo per talkgroup">P25</span>':`<span class="mixer-sliders"><input class="mixer-volume" type="range" min="0" max="1" step="0.05" value="${item.volume}" title="Channel volume" aria-label="Volume for ${escapeHTML(item.channel.name)}"><input class="mixer-pan advanced-only" type="range" min="-1" max="1" step="0.1" value="${item.pan||0}" title="Stereo pan" aria-label="Stereo pan for ${escapeHTML(item.channel.name)}"></span>`}
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
    <td class="frequency">${formatFrequency(event.frequencyHz)}</td><td>${escapeHTML(event.protocolName || event.modulation)}${event.talkgroupID?`<small>TG ${event.talkgroupID}${event.sourceRadioID?` · Radio ${event.sourceRadioID}`:''}${event.encrypted?' · Encrypted':''}</small>`:''}${event.ctcssHz?`<small>CTCSS ${Number(event.ctcssHz).toFixed(1)} Hz</small>`:''}</td>
    <td>${event.signalDBFS.toFixed(0)} dBFS</td><td>${event.durationSeconds.toFixed(1)}s</td>
    <td class="transcript" title="${escapeHTML(event.transcript || '')}">${escapeHTML(event.transcript || '—')}</td>
    <td>${event.audioPath ? `<button class="row-action play-event" data-event-id="${event.id}" title="Play or pause this recording">▶</button>` : ''}</td></tr>`).join('')
    : '<tr><td colspan="7" class="empty-state compact">No events logged</td></tr>';
}

let eventSearchTimer;
async function searchEvents() {
  clearTimeout(eventSearchTimer);
  eventSearchTimer = setTimeout(async()=>{
    try {
      const query=$('#event-search').value.trim();
      state.events=await api(`/api/events?limit=500${query?`&q=${encodeURIComponent(query)}`:''}`);
      renderEvents();
    } catch(error) { toast(error.message,true); }
  },180);
}

function renderProfiles() {
  $('#profile-grid').innerHTML = state.profiles.length ? state.profiles.map(profile => `
    <article class="profile-card ${profile.id === state.selectedProfileID ? 'selected' : ''}" data-profile-id="${profile.id}">
      <div class="card-top"><div><h3>${escapeHTML(profile.name)}</h3><p>${escapeHTML(profile.summary || 'Custom scan configuration')}</p></div>${profile.builtIn ? '<span class="chip">Built-in</span>' : ''}</div>
      <div class="profile-stats"><div><span>Ranges</span><strong>${profile.ranges.length}</strong></div><div><span>Channels</span><strong>${profile.channels.length}</strong></div><div><span>P25</span><strong>${(profile.p25Systems || []).length}</strong></div><div><span>Receivers</span><strong>${profile.deviceAssignments.length}</strong></div></div>
      <div class="card-actions">
        <button class="select-profile" title="Use this profile">Use</button>
        <button class="${profile.builtIn ? 'duplicate-profile' : 'edit-profile'}" title="${profile.builtIn ? 'Make an editable copy' : 'Edit this profile'}">${profile.builtIn ? 'Duplicate' : 'Edit'}</button>
        <button class="export-profile" title="Download this profile for sharing">Export</button>
      </div>
    </article>`).join('') : '<div class="empty-state">No profiles yet. Create one or import a shared channel bank.</div>';
}

function renderHardware() {
  $('#device-grid').innerHTML = state.devices.length ? state.devices.map(device => `
    <article class="hardware-card"><div class="hardware-title"><i class="${device.connected ? 'ready' : device.available ? 'optional' : ''}"></i><h3>${escapeHTML(device.name)}</h3></div>
		<p>${escapeHTML(hardwareActivityText(device))}</p>
      <div class="hardware-detail">${device.kind === 'HackRF' ? 'LNA 0–40 dB · VGA 0–62 dB · RF amp · antenna power · 2–20 MS/s' : device.kind === 'RTL-SDR' ? 'Tuner AGC/manual gain · PPM correction · 0.225–3.2 MS/s' : 'SoapySDR gain · PPM and device-specific controls'}<br>${escapeHTML(hardwareRangeText(device))}<br>${escapeHTML(device.driver)}${device.serial ? ` · ${escapeHTML(device.serial)}` : ''}${device.helperArchitecture ? ` · ${escapeHTML(device.helperArchitecture)}` : ''}</div>
      ${hardwareTelemetryHTML(device)}
      <footer><span>${escapeHTML(device.kind)}</span><span>${device.connected ? 'Connected' : device.available ? 'Driver ready' : 'Driver needed'}</span></footer>
      ${device.connected ? calibrationControls(device) : ''}
      ${device.connected || device.available ? '' : setupActions(device.kind === 'HackRF' ? 'hackrf' : device.kind === 'RTL-SDR' ? 'rtlsdr' : 'soapysdr')}</article>`).join('') : '<div class="empty-state">No receiver drivers were detected. Install one below, then refresh.</div>';
  $('#decoder-grid').innerHTML = state.decoders.length ? state.decoders.map(decoder => `
    <article class="hardware-card"><div class="hardware-title"><i class="${decoder.state}"></i><h3>${escapeHTML(decoder.name)}</h3></div>
      <p>${escapeHTML(decoder.note)}</p><footer><span>${escapeHTML(decoder.standards.join(' · '))}</span><span>${decoder.state === 'ready' ? 'Ready' : 'Optional'}</span></footer>
      <div class="card-actions decoder-card-actions"><button class="open-decoder" data-decoder-id="${escapeHTML(decoder.id)}" title="Open the ${escapeHTML(decoder.name)} workspace">Open</button></div>
      ${decoder.state === 'ready' || decoder.id === 'analog' ? '' : setupActions(decoder.id)}</article>`).join('') : '<div class="empty-state">Decoder status is unavailable.</div>';
  renderSetupJob();
  const remoteList=$('#remote-list'); if(remoteList) remoteList.innerHTML=state.remoteReceivers.map(item=>`<div class="remote-row"><span><strong>${escapeHTML(item.name)}</strong><small>${escapeHTML(item.host)}:${item.port}</small></span><button class="remove-remote" data-remote-id="${escapeHTML(item.id)}" title="Remove this remote receiver">Remove</button></div>`).join('')||'<span class="empty-state compact">No remote receivers saved</span>';
}

function hardwareRangeText(device){return device.frequencyMinimumHz&&device.frequencyMaximumHz?`${formatFrequency(device.frequencyMinimumHz)}–${formatFrequency(device.frequencyMaximumHz)} nominal · ${device.frequencyRangeNote||'model-dependent range'}`:'Frequency range reported by the installed driver';}

function hardwareActivityText(device) {
  const mapperJob=(state.mapper?.jobs||[]).find(job=>job.config?.deviceID===device.id&&(job.state==='running'||job.state==='stopping'));
  if(mapperJob)return `Mapper · ${mapperJob.name} · ${mapperJob.progress?.mode||mapperJob.config?.mode||'active'}`;
  if(device.connected&&state.status?.running)return `Streaming · ${state.status.mode}`;
  return device.note||'Connected and ready for assignment.';
}

function hardwareTelemetryHTML(device) {
  const telemetry=state.status?.receiverTelemetry;
  if(!telemetry||telemetry.deviceID!==device.id)return '';
  const snr=Number(telemetry.signalDBFS)-Number(telemetry.noiseDBFS), stateText=telemetry.overloaded?'Overloaded':telemetry.signalDetected?'Signal':'Noise only';
  return `<div class="hardware-live"><span>${escapeHTML(stateText)}</span><strong>${Number(telemetry.signalDBFS).toFixed(1)} dBFS</strong><small>SNR ${snr.toFixed(1)} dB · noise ${Number(telemetry.noiseDBFS).toFixed(1)} · ${Number(telemetry.sampleRateHz/1e6).toFixed(1)} MS/s${state.status.droppedSamples?` · ${state.status.droppedSamples} drops`:''}</small></div>`;
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

function updateCharacterizationRangeControls(){const mode=$('#characterization-range-mode')?.value||'receiver';$$('.characterization-antenna-range').forEach(item=>item.classList.toggle('hidden',mode!=='antenna'));$$('.characterization-custom-range').forEach(item=>item.classList.toggle('hidden',mode!=='custom'));}
function updateCharacterizationGainControls(){const saved=$('#characterization-use-calibration')?.checked;for(const id of ['characterization-gain','characterization-lna','characterization-vga','characterization-amp']){const control=$('#'+id);if(control)control.disabled=!!saved||!!state.characterization?.running;}}

function renderCharacterization(){
  const form=$('#characterization-form');if(!form)return;const status=state.characterization||{},results=status.results||[],connected=state.devices.filter(device=>device.connected&&device.available&&device.kind!=='Remote');
  const selected=new Set($$('#characterization-devices input:checked').map(input=>input.value));
  $('#characterization-devices').innerHTML=connected.length?connected.map((device,index)=>`<label title="${escapeHTML(device.frequencyRangeNote||'Receiver tuning range')}"><input type="checkbox" value="${escapeHTML(device.id)}" ${selected.size?selected.has(device.id)?'checked':'':index<2?'checked':''}><span><strong>${escapeHTML(device.name)}</strong><small>${device.frequencyMinimumHz&&device.frequencyMaximumHz?`${formatFrequency(device.frequencyMinimumHz)}–${formatFrequency(device.frequencyMaximumHz)}`:'Driver-defined range'}</small></span></label>`).join(''):'<span>No connected receivers</span>';
  const running=!!status.running,total=results.reduce((sum,item)=>sum+Number(item.totalPoints||0),0),complete=results.reduce((sum,item)=>sum+Number(item.completedPoints||0),0),percent=total?Math.round(complete/total*100):0;
  $('#characterization-state').textContent=running?`${percent}%`:(results.length?'Complete':'Idle');$('#characterization-state').className=`chip ${running?'ready':''}`;$('#characterization-stop').disabled=!running;$('#characterization-start-button').disabled=running||!connected.length;$('#characterization-clear').disabled=running||!results.length;$('#characterization-export').disabled=!results.some(item=>item.points?.length);
  form.querySelectorAll('input,select').forEach(control=>control.disabled=running);$('#characterization-progress-bar').style.width=`${percent}%`;
  const eta=status.expectedCompletionAt?new Date(status.expectedCompletionAt):null;$('#characterization-detail').textContent=running?`${complete} of ${total} points · ${eta?`about ${compactDuration(Math.max(0,(eta-Date.now())/1000))} remaining`:'calculating'} · receivers run in parallel`:status.lastError||status.measurementNotice||'Ambient comparison only; laboratory sensitivity requires a calibrated source.';
  $('#characterization-results').innerHTML=results.map((result,index)=>{const points=result.points||[],progress=result.totalPoints?Math.round(result.completedPoints/result.totalPoints*100):0,current=result.currentFrequencyHz?` · now ${formatFrequency(result.currentFrequencyHz)}`:'';return `<article class="characterization-result ${result.error?'error':''}"><div class="characterization-result-head"><div><h3>${escapeHTML(result.deviceName)}</h3><span>${escapeHTML(result.deviceKind)} · ${formatFrequency(result.testedMinimumHz)}–${formatFrequency(result.testedMaximumHz)}${current}</span></div><strong>${progress}%</strong></div><canvas data-characterization-chart="${index}" height="150" aria-label="Observed response across tested frequencies"></canvas><div class="characterization-metrics"><div><span>Best observed</span><strong>${result.bestObservedFrequencyHz?formatFrequency(result.bestObservedFrequencyHz):'—'}</strong></div><div><span>Average noise</span><strong>${points.length?Number(result.averageNoiseDBFS).toFixed(1)+' dBFS':'—'}</strong></div><div><span>Detected</span><strong>${result.detectedPoints||0} / ${result.completedPoints||0}</strong></div><div><span>Overloaded</span><strong>${result.overloadedPoints||0}</strong></div></div><p>${escapeHTML(result.error||result.recommendation||result.frequencyRangeNote||'Waiting for measurements')}</p></article>`;}).join('')||'<div class="empty-state compact">Select both receivers, choose a full, antenna, or custom range, then start a comparison.</div>';
  updateCharacterizationRangeControls();updateCharacterizationGainControls();requestAnimationFrame(()=>results.forEach((result,index)=>drawCharacterizationChart($(`[data-characterization-chart="${index}"]`),result.points||[])));
}

function drawCharacterizationChart(canvas,points){if(!canvas)return;const ratio=Math.min(devicePixelRatio||1,2),width=Math.max(320,canvas.clientWidth||640),height=150;canvas.width=width*ratio;canvas.height=height*ratio;const ctx=canvas.getContext('2d');ctx.scale(ratio,ratio);ctx.fillStyle='#070b10';ctx.fillRect(0,0,width,height);ctx.strokeStyle='#202a34';ctx.lineWidth=1;for(let line=1;line<4;line++){const y=line*height/4;ctx.beginPath();ctx.moveTo(0,y);ctx.lineTo(width,y);ctx.stroke();}if(points.length<2)return;const values=points.map(point=>Number(point.relativeScore)||0),noise=points.map(point=>Number(point.noiseDBFS)||-120),minNoise=Math.min(...noise),maxNoise=Math.max(...noise, minNoise+1);ctx.strokeStyle='#667584';ctx.lineWidth=1.2;ctx.beginPath();noise.forEach((value,index)=>{const x=index/(points.length-1)*width,y=height-12-(value-minNoise)/(maxNoise-minNoise)*(height-24);index?ctx.lineTo(x,y):ctx.moveTo(x,y);});ctx.stroke();ctx.strokeStyle='#49d5aa';ctx.lineWidth=2;ctx.beginPath();values.forEach((value,index)=>{const x=index/(points.length-1)*width,y=height-12-clampNumber(value,0,100)/100*(height-24);index?ctx.lineTo(x,y):ctx.moveTo(x,y);});ctx.stroke();}
function clampNumber(value,min,max){return Math.min(max,Math.max(min,value));}

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
	const tunedChannel = state.mixer.find(item => item.id === 'quick-tune-channel')?.channel;
	if (tuning && tunedChannel) {
		if (document.activeElement !== $('#tuner-frequency') && $('#tuner-frequency').dataset.pending !== 'true') {
			const listenMHz = (tunedChannel.frequencyHz / 1e6).toFixed(6);
			$('#tuner-frequency').value = listenMHz;
			$('#tuner-readout').textContent = listenMHz;
		}
		if (document.activeElement !== $('#tuner-mode')) $('#tuner-mode').value = tunedChannel.mode;
		if (document.activeElement !== $('#live-mode')) $('#live-mode').value = tunedChannel.mode;
		if (document.activeElement !== $('#tuner-bandwidth')) $('#tuner-bandwidth').value = tunedChannel.bandwidthHz / 1e3;
		if (document.activeElement !== $('#live-bandwidth')) $('#live-bandwidth').value = tunedChannel.bandwidthHz / 1e3;
	}
	const telemetry=state.status?.receiverTelemetry, receiving=!!telemetry?.signalDetected, overloaded=!!telemetry?.overloaded;
	for (const prefix of ['live','tuner']) { const light=$('#'+prefix+'-signal-light')?.parentElement; if(light){light.classList.toggle('active',receiving);light.classList.toggle('overloaded',overloaded);} const label=$('#'+prefix+'-signal-text'); if(label) label.textContent=overloaded?`Overload ${Number(telemetry.clippedPercent).toFixed(1)}%`:receiving?'Signal':'No signal'; }
  $('#tuner-start').disabled = !connected.length;
	$('#tuner-start').textContent=tuning&&$('#tuner-lock-center').checked?'Apply VFO':tuning?'Retune':'Tune';
  $('#tuner-stop').disabled = !tuning;
  const hardwareMHz=state.spectrum?.centerFrequencyHz?(state.spectrum.centerFrequencyHz/1e6).toFixed(6):'starting',listenMHz=Number($('#tuner-frequency').value||0).toFixed(6);
	const intelligence=state.status?.signalAnalysis, analysisText=intelligence?.modulation&&intelligence.modulation!=='UNKNOWN'?` · ${intelligence.modulation} ${Math.round((intelligence.confidence||0)*100)}%`:'';
	$('#tuner-status').textContent = tuning ? `Hardware ${hardwareMHz} MHz · Listen ${listenMHz} MHz${analysisText}` : connected.length ? 'Ready to tune.' : 'Connect a receiver, then refresh Hardware.';
  const snapshot = state.spectrum;
  if (snapshot?.binsDBFS?.length) {
		const sorted=[...snapshot.binsDBFS].sort((a,b)=>a-b),noise=sorted[Math.floor(sorted.length*.25)]??-120,peak=Math.max(...snapshot.binsDBFS),snr=Math.max(0,peak-noise);
    $('#tuner-center').textContent = formatFrequency(snapshot.centerFrequencyHz);
		if(tuning&&document.activeElement!==$('#tuner-hardware-center'))$('#tuner-hardware-center').value=(snapshot.centerFrequencyHz/1e6).toFixed(6);
    $('#tuner-span').textContent = `${(snapshot.sampleRateHz / 1e6).toFixed(2)} MS/s · ${snapshot.binsDBFS.length} bins · ~${snr.toFixed(1)} dB SNR`;
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
    noiseReduction:$('#tuner-noise').value,useCalibration:$('#tuner-use-calibration').checked,lockCenter:$('#tuner-lock-center').checked,
    hardwareCenterHz:$('#tuner-lock-center').checked&&state.status?.running&&state.status?.activeProfileID==='quick-tune'?(state.spectrum?.centerFrequencyHz||0):(Number($('#tuner-hardware-center').value)*1e6||0)};
}

const decoderModeMap={digital:'dsd-fme',dmr:'dmr',p25:'dsd-fme',nxdn:'nxdn',nxdn48:'nxdn48',nxdn96:'nxdn96','d-star':'d-star',ysf:'ysf',m17:'m17',pocsag:'multimon-ng',acars:'acarsdec','ads-b':'dump1090','rtl-433':'rtl-433',ais:'ais'};
function decoderForSelectedMode(mode){return decoderModeMap[String(mode||'').toLowerCase()]||'';}
function profileModeOptions(){return '<option value="auto">Auto</option><option value="am">AM</option><option value="nfm">NFM</option><option value="wfm">WFM</option><option value="digital">Digital auto</option><option value="dmr">DMR</option><option value="p25">P25 conventional</option><option value="nxdn">NXDN</option><option value="nxdn48">NXDN48</option><option value="d-star">D-STAR</option><option value="ysf">YSF</option><option value="m17">M17</option>'}
function profileDecoderOptions(){return '<option value="">Auto / analog</option><option value="dmr">DMR · DSD-FME</option><option value="dsd-fme">Digital voice · DSD-FME</option><option value="nxdn">NXDN · DSD-FME</option><option value="d-star">D-STAR · DSD-FME</option><option value="ysf">YSF · DSD-FME</option><option value="m17">M17 · DSD-FME</option><option value="rtl-433">rtl_433 sensors</option><option value="dump1090">ADS-B / Mode S</option><option value="multimon-ng">POCSAG / FLEX</option><option value="acarsdec">ACARS</option><option value="ais">AIS</option>'}

function decoderMatchesEvent(decoder, event) {
  const identity = `${event.protocolName || ''} ${event.modulation || ''}`.toLowerCase();
  if (decoder.id === 'analog') return !event.protocolName && ['am','nfm','wfm','fm'].includes(String(event.modulation || '').toLowerCase());
  if (decoder.id === 'p25') return identity.includes('p25');
  const keys = {
    'dsd-fme':['digital voice','dmr','nxdn','d-star','ysf','m17'], 'rtl-433':['sensor','tpms','weather'], dump1090:['ads-b','mode s'],
    'multimon-ng':['paging','signaling','pocsag','flex','mdc1200','dtmf'], acarsdec:['acars'], ais:['ais']
  }[decoder.id] || decoder.standards.map(value => value.toLowerCase());
  return keys.some(key => identity.includes(key));
}

function decoderMatchesConfig(decoderID,value){value=String(value||'').toLowerCase();if(decoderID==='dsd-fme')return ['dsd-fme','dmr','nxdn','nxdn48','nxdn96','d-star','ysf','m17'].includes(value);return value===decoderID;}

function renderDecoders() {
  const nav = $('#decoder-nav'), detail = $('#decoder-detail');
  if (!nav || !detail) return;
  if (!state.decoders.some(item => item.id === state.selectedDecoderID)) state.selectedDecoderID = state.decoders[0]?.id || 'analog';
  nav.innerHTML = state.decoders.map(decoder => `<button class="decoder-nav-item ${decoder.id === state.selectedDecoderID ? 'active' : ''}" data-decoder-id="${escapeHTML(decoder.id)}" aria-pressed="${decoder.id === state.selectedDecoderID}">
    <i class="${escapeHTML(decoder.state)}"></i><span><strong>${escapeHTML(decoder.name)}</strong><small>${escapeHTML(decoder.standards.slice(0,2).join(' · '))}</small></span></button>`).join('');
  const decoder = state.decoders.find(item => item.id === state.selectedDecoderID);
  if (!decoder) { detail.innerHTML = '<div class="empty-state">No decoders found</div>'; return; }
  const events = state.events.filter(event => decoderMatchesEvent(decoder,event)).slice(0,12);
  const relevantProfiles = state.profiles.filter(profile => decoder.id === 'p25' ? (profile.p25Systems || []).length :
	profile.channels.some(channel => decoder.id === 'analog' ? ['am','nfm','wfm','fm','auto'].includes(channel.mode) : decoderMatchesConfig(decoder.id,channel.decoder)) ||
	profile.ranges.some(range => decoderMatchesConfig(decoder.id,range.decoder)));
  const setup = decoder.state === 'ready' || decoder.id === 'analog' ? '' : setupActions(decoder.id);
  const p25 = decoder.id === 'p25' ? renderP25DecoderWorkspace() : '';
  detail.innerHTML = `<article class="panel decoder-hero">
    <div><div class="decoder-heading"><i class="${escapeHTML(decoder.state)}"></i><h2>${escapeHTML(decoder.name)}</h2><span class="chip">${decoder.state === 'ready' ? 'Ready' : 'Setup'}</span></div>
	<p>${escapeHTML(decoder.note)}</p><div class="decoder-standards">${decoder.standards.map(item=>`<span>${escapeHTML(item)}</span>`).join('')}</div></div><div class="card-actions"><button class="decoder-new-profile" data-decoder-config="${escapeHTML(decoder.id)}" title="Create a channel or range configuration for this decoder">New configuration</button>${setup}</div></article>
    ${p25}
    <div class="decoder-columns">
      <article class="panel"><div class="panel-head"><div><h2>Profiles</h2><span>Configurations using this decoder</span></div></div><div class="decoder-profile-list">${relevantProfiles.length ? relevantProfiles.map(profile=>`<button class="decoder-profile" data-profile-id="${escapeHTML(profile.id)}"><span><strong>${escapeHTML(profile.name)}</strong><small>${escapeHTML(profile.summary || '')}</small></span><b>Use</b></button>`).join('') : '<div class="empty-state compact">No matching profiles</div>'}</div></article>
	  <article class="panel"><div class="panel-head"><div><h2>Recent activity</h2><span>Events identified by this decoder</span></div></div><div class="decoder-event-list">${events.length ? events.map(event=>{const message=event.decoderMessages?.[0],details=message?[message.timeSlot?`Slot ${message.timeSlot}`:'',message.colorCode?`CC ${message.colorCode}`:'',message.talkgroup?`TG ${message.talkgroup}`:'',message.sourceID?`Source ${message.sourceID}`:'',message.encrypted?'Encrypted':''].filter(Boolean).join(' · '):'';return `<div><span><strong>${escapeHTML(event.label || formatFrequency(event.frequencyHz))}</strong><small>${formatFrequency(event.frequencyHz)} · ${timeAgo(event.startedAt)}${details?` · ${escapeHTML(details)}`:''}</small></span><b>${escapeHTML(event.protocolName || event.modulation)}</b></div>`;}).join('') : '<div class="empty-state compact">No decoded activity yet</div>'}</div></article>
    </div>`;
}

function renderP25DecoderWorkspace() {
  const status = state.p25Status || {};
  const profile = state.profiles.find(item=>item.id===(status.profileID||state.status?.activeProfileID));
  const configuredRate = profile?.settings?.p25SampleRateHz || 0;
  const talkgroups = state.mixer.filter(item => item.talkgroupID).sort((left, right) => {
    if (!!left.active !== !!right.active) return left.active ? -1 : 1;
    const leftTime = left.lastHeardAt ? new Date(left.lastHeardAt).getTime() : 0;
    const rightTime = right.lastHeardAt ? new Date(right.lastHeardAt).getTime() : 0;
    if (state.p25Order === 'heard') return (right.eventCount || 0) - (left.eventCount || 0) || rightTime - leftTime || (left.talkgroupID || 0) - (right.talkgroupID || 0);
    return rightTime - leftTime || (right.eventCount || 0) - (left.eventCount || 0) || (left.talkgroupID || 0) - (right.talkgroupID || 0);
  });
  const active = talkgroups.filter(item => item.active).length;
  const connected = state.devices.filter(item=>item.connected);
  const calibrated = connected.filter(item=>item.calibration).length;
  const voiceSetup=setupComponent('p25-voice')?.state==='ready'?'':setupActions('p25-voice');
  return `<article class="panel p25-overview"><div class="p25-metrics"><div><span>Engine</span><strong>${escapeHTML(status.engine || 'Bundled')}</strong></div><div><span>Reception</span><strong>${escapeHTML(status.reception || status.state || 'setup')}</strong></div><div><span>Control channel</span><strong>${status.controlChannelHz ? formatFrequency(status.controlChannelHz) : 'Searching'}</strong>${status.controlChannelHz ? `<small>${status.controlSource === 'decoded' ? 'Decoded current' : 'Configured primary'}</small>` : ''}</div><div><span>Capture width</span><strong>${status.captureRateHz ? `${status.captureRateHz/1e6} MS/s` : 'Auto'}</strong></div><div><span>Talkgroups</span><strong>${talkgroups.length}</strong></div><div><span>Calibration</span><strong>${calibrated}/${connected.length}</strong></div></div><p class="hardware-detail">${escapeHTML(status.note || '')}${calibrated ? ' · Saved PPM, gain, and front-end calibration applied; P25 IQ tracking remains automatic.' : ' · Calibrate the receiver on the Hardware page for best results.'}</p>${voiceSetup}
    <div class="panel-head"><div><h2>Talkgroup mixer</h2><span>${active ? `${active} active · ` : ''}Mute and solo independently</span></div><div class="p25-mixer-tools"><label>Capture<select id="p25-live-rate" title="P25 receiver bandwidth; changing it restarts the active P25 profile">${p25RateOptions(configuredRate)}</select></label><label>Order<select id="p25-order" title="Order calls by latest activity or total received"><option value="recent" ${state.p25Order === 'recent' ? 'selected' : ''}>Most recent</option><option value="heard" ${state.p25Order === 'heard' ? 'selected' : ''}>Most received</option></select></label><button class="decoder-mute-all icon-button" title="Mute or unmute every P25 talkgroup">M</button></div></div>
    <div class="mixer-list p25-mixer">${talkgroups.length ? mixerRows(talkgroups) : '<div class="empty-state compact">Start a P25 profile to load talkgroups</div>'}</div></article>`;
}

function p25RateOptions(configuredRate) {
  const option = (rate, label) => `<option value="${rate}" ${configuredRate===rate?'selected':''}>${label}</option>`;
  const rtlRates = [[1024000,'1.024'],[1200000,'1.2'],[1440000,'1.44'],[1600000,'1.6'],[1800000,'1.8'],[1920000,'1.92'],[2048000,'2.048'],[2304000,'2.304'],[2400000,'2.4'],[2560000,'2.56'],[2880000,'2.88']];
  const hackRFRates = [[5000000,'5'],[8000000,'8'],[10000000,'10'],[20000000,'20']];
  return option(0, 'Auto · match receiver') + `<optgroup label="RTL-SDR">${rtlRates.map(([rate,label])=>option(rate, `${label} MS/s`)).join('')}</optgroup><optgroup label="HackRF">${hackRFRates.map(([rate,label])=>option(rate, `${label} MS/s`)).join('')}</optgroup>`;
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
	const highestActivePriority=Math.max(0,...state.mixer.filter(item=>item.active&&!item.muted).map(item=>Number(item.channel?.priority)||0));
  for (const item of state.mixer) {
    const gain = liveAudio.gains.get(item.id);
    if (!gain) continue;
		const ducked=highestActivePriority>(Number(item.channel?.priority)||0),target=item.muted||(solo&&!item.solo)?0:item.volume*(ducked?.18:1);
    gain.gain.setTargetAtTime(target, liveAudio.context.currentTime, .015);
		const panner=liveAudio.panners.get(item.id);if(panner)panner.pan.setTargetAtTime(Number(item.pan)||0,liveAudio.context.currentTime,.015);
  }
}

function applyMasterAudio(save = true) {
  masterAudio.volume = Math.max(0, Math.min(1, Number(masterAudio.volume) || 0));
  if (save) localStorage.setItem('gpsdr-master-audio-v1', JSON.stringify(masterAudio));
  const target = masterAudio.muted ? 0 : masterAudio.volume;
  if (liveAudio.masterGain && liveAudio.context) liveAudio.masterGain.gain.setTargetAtTime(target, liveAudio.context.currentTime, .015);
  recordingPlayer.volume = masterAudio.volume;
  recordingPlayer.muted = masterAudio.muted;
  const wrap = $('.master-audio'), button = $('#master-mute'), slider = $('#master-volume'), value = $('#master-volume-value');
  if (wrap) wrap.classList.toggle('muted', masterAudio.muted);
  if (button) { button.textContent = masterAudio.muted ? '🔇' : masterAudio.volume < .45 ? '🔉' : '🔊'; button.setAttribute('aria-pressed', String(masterAudio.muted)); button.title = masterAudio.muted ? 'Unmute all GP-SDR audio' : 'Mute all GP-SDR audio'; }
  if (slider && document.activeElement !== slider) slider.value = masterAudio.volume;
  if (value) value.textContent = `${Math.round(masterAudio.volume * 100)}%`;
}

function channelGain(channelID) {
  let gain = liveAudio.gains.get(channelID);
  if (!gain) {
		gain = liveAudio.context.createGain();
		if(liveAudio.context.createStereoPanner){const panner=liveAudio.context.createStereoPanner();gain.connect(panner);panner.connect(liveAudio.masterGain);liveAudio.panners.set(channelID,panner);}else gain.connect(liveAudio.masterGain);
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
  if (!liveAudio.context) {
    liveAudio.context = new AudioContextClass({latencyHint:'interactive'});
    liveAudio.masterGain = liveAudio.context.createGain();
    liveAudio.masterGain.connect(liveAudio.context.destination);
    applyMasterAudio(false);
  }
  await liveAudio.context.resume();
	const audioState=$('#audio-state'); if(audioState) audioState.textContent=liveAudio.context.state==='running'?'Audio ready':'Audio blocked';
  if (liveAudio.controller) return;
  const controller = new AbortController(); liveAudio.controller = controller;
  void pumpLiveAudio(controller);
}

async function pumpLiveAudio(controller) {
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
	if(canvas.id==='tuner-spectrum'&&state.spectrum){const center=Number($('#tuner-frequency').value)*1e6,bandwidth=Number($('#tuner-bandwidth').value)*1e3,start=state.spectrum.startFrequencyHz,end=state.spectrum.endFrequencyHz,x1=(center-bandwidth/2-start)/(end-start)*width,x2=(center+bandwidth/2-start)/(end-start)*width;ctx.fillStyle='rgba(73,213,170,.08)';ctx.fillRect(Math.max(0,x1),0,Math.min(width,x2)-Math.max(0,x1),height);ctx.strokeStyle='rgba(73,213,170,.45)';ctx.strokeRect(Math.max(0,x1),.5,Math.min(width,x2)-Math.max(0,x1),height-1);}
	if(canvas.id==='mapper-spectrum'&&state.spectrum){const frequencies=mapperSpectrumJob()?.progress?.currentFrequenciesHz||[],start=state.spectrum.startFrequencyHz,end=state.spectrum.endFrequencyHz;ctx.font='9px -apple-system,system-ui,sans-serif';frequencies.forEach((frequency,index)=>{if(frequency<start||frequency>end)return;const x=(frequency-start)/(end-start)*width;ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,height);ctx.strokeStyle='rgba(248,183,82,.78)';ctx.lineWidth=1;ctx.stroke();ctx.fillStyle='rgba(248,202,120,.95)';ctx.fillText(String(index+1),Math.min(width-14,x+3),12+(index%2)*11);});}
	if(canvas.id!=='mapper-spectrum'&&displayPrefs.markers&&state.spectrum){const seen=new Set(),channels=state.profiles.flatMap(profile=>profile.channels||[]).filter(channel=>channel.enabled!==false&&channel.frequencyHz>=state.spectrum.startFrequencyHz&&channel.frequencyHz<=state.spectrum.endFrequencyHz&&(!seen.has(Math.round(channel.frequencyHz))&&seen.add(Math.round(channel.frequencyHz)))).slice(0,24);ctx.font='9px -apple-system,system-ui,sans-serif';for(const channel of channels){const x=(channel.frequencyHz-state.spectrum.startFrequencyHz)/(state.spectrum.endFrequencyHz-state.spectrum.startFrequencyHz)*width;ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,height);ctx.strokeStyle='rgba(90,152,223,.28)';ctx.lineWidth=1;ctx.stroke();ctx.save();ctx.translate(x+3,10);ctx.rotate(Math.PI/2);ctx.fillStyle='rgba(151,164,180,.8)';ctx.fillText(String(channel.name||'Channel').slice(0,22),0,0);ctx.restore();}}
	const sourceBins = state.spectrum?.binsDBFS || [];
	const desired=Math.min(displayPrefs.detail,sourceBins.length), stride=Math.max(1,Math.floor(sourceBins.length/Math.max(1,desired)));
	let bins=sourceBins.filter((_,index)=>index%stride===0).slice(0,desired);
	const old=spectrumHistory.get(canvas); if(old?.length===bins.length && displayPrefs.smoothing>0){const weight=displayPrefs.smoothing/100;bins=bins.map((value,index)=>value*(1-weight)+old[index]*weight);} spectrumHistory.set(canvas,bins);
	let peaks=spectrumPeaks.get(canvas);if(!peaks||peaks.length!==bins.length)peaks=[...bins];else if(displayPrefs.peakHold)peaks=peaks.map((value,index)=>Math.max(value,bins[index]));else peaks=[...bins];spectrumPeaks.set(canvas,peaks);
	const points = bins.length || 180;
	if(displayPrefs.peakHold&&peaks.length){ctx.beginPath();for(let i=0;i<peaks.length;i++){const x=i/(peaks.length-1)*width,normalized=normalizeSpectrumDB(peaks[i]),y=height-8-normalized*(height-18);i?ctx.lineTo(x,y):ctx.moveTo(x,y);}ctx.strokeStyle='rgba(248,183,82,.72)';ctx.lineWidth=1;ctx.setLineDash([3,3]);ctx.stroke();ctx.setLineDash([]);}
  ctx.beginPath();
  for (let i = 0; i < points; i++) {
    const x = i / (points - 1) * width;
    const db = bins.length ? bins[i] : -110;
    const normalized = normalizeSpectrumDB(db);
    const y = height - 8 - normalized * (height - 18);
    i ? ctx.lineTo(x,y) : ctx.moveTo(x,y);
  }
  const gradient = ctx.createLinearGradient(0,0,width,0); gradient.addColorStop(0,'#3f9e86'); gradient.addColorStop(.5,'#55e0b6'); gradient.addColorStop(1,'#5a98df');
  ctx.strokeStyle = state.status?.running||mapperActiveJobs().length ? gradient : '#39434f'; ctx.lineWidth = 1.4; ctx.stroke();
  ctx.lineTo(width,height); ctx.lineTo(0,height); ctx.closePath();
  const fill = ctx.createLinearGradient(0,0,0,height); fill.addColorStop(0,'rgba(73,213,170,.2)'); fill.addColorStop(1,'rgba(73,213,170,0)'); ctx.fillStyle = fill; ctx.fill();
	const cursor=spectrumCursors.get(canvas);if(cursor){const x=cursor.fraction*width;ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,height);ctx.strokeStyle='rgba(255,255,255,.72)';ctx.lineWidth=1;ctx.stroke();}
}

function normalizeSpectrumDB(db){return Math.max(0,Math.min(1,(db-displayPrefs.floor)/(displayPrefs.ceiling-displayPrefs.floor)));}

function drawSpectrum() {
  drawSpectrumCanvas($('#spectrum'));
  drawSpectrumCanvas($('#tuner-spectrum'));
  drawSpectrumCanvas($('#mapper-spectrum'));
}

function waterfallColor(db) {
  const value = normalizeSpectrumDB(db);
  if (value < .25) return [3, 10 + value * 80, 24 + value * 100];
  if (value < .55) return [15, 45 + value * 210, 105 + value * 150];
  if (value < .8) return [45 + value * 190, 210, 150 - value * 70];
  return [255, 232 - value * 80, 120 - value * 90];
}

function drawWaterfall() {
	for(const canvas of [$('#live-waterfall'),$('#waterfall'),$('#mapper-waterfall')]) drawWaterfallCanvas(canvas);
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
    p25Systems: [], settings: {noiseMarginDB:8,revisitSeconds:20,recordAudio:true,recordIQForUnknown:true,transcribeVoice:false,maxRecordingDays:30,p25SampleRateHz:0}, builtIn:false };
}

function openProfileEditor(profile = emptyProfile()) {
  state.editingProfile = structuredClone(profile);
  $('#profile-dialog-title').textContent = profile.name ? 'Edit profile' : 'New profile';
  $('#profile-name').value = profile.name; $('#profile-summary').value = profile.summary || '';
  state.editingProfile.p25Systems ||= [];
  $('#record-audio').checked = profile.settings?.recordAudio !== false;
  $('#record-iq').checked = profile.settings?.recordIQForUnknown !== false;
  $('#transcribe-voice').checked = !!profile.settings?.transcribeVoice;
  $('#recording-retention').value = String(profile.settings?.maxRecordingDays ?? 30);
  $('#p25-sample-rate').value = String(profile.settings?.p25SampleRateHz || 0);
  renderEditorRows(); $('#profile-dialog').showModal();
}

function renderEditorRows() {
  const profile = state.editingProfile;
  $('#ranges-editor').innerHTML = profile.ranges.length ? profile.ranges.map(range => `
    <div class="editor-row" data-id="${range.id}"><input data-key="name" value="${escapeHTML(range.name)}" placeholder="Name" title="Range name">
      <input data-key="startHz" type="number" value="${range.startHz / 1e6}" step="0.0001" placeholder="Start MHz" title="Start frequency in MHz">
      <input data-key="endHz" type="number" value="${range.endHz / 1e6}" step="0.0001" placeholder="End MHz" title="End frequency in MHz">
      <input data-key="stepHz" type="number" value="${range.stepHz / 1000}" step="0.001" placeholder="Step kHz" title="Channel step in kHz">
	  <select data-key="preferredMode" title="Preferred modulation or digital protocol">${profileModeOptions()}</select>
	  <select data-key="decoder" title="Decoder used for activity in this range">${profileDecoderOptions()}</select>
      <button type="button" class="remove-row" title="Remove range">×</button></div>`).join('') : '<div class="editor-empty">No sweep ranges</div>';
  profile.ranges.forEach(range => { const row = $(`#ranges-editor [data-id="${range.id}"]`); if (row) { row.querySelector('[data-key="preferredMode"]').value = range.preferredMode; row.querySelector('[data-key="decoder"]').value = range.decoder || ''; } });
  $('#channels-editor').innerHTML = profile.channels.length ? profile.channels.map(channel => `
    <div class="editor-row channel" data-id="${channel.id}"><input data-key="name" value="${escapeHTML(channel.name)}" placeholder="Name" title="Channel name">
      <input data-key="frequencyHz" type="number" value="${channel.frequencyHz / 1e6}" step="0.0001" placeholder="MHz" title="Frequency in MHz">
	  <select data-key="mode" title="Modulation or digital protocol">${profileModeOptions()}</select>
	  <select data-key="decoder" title="Decoder for this channel">${profileDecoderOptions()}</select>
			<input class="advanced-only" data-key="priority" type="number" min="0" max="100" step="1" value="${Number(channel.priority)||0}" placeholder="Priority" title="Higher-priority active channels automatically duck lower-priority audio">
      <button type="button" class="remove-row" title="Remove channel">×</button></div>`).join('') : '<div class="editor-empty">No fixed channels</div>';
  profile.channels.forEach(channel => { const row = $(`#channels-editor [data-id="${channel.id}"]`); if (row) { row.querySelector('[data-key="mode"]').value = channel.mode; row.querySelector('[data-key="decoder"]').value = channel.decoder || ''; } });
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
	  dwellMilliseconds: decoderForSelectedMode(row.querySelector('[data-key="preferredMode"]').value)?2500:180, preferredMode: row.querySelector('[data-key="preferredMode"]').value, decoder:row.querySelector('[data-key="decoder"]').value.trim()||decoderForSelectedMode(row.querySelector('[data-key="preferredMode"]').value)||null, enabled: true
  }));
  profile.channels = $$('#channels-editor .editor-row').map(row => ({
    id: row.dataset.id, name: row.querySelector('[data-key="name"]').value.trim() || 'Channel',
    frequencyHz: Number(row.querySelector('[data-key="frequencyHz"]').value) * 1e6,
    bandwidthHz: row.querySelector('[data-key="mode"]').value === 'wfm' ? 180000 : 12500,
    mode: row.querySelector('[data-key="mode"]').value,
	  decoder: row.querySelector('[data-key="decoder"]').value.trim() || decoderForSelectedMode(row.querySelector('[data-key="mode"]').value) || null, enabled: true, priority: Number(row.querySelector('[data-key="priority"]')?.value)||0
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
  profile.settings.recordIQForUnknown = $('#record-iq').checked;
  profile.settings.transcribeVoice = $('#transcribe-voice').checked;
  profile.settings.maxRecordingDays = Number($('#recording-retention').value);
  profile.settings.p25SampleRateHz = Number($('#p25-sample-rate').value) || 0;
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
  const referenceArea={provider:'RadioReference',latitude:result.location.latitude,longitude:result.location.longitude,radiusMiles:result.radiusMiles,label:baseName,importedAt:new Date().toISOString()};
  const profiles = [];
  if (channelIndexes.length) {
    const profile = emptyProfile();
    profile.name = baseName + ' · Channels';
    profile.summary = 'RadioReference location import';
    profile.referenceArea=referenceArea;
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
      profile.referenceArea=referenceArea;
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
      if (install.closest('#missing-components-dialog')) { $('#missing-components-dialog').close(); setView('hardware'); }
      toast('Installation started'); await refreshAll();
    } catch (error) { toast(error.message,true); await refreshAll(); }
    return;
  }
  const nav = event.target.closest('.nav-item'); if (nav) return setView(nav.dataset.view);
  const removeRemote=event.target.closest('.remove-remote'); if(removeRemote){try{await api('/api/remote-receivers?id='+encodeURIComponent(removeRemote.dataset.remoteId),{method:'DELETE'});toast('Remote receiver removed');await refreshAll();}catch(error){toast(error.message,true);}return;}
	const decoderConfig=event.target.closest('[data-decoder-config]');if(decoderConfig){const id=decoderConfig.dataset.decoderConfig,mode=id==='dsd-fme'?'dmr':id==='p25'?'p25':id==='multimon-ng'?'pocsag':id==='acarsdec'?'acars':id==='analog'?'nfm':'auto',decoder=id==='analog'?'':id==='p25'?'dsd-fme':id;const profile=emptyProfile();profile.name=id==='dsd-fme'?'DMR channels':`${state.decoders.find(item=>item.id===id)?.name||id} channels`;profile.summary='Custom decoder channel bank';profile.channels=[{id:crypto.randomUUID(),name:'New channel',frequencyHz:450e6,bandwidthHz:12500,mode,decoder:decoder||null,enabled:true,priority:5}];openProfileEditor(profile);return;}
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
	$('#tuner-bandwidth').value = event.target.value === 'wfm' ? '180' : event.target.value === 'am'||event.target.value==='acars' ? '10' : '12.5';
	$('#live-mode').value=event.target.value;$('#live-bandwidth').value=$('#tuner-bandwidth').value;
	if(event.target.value==='wfm'){ $('#tuner-lna').value='32'; $('#tuner-vga').value='24'; $('#tuner-squelch').value='4'; $('#tuner-agc').checked=true; $('#tuner-monitor-open').checked=true; }
	queueReceiverControls();
});

async function applyReceiverControls() {
	if(receiverApplying) return;
	receiverApplying=true; const panel=$('.receiver-controls-panel'), button=$('#live-apply-radio'); panel?.classList.add('applying'); panel?.classList.remove('applied'); button.disabled=true; button.textContent='Applying…'; $('#audio-state').textContent='Applying settings';
	try { await startLiveAudio(); await api('/api/tuner/start',{method:'POST',body:JSON.stringify(tunerRequest())}); delete $('#tuner-frequency').dataset.pending; button.textContent='Applied'; panel?.classList.add('applied'); $('#audio-state').textContent='Settings applied'; $('#tuner-status').textContent='Settings applied'; }
	catch(error) { delete $('#tuner-frequency').dataset.pending; button.textContent='Retry'; $('#audio-state').textContent='Apply failed'; $('#tuner-status').textContent='Apply failed'; toast(error.message,true); }
	finally { receiverApplying=false; button.disabled=false; panel?.classList.remove('applying'); }
}

function queueReceiverControls() { if(!state.status?.running || state.status?.activeProfileID!=='quick-tune') return; clearTimeout(receiverApplyTimer); $('#live-apply-radio').textContent='Pending…'; $('#audio-state').textContent='Settings pending'; $('#tuner-status').textContent='Applying settings…'; receiverApplyTimer=setTimeout(applyReceiverControls,250); }

$$('#view-live .receiver-control-grid input, #view-live .receiver-control-grid select').forEach(control=>control.addEventListener(control.type==='number'?'input':'change',()=>{
	const map={'live-radio-device':'tuner-device','live-mode':'tuner-mode','live-bandwidth':'tuner-bandwidth','live-lna':'tuner-lna','live-vga':'tuner-vga','live-ppm':'tuner-ppm','live-iq-gain':'tuner-iq-gain','live-iq-phase':'tuner-iq-phase','live-squelch':'tuner-squelch','live-amp':'tuner-amp','live-bias':'tuner-bias','live-dc':'tuner-dc','live-iq-swap':'tuner-iq-swap','live-agc':'tuner-agc','live-monitor-open':'tuner-monitor-open','live-use-calibration':'tuner-use-calibration'};
	const target=$('#'+map[control.id]); if(target){if(control.type==='checkbox')target.checked=control.checked;else target.value=control.value;} queueReceiverControls();
}));
$$('#view-tuner .advanced-radio input, #view-tuner .advanced-radio select, #tuner-gain, #tuner-rate').forEach(control=>control.addEventListener(control.type==='number'?'input':'change',queueReceiverControls));
$('#tuner-form').addEventListener('submit', async event => {
  event.preventDefault();
	const request = tunerRequest();
	try { void startLiveAudio(); await api('/api/tuner/start',{method:'POST',body:JSON.stringify(request)});delete $('#tuner-frequency').dataset.pending;rememberTunerFrequency(request);toast(request.lockCenter&&state.status?.activeProfileID==='quick-tune'?'Software VFO moved':'Tuner started'); await refreshAll(); }
  catch(error) { stopLiveAudio(); toast(error.message,true); }
});
$('#live-apply-radio').addEventListener('click', async () => {
	[['live-radio-device','tuner-device'],['live-mode','tuner-mode'],['live-bandwidth','tuner-bandwidth'],['live-lna','tuner-lna'],['live-vga','tuner-vga'],['live-ppm','tuner-ppm'],['live-iq-gain','tuner-iq-gain'],['live-iq-phase','tuner-iq-phase'],['live-squelch','tuner-squelch']].forEach(([from,to])=>$('#'+to).value=$('#'+from).value);
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
$('#event-search').addEventListener('input', searchEvents);
$('#mixer-search').addEventListener('input', renderMixer);
$('#mixer-sort').value = state.mixerOrder;
$('#mixer-sort').addEventListener('change', event => { state.mixerOrder = event.target.value; localStorage.setItem('gpsdr-mixer-order', state.mixerOrder); renderMixer(); });
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
$('#local-db-choose').addEventListener('click',()=>{const native=window.webkit?.messageHandlers?.gpsdrNative;if(native)native.postMessage({action:'chooseLocalDatabaseFolder'});else $('#local-db-upload').click();});
$('#local-db-form').addEventListener('submit',event=>{event.preventDefault();window.setLocalDatabaseFolder($('#local-db-folder').value.trim());});
$('#local-db-scan').addEventListener('click',async()=>{try{state.localDatabase=await api('/api/local-database/scan',{method:'POST',body:'{}'});renderLocalDatabase();toast('Local database scan started');setTimeout(refreshAll,800);}catch(error){toast(error.message,true);}});
$('#local-db-upload').addEventListener('change',async event=>{const files=[...event.target.files].filter(file=>/\.(csv|tsv|json)$/i.test(file.name));if(!files.length)return;let imported=0,banks=0;try{for(const file of files){const filename=file.webkitRelativePath||file.name;const isJSON=file.name.toLowerCase().endsWith('.json');const endpoint=isJSON?'/api/profiles/import':'/api/profiles/import-database?filename='+encodeURIComponent(filename);const result=await api(endpoint,{method:'POST',body:await file.text()});imported++;banks+=isJSON?1:(result.profileCount||0);}toast(`${imported} files imported · ${banks} channel banks`);await refreshAll();}catch(error){toast(`${imported?`${imported} imported · `:''}${error.message}`,true);}event.target.value='';});
$('#rr-credentials-form').addEventListener('submit',async event=>{
  event.preventDefault(); const username=$('#rr-username').value.trim(),password=$('#rr-password').value,appKey=$('#rr-app-key').value.trim();
  if(!username||!password||!appKey)return toast('Username, password, and approved app key are required',true);
  try{await api('/api/radioreference/credentials',{method:'PUT',body:JSON.stringify({username,password,appKey})});$('#rr-password').value='';$('#rr-app-key').value='';toast('RadioReference saved to Mac Keychain');await refreshAll();}catch(error){toast(error.message,true);}
});
$('#rr-clear').addEventListener('click',async()=>{try{await api('/api/radioreference/credentials',{method:'DELETE'});$('#rr-username').value='';$('#rr-password').value='';$('#rr-app-key').value='';toast('RadioReference credentials cleared');await refreshAll();}catch(error){toast(error.message,true);}});
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
$('#missing-components-ignore').addEventListener('click',()=>localStorage.setItem('gpsdr-ignored-components',$('#missing-components-dialog').dataset.signature||''));
$('#missing-components-dialog .dialog-close').addEventListener('click',()=>sessionStorage.setItem('gpsdr-dismissed-components',$('#missing-components-dialog').dataset.signature||''));
$('#missing-components-review').addEventListener('click',()=>{$('#missing-components-dialog').close();setView('hardware');});
$('#remote-form').addEventListener('submit',async event=>{event.preventDefault();try{await api('/api/remote-receivers',{method:'PUT',body:JSON.stringify({name:$('#remote-name').value.trim(),host:$('#remote-host').value.trim(),port:Number($('#remote-port').value),enabled:true})});toast('Remote receiver saved');$('#remote-host').value='';await refreshAll();}catch(error){toast(error.message,true);}});
$('#characterization-range-mode').addEventListener('change',updateCharacterizationRangeControls);
$('#characterization-use-calibration').addEventListener('change',updateCharacterizationGainControls);
$('#characterization-form').addEventListener('submit',async event=>{event.preventDefault();const deviceIDs=$$('#characterization-devices input:checked').map(input=>input.value),rangeMode=$('#characterization-range-mode').value,startMHz=rangeMode==='antenna'?Number($('#characterization-antenna-min').value):Number($('#characterization-start').value),endMHz=rangeMode==='antenna'?Number($('#characterization-antenna-max').value):Number($('#characterization-end').value);try{state.characterization=await api('/api/calibrations/characterization/start',{method:'POST',body:JSON.stringify({deviceIDs,name:`${$('#characterization-antenna').value.trim()||'Receiver'} comparison`,antennaLabel:$('#characterization-antenna').value.trim(),rangeMode,startHz:startMHz*1e6,endHz:endMHz*1e6,antennaMinimumHz:Number($('#characterization-antenna-min').value)*1e6,antennaMaximumHz:Number($('#characterization-antenna-max').value)*1e6,points:Number($('#characterization-points').value),dwellMilliseconds:Number($('#characterization-dwell').value),sampleRateHz:Number($('#characterization-rate').value)||0,useCalibration:$('#characterization-use-calibration').checked,gainDB:Number($('#characterization-gain').value),lnaGainDB:Number($('#characterization-lna').value),vgaGainDB:Number($('#characterization-vga').value),ampEnabled:$('#characterization-amp').checked})});toast('Receiver comparison started');renderCharacterization();}catch(error){toast(error.message,true);}});
$('#characterization-stop').addEventListener('click',async()=>{try{state.characterization=await api('/api/calibrations/characterization/stop',{method:'POST'});toast('Receiver comparison stopping');renderCharacterization();}catch(error){toast(error.message,true);}});
$('#characterization-export').addEventListener('click',()=>{window.location.href=`/api/calibrations/characterization/export${serverToken?`?token=${encodeURIComponent(serverToken)}`:''}`;});
$('#characterization-clear').addEventListener('click',async()=>{if(!await confirmAction({title:'Clear receiver comparison?',message:'Remove the saved receiver and antenna response measurements? Device calibration values are kept.',confirmLabel:'Clear results'}))return;try{state.characterization=await api('/api/calibrations/characterization',{method:'DELETE'});toast('Receiver comparison cleared');renderCharacterization();}catch(error){toast(error.message,true);}});
function collectMapperJob(){const workflow=mapperWorkflowValue(),start=Number($('#mapper-start').value)*1e6,end=Number($('#mapper-end').value)*1e6,deviceID=$('#mapper-device').value,step=Number($('#mapper-step').value)*1000,dwell=mapperDwellMilliseconds(),includeLocation=$('#mapper-location').checked;
  const latitude=$('#mapper-latitude').value===''?null:Number($('#mapper-latitude').value),longitude=$('#mapper-longitude').value===''?null:Number($('#mapper-longitude').value);
  const decipherListenSeconds=Math.round(Number($('#mapper-listen-value').value)*Number($('#mapper-listen-unit').value));
  if(workflow==='decipher'&&(decipherListenSeconds<5||decipherListenSeconds>7*86400))throw new Error('Choose 5 seconds to 7 days per channel');
  if(!deviceID)throw new Error('Select a receiver'); if(workflow!=='decipher'&&(!Number.isFinite(start)||!Number.isFinite(end)||end<start||step<=0))throw new Error('Enter a valid mapping range');if(workflow!=='decipher'&&(dwell<100||dwell>7*86400000))throw new Error('Choose 0.1 seconds to 7 days per channel'); if(includeLocation&&(!Number.isFinite(latitude)||!Number.isFinite(longitude)))throw new Error('Add a valid location or turn location tagging off');
	return {id:$('#mapper-job-id').value,name:$('#mapper-job-name').value.trim(),config:{mode:workflow,preferredMode:$('#mapper-mode').value,preferredDecoder:$('#mapper-decoder').value,deviceID,startHz:start,endHz:end,stepHz:step,dwellMilliseconds:dwell,sampleRateHz:Number($('#mapper-rate').value)||0,concurrentChannels:Number($('#mapper-concurrent').value)||0,gainMode:$('#mapper-gain-mode').value,gainDB:Number($('#mapper-gain').value),lnaGainDB:Number($('#mapper-lna').value),vgaGainDB:Number($('#mapper-vga').value),ampMode:$('#mapper-amp-mode').value,sensitivity:$('#mapper-sensitivity').value,noiseMarginDB:Number($('#mapper-noise-margin').value),decipherListenSeconds,identifyMinimumHits:Number($('#mapper-identify-min-hits').value)||1,identifyHitSource:$('#mapper-identify-hit-source').value,identifyMinimumOccupancy:Number($('#mapper-identify-occupancy').value)||0,identifySeenWithinHours:Number($('#mapper-identify-age').value)||0,identifyMaximumChannels:Number($('#mapper-identify-limit').value)||0,identifyOrder:$('#mapper-identify-order').value,transcribe:$('#mapper-transcribe').checked,includeLocation,locationPrecision:$('#mapper-location-precision').value,locationLabel:$('#mapper-location-label').value.trim(),latitude,longitude}};
}
async function saveMapperJob(start=false){try{const job=await api('/api/mapper/jobs',{method:'POST',body:JSON.stringify(collectMapperJob())});$('#mapper-job-id').value=job.id;if(start){state.mapper=await api('/api/mapper/jobs/start',{method:'POST',body:JSON.stringify({id:job.id})});toast(`${job.name} started`);}else toast(`${job.name} saved`);await refreshAll();return job;}catch(error){toast(error.message,true);return null;}}
function loadMapperJob(job,duplicate=false){const config=job?.config||{},suffix=duplicate?' copy':'';$('#mapper-job-id').value=duplicate?'':job?.id||'';$('#mapper-job-name').value=(job?.name||'')+suffix;if(config.deviceID)$('#mapper-device').value=config.deviceID;setMapperWorkflow(config.mode||'adaptive');$('#mapper-mode').value=config.preferredMode||'auto';$('#mapper-decoder').value=config.preferredDecoder||'auto';if(config.startHz)$('#mapper-start').value=config.startHz/1e6;if(config.endHz)$('#mapper-end').value=config.endHz/1e6;if(config.stepHz)$('#mapper-step').value=config.stepHz/1000;setMapperDwellInput(config.dwellMilliseconds||500);$('#mapper-rate').value=String(config.sampleRateHz||0);$('#mapper-concurrent').value=String(config.concurrentChannels||0);$('#mapper-gain-mode').value=config.gainMode||'auto';$('#mapper-gain').value=String(config.gainDB??20);$('#mapper-lna').value=String(config.lnaGainDB??16);$('#mapper-vga').value=String(config.vgaGainDB??16);$('#mapper-amp-mode').value=config.ampMode||'auto';$('#mapper-sensitivity').value=config.sensitivity||'auto';$('#mapper-noise-margin').value=String(config.noiseMarginDB||6);setMapperListenInput(config.decipherListenSeconds||60);$('#mapper-identify-min-hits').value=String(config.identifyMinimumHits||2);$('#mapper-identify-hit-source').value=config.identifyHitSource||'discovery';$('#mapper-identify-occupancy').value=String(config.identifyMinimumOccupancy??.1);$('#mapper-identify-age').value=String(config.identifySeenWithinHours??168);$('#mapper-identify-limit').value=String(config.identifyMaximumChannels??50);$('#mapper-identify-order').value=config.identifyOrder||'hits';$('#mapper-transcribe').checked=!!config.transcribe;$('#mapper-location').checked=!!config.includeLocation;$('#mapper-location-precision').value=config.locationPrecision||'approximate';$('#mapper-location-label').value=config.locationLabel||'';$('#mapper-latitude').value=config.latitude??'';$('#mapper-longitude').value=config.longitude??'';updateMapperWorkflow();$('#mapper-job-name').focus();}
function resetMapperJob(){loadMapperJob({config:{mode:'adaptive',preferredDecoder:'auto',startHz:10e6,endHz:6e9,stepHz:12500,dwellMilliseconds:500,decipherListenSeconds:60,sampleRateHz:0,concurrentChannels:0,gainMode:'auto',lnaGainDB:16,vgaGainDB:16,ampMode:'auto',sensitivity:'auto',noiseMarginDB:6,identifyMinimumHits:2,identifyHitSource:'discovery',identifyMinimumOccupancy:.1,identifySeenWithinHours:168,identifyMaximumChannels:50,identifyOrder:'hits',transcribe:true}},false);$('#mapper-job-id').value='';}
$('#mapper-form').addEventListener('submit',async event=>{event.preventDefault();await saveMapperJob(true);});
$('#mapper-mode').addEventListener('change',event=>{const decoder=decoderForSelectedMode(event.target.value);if(decoder)$('#mapper-decoder').value=decoder;});
$('#mapper-save-job').addEventListener('click',()=>saveMapperJob(false));
$('#mapper-new-job').addEventListener('click',resetMapperJob);
$('#mapper-import-jobs').addEventListener('click',()=>$('#mapper-job-files').click());
$('#mapper-job-files').addEventListener('change',async event=>{let imported=0;try{for(const file of event.target.files){const job=JSON.parse(await file.text());job.id='';job.state='idle';job.progress={};await api('/api/mapper/jobs',{method:'POST',body:JSON.stringify(job)});imported++;}toast(`${imported} Mapper ${imported===1?'job':'jobs'} imported`);await refreshAll();}catch(error){toast(`${imported?`${imported} imported · `:''}${error.message}`,true);}event.target.value='';});
$('#mapper-stop-button').addEventListener('click',async()=>{try{state.mapper=await api('/api/mapper/jobs/stop-all',{method:'POST',body:'{}'});toast('All Mapper jobs stopping');await refreshAll();}catch(error){toast(error.message,true);}});
$('#mapper-job-grid').addEventListener('click',async event=>{const button=event.target.closest('[data-job-action]'),card=event.target.closest('[data-job-id]');if(!button||!card)return;const job=state.mapper?.jobs?.find(item=>item.id===card.dataset.jobId);if(!job)return;try{if(button.dataset.jobAction==='edit'){loadMapperJob(job);return;}if(button.dataset.jobAction==='duplicate'){loadMapperJob(job,true);return;}if(button.dataset.jobAction==='export'){window.location.href=`/api/mapper/jobs/export?id=${encodeURIComponent(job.id)}${serverToken?`&token=${encodeURIComponent(serverToken)}`:''}`;return;}if(button.dataset.jobAction==='delete'){if(!await confirmAction({title:'Delete Mapper job?',message:`Delete ${job.name}? Its saved settings will be removed; mapped results are kept.`,confirmLabel:'Delete job'}))return;state.mapper=await api(`/api/mapper/jobs?id=${encodeURIComponent(job.id)}`,{method:'DELETE'});toast('Mapper job deleted');}if(button.dataset.jobAction==='start'){state.mapper=await api('/api/mapper/jobs/start',{method:'POST',body:JSON.stringify({id:job.id})});toast(`${job.name} started`);}if(button.dataset.jobAction==='stop'){state.mapper=await api('/api/mapper/jobs/stop',{method:'POST',body:JSON.stringify({id:job.id})});toast(`${job.name} stopping`);}await refreshAll();}catch(error){toast(error.message,true);}});
for(const id of ['mapper-filter-job','mapper-filter-device','mapper-filter-type','mapper-filter-state','mapper-sort','mapper-filter-search','mapper-filter-repeated'])$('#'+id).addEventListener(id==='mapper-filter-search'?'input':'change',renderMapper);
$('#mapper-filter-reset').addEventListener('click',()=>{for(const id of ['mapper-filter-job','mapper-filter-device','mapper-filter-type','mapper-filter-state'])$('#'+id).value='';$('#mapper-sort').value='recent';$('#mapper-filter-search').value='';$('#mapper-filter-repeated').checked=false;renderMapper();toast('Mapper filters reset');});
$('#mapper-results-toggle').addEventListener('click',()=>setMapperResultsCollapsed(!mapperResultsCollapsed));
$('#mapper-clear').addEventListener('click',async()=>{if(!await confirmAction({title:'Clear Mapper results?',message:'Remove every recorded Mapper frequency and its activity history? Saved jobs and settings are kept.',confirmLabel:'Clear results'}))return;try{state.mapper=await api('/api/mapper/clear',{method:'POST',body:'{}'});renderMapper();toast('Mapper results cleared');}catch(error){toast(error.message,true);}});
$('#mapper-save').addEventListener('click',async()=>{try{const result=await api('/api/mapper/save',{method:'POST',body:'{}'});toast(`${result.rows} rows saved · ${result.path}`);}catch(error){toast(error.message,true);}});
$('#mapper-download').addEventListener('click',()=>{window.location.href=`/api/mapper/export.csv${serverToken?`?token=${encodeURIComponent(serverToken)}`:''}`;});
$('#mapper-sheet-form').addEventListener('submit',async event=>{event.preventDefault();try{state.mapper=await api('/api/mapper',{method:'PUT',body:JSON.stringify({...(state.mapper?.config||{}),sheetURL:$('#mapper-sheet-url').value.trim(),webhookURL:$('#mapper-webhook').value.trim(),contributor:$('#mapper-contributor').value.trim()||'GP-SDR',secret:$('#mapper-secret').value,autoUpload:$('#mapper-auto-upload').checked,uploadVerifiedOnly:$('#mapper-upload-verified').checked})});renderMapper();toast('Additions Queue settings saved');}catch(error){toast(error.message,true);}});
$('#mapper-body').addEventListener('click',async event=>{const button=event.target.closest('.mapper-send-one');if(button){button.disabled=true;try{state.mapper=await api('/api/mapper/upload-one',{method:'POST',body:JSON.stringify({frequencyHz:Number(button.dataset.frequencyHz)})});renderMapper();if(state.mapper.lastError)toast(state.mapper.lastError,true);else toast('Observation added to Additions Queue');}catch(error){toast(error.message,true);button.disabled=false;}return;}const row=event.target.closest('.mapper-result-row');if(!row)return;const key=row.dataset.mapperFrequency;if(expandedMapperFrequencies.has(key))expandedMapperFrequencies.delete(key);else expandedMapperFrequencies.add(key);row.classList.toggle('expanded');row.querySelector('.mapper-frequency-button')?.setAttribute('aria-expanded',String(expandedMapperFrequencies.has(key)));$('#mapper-body').querySelector(`[data-mapper-detail="${CSS.escape(key)}"]`)?.classList.toggle('hidden',!expandedMapperFrequencies.has(key));});
$('#mapper-script-download').addEventListener('click',()=>{window.location.href=`/api/mapper/apps-script.gs${serverToken?`?token=${encodeURIComponent(serverToken)}`:''}`;});
$('#mapper-upload-now').addEventListener('click',async()=>{try{state.mapper=await api('/api/mapper/upload',{method:'POST',body:'{}'});renderMapper();if(state.mapper.lastError)toast(state.mapper.lastError,true);else toast('New Mapper activity uploaded');}catch(error){toast(error.message,true);}});
$('#mapper-workflow').addEventListener('change',updateMapperWorkflow);
$('#mapper-gain-mode').addEventListener('change',updateMapperGainControls);
$('#mapper-sensitivity').addEventListener('change',updateMapperGainControls);
$('#mapper-current-location').addEventListener('click',()=>{const button=$('#mapper-current-location'),native=window.webkit?.messageHandlers?.gpsdrNative,nativeLocation=window.gpsdrNativeCapabilities?.includes('location');if(button.dataset.locationSettings==='true'&&nativeLocation){native.postMessage({action:'openLocationSettings'});return;}button.textContent='Locating…';button.dataset.locationSettings='';if(nativeLocation){native.postMessage({action:'requestLocation'});return;}if(!navigator.geolocation)return mapperLocationError('Location is unavailable on this device');navigator.geolocation.getCurrentPosition(position=>setMapperLocation(position.coords.latitude,position.coords.longitude),error=>mapperLocationError(error.message||'Could not read location'),{enableHighAccuracy:false,timeout:10000,maximumAge:300000});});
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
$('#master-mute').addEventListener('click',()=>{masterAudio.muted=!masterAudio.muted;applyMasterAudio();toast(masterAudio.muted?'GP-SDR audio muted':'GP-SDR audio unmuted');});
$('#master-volume').addEventListener('input',event=>{masterAudio.volume=Number(event.currentTarget.value);masterAudio.muted=false;applyMasterAudio();});
document.addEventListener('change', async event => {
	const order=event.target.closest('#p25-order');if(order){state.p25Order=order.value;localStorage.setItem('gpsdr-p25-order',order.value);renderDecoders();return;}
	const p25Rate=event.target.closest('#p25-live-rate');if(p25Rate){const profile=state.profiles.find(item=>item.id===(state.p25Status?.profileID||state.status?.activeProfileID));if(!profile)return toast('Start or select a P25 profile first',true);try{const wasRunning=state.status?.running&&state.status.activeProfileID===profile.id;let updated=profile.builtIn?await api('/api/profiles/duplicate?id='+encodeURIComponent(profile.id),{method:'POST'}):structuredClone(profile);updated.settings||={};updated.settings.p25SampleRateHz=Number(p25Rate.value)||0;updated=await api('/api/profiles',{method:'POST',body:JSON.stringify(updated)});if(wasRunning){await api('/api/control/stop',{method:'POST',body:'{}'});await api('/api/control/start',{method:'POST',body:JSON.stringify({profileID:updated.id})});}state.selectedProfileID=updated.id;toast(profile.builtIn?'Editable profile created · capture width applied':'P25 capture width applied');await refreshAll();}catch(error){toast(error.message,true);}return;}
	const volume=event.target.closest('.mixer-volume'),pan=event.target.closest('.mixer-pan');if(!volume&&!pan)return;const slider=volume||pan,row=slider.closest('[data-mixer-id]');
	try{await api('/api/mixer',{method:'POST',body:JSON.stringify({id:row.dataset.mixerId,...(volume?{volume:Number(slider.value)}:{pan:Number(slider.value)})})});await refreshAll();}catch(error){toast(error.message,true);}
});

$('#add-range').addEventListener('click',()=>{state.editingProfile.ranges.push({id:crypto.randomUUID(),name:'New range',startHz:144e6,endHz:148e6,stepHz:12500,dwellMilliseconds:180,preferredMode:'auto',decoder:null,enabled:true});renderEditorRows();});
$('#add-channel').addEventListener('click',()=>{state.editingProfile.channels.push({id:crypto.randomUUID(),name:'New channel',frequencyHz:462.55e6,bandwidthHz:12500,mode:'nfm',decoder:null,enabled:true,priority:5});renderEditorRows();});
$('#profile-dialog').addEventListener('change',event=>{if(!event.target.matches('[data-key="mode"],[data-key="preferredMode"]'))return;const row=event.target.closest('.editor-row'),decoder=row?.querySelector('[data-key="decoder"]'),suggested=decoderForSelectedMode(event.target.value);if(decoder&&suggested)decoder.value=suggested;});
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
const savedView = localStorage.getItem('gpsdr-last-view');
if (decoderHash) { state.selectedDecoderID = decodeURIComponent(decoderHash[1]); setView('decoders'); }
else if (['live','tuner','activity','mapper','profiles','decoders','hardware','settings'].includes(savedView)) setView(savedView);
const interfaceMode=localStorage.getItem('gpsdr-interface-mode')||'beginner';
$('#interface-mode').value=interfaceMode;document.body.classList.toggle('advanced-mode',interfaceMode==='advanced');
$('#interface-mode').addEventListener('change',event=>{const advanced=event.target.value==='advanced';document.body.classList.toggle('advanced-mode',advanced);localStorage.setItem('gpsdr-interface-mode',event.target.value);toast(advanced?'Advanced controls enabled':'Automatic controls enabled');});
$('#storage-policy-form').addEventListener('submit',async event=>{event.preventDefault();const gb=1024**3;try{const storage=await api('/api/storage/policy',{method:'PUT',body:JSON.stringify({autoCleanup:$('#storage-auto-cleanup').checked,autoRemoveQuarantine:$('#storage-auto-remove-rejected').checked,quarantineRetentionHours:Number($('#storage-rejected-hours').value),maxCaptureDays:Number($('#storage-max-days').value),recordingCapBytes:Math.round(Number($('#storage-recording-cap').value)*gb),iqCapBytes:Math.round(Number($('#storage-iq-cap').value)*gb)})});state.status.storage=storage;renderStatus();toast('Storage limits saved');}catch(error){toast(error.message,true);}});
$('#storage-clean-now').addEventListener('click',async()=>{if(!await confirmAction({title:'Clean stored captures?',message:'Remove the oldest GP-SDR recordings and IQ evidence until the saved age and size limits are met? Profiles, Mapper results, and channel data will be kept.',confirmLabel:'Clean captures'}))return;try{$('#storage-clean-now').disabled=true;const storage=await api('/api/storage/cleanup',{method:'POST'});state.status.storage=storage;renderStatus();toast(`Cleanup complete · ${formatBytes(storage.lastCleanup?.bytesFreed||0)} freed`);}catch(error){toast(error.message,true);$('#storage-clean-now').disabled=false;}});
resetMapperJob();
refreshAll();
setInterval(async()=>{
  if(document.hidden)return;
	if(state.view==='mapper'&&$('#mapper-form').contains(document.activeElement))return;
	try{const requests=[api('/api/status'),api('/api/mixer'),api('/api/p25/status')],mapperIndex=state.view==='mapper'?requests.push(api('/api/mapper/jobs'))-1:-1,characterizationIndex=state.view==='hardware'&&state.characterization?.running?requests.push(api('/api/calibrations/characterization'))-1:-1,responses=await Promise.all(requests),[status,mixer,p25Status]=responses,mapperJobs=mapperIndex>=0?responses[mapperIndex]:null,characterization=characterizationIndex>=0?responses[characterizationIndex]:null;Object.assign(state,{status,mixer,p25Status});if(characterization){state.characterization=characterization;renderCharacterization();}if(mapperJobs&&state.mapper){state.mapper.jobs=mapperJobs;const active=mapperJobs.filter(job=>job.state==='running'||job.state==='stopping');if(!$('#mapper-form').contains(document.activeElement))$('#mapper-job-grid').innerHTML=mapperJobs.map(mapperJobHTML).join('')||'<div class="empty-state compact">Create one job per SDR. Each receiver can run its own range and workflow.</div>';$('#mapper-state').textContent=active.length?`${active.length} active`:'Idle';$('#mapper-state').className=`chip ${active.length?'ready':''}`;$('#mapper-stop-button').disabled=!active.length;}renderStatus();if(state.view==='live')renderMixer();if(state.view==='tuner')renderTuner();if(state.view==='decoders')renderDecoders();if(state.view==='mapper')renderMapperProgress();}catch(_){ }
},750);
setInterval(async()=>{
	if(document.hidden)return;
	if(state.view==='mapper'&&$('#mapper-form').contains(document.activeElement))return;
	try{const eventQuery=$('#event-search')?.value.trim()||'';const requests=[api(`/api/events?limit=150${eventQuery?`&q=${encodeURIComponent(eventQuery)}`:''}`),api('/api/signals?limit=400')];if(state.view==='mapper')requests.push(api('/api/mapper'));const [events,signals,mapper]=await Promise.all(requests);Object.assign(state,{events,signals});if(mapper)state.mapper=mapper;renderLatest();if(state.view==='activity'){renderSignals();renderEvents();}if(state.view==='mapper'&&mapper)renderMapper();}catch(_){ }
},5000);
async function pollSpectrum() {
	const mapperRunning=state.view==='mapper'&&mapperActiveJobs().length>0;
	if(!document.hidden && ((state.status?.running&&(state.view==='live'||state.view==='tuner'))||mapperRunning)){try{state.spectrum=await api('/api/spectrum?bins='+displayPrefs.detail);renderTuner();if(state.view==='mapper')renderMapperRF();drawSpectrum();drawWaterfall();}catch(_){ }}
	setTimeout(pollSpectrum,Math.max(40,1000/displayPrefs.fps));
}
pollSpectrum();

function saveDisplayPrefs(){localStorage.setItem('gpsdr-display-v2',JSON.stringify(displayPrefs));$('#display-smoothing-value').textContent=displayPrefs.smoothing+'%';$$('#live-waterfall,#waterfall,#mapper-waterfall').forEach(canvas=>{canvas.dataset.lastFrame='';canvas.width=0;});if(!displayPrefs.peakHold)[$('#spectrum'),$('#tuner-spectrum'),$('#mapper-spectrum')].forEach(canvas=>spectrumPeaks.delete(canvas));drawSpectrum();drawWaterfall();}
for(const [id,key,number] of [['display-fps','fps',true],['display-quality','quality',true],['display-detail','detail',true],['display-smoothing','smoothing',true]]){const control=$('#'+id);control.value=displayPrefs[key];control.addEventListener('input',()=>{displayPrefs[key]=number?Number(control.value):control.value;saveDisplayPrefs();});}
$('#display-peak-hold').checked=!!displayPrefs.peakHold;$('#display-peak-hold').addEventListener('change',event=>{displayPrefs.peakHold=event.target.checked;saveDisplayPrefs();});
$('#display-markers').checked=displayPrefs.markers!==false;$('#display-markers').addEventListener('change',event=>{displayPrefs.markers=event.target.checked;saveDisplayPrefs();});
function frequencyAtPointer(event){const snapshot=state.spectrum;if(!snapshot?.binsDBFS?.length)return null;const rect=event.currentTarget.getBoundingClientRect(),fraction=Math.max(0,Math.min(1,(event.clientX-rect.left)/rect.width));return {fraction,frequencyHz:snapshot.startFrequencyHz+fraction*(snapshot.endFrequencyHz-snapshot.startFrequencyHz)};}
function setVFOFromPointer(event){const point=frequencyAtPointer(event);if(!point)return;$('#tuner-frequency').value=(point.frequencyHz/1e6).toFixed(6);$('#tuner-frequency').dataset.pending='true';$('#tuner-readout').textContent=(point.frequencyHz/1e6).toFixed(6);drawSpectrum();if($('#tuner-lock-center').checked&&state.status?.activeProfileID==='quick-tune'){queueReceiverControls();toast(`Software VFO ${(point.frequencyHz/1e6).toFixed(6)} MHz`);}else toast(`Tuner set to ${(point.frequencyHz/1e6).toFixed(6)} MHz`);}
$('#tuner-spectrum').addEventListener('click',setVFOFromPointer);$('#waterfall').addEventListener('click',setVFOFromPointer);
$('#tuner-spectrum').addEventListener('mousemove',event=>{const point=frequencyAtPointer(event);if(!point)return;spectrumCursors.set(event.currentTarget,point);const index=Math.min(state.spectrum.binsDBFS.length-1,Math.max(0,Math.round(point.fraction*(state.spectrum.binsDBFS.length-1)))),db=state.spectrum.binsDBFS[index];$('#tuner-cursor').textContent=`${(point.frequencyHz/1e6).toFixed(6)} MHz · ${db.toFixed(1)} dBFS`;drawSpectrumCanvas(event.currentTarget);});
$('#tuner-spectrum').addEventListener('mouseleave',event=>{spectrumCursors.delete(event.currentTarget);$('#tuner-cursor').textContent='Hover for frequency';drawSpectrumCanvas(event.currentTarget);});
$('#tuner-frequency').addEventListener('keydown',event=>{if(event.key!=='ArrowUp'&&event.key!=='ArrowDown')return;event.preventDefault();const direction=event.key==='ArrowUp'?1:-1,next=Math.max(.001,Number(event.currentTarget.value)+direction*Number($('#tuner-step').value));event.currentTarget.value=next.toFixed(6);event.currentTarget.dataset.pending='true';$('#tuner-readout').textContent=next.toFixed(6);queueReceiverControls();});
$('#tuner-bookmark').addEventListener('click',async()=>{const frequencyHz=Number($('#tuner-frequency').value)*1e6,mode=$('#tuner-mode').value;if(!Number.isFinite(frequencyHz)||frequencyHz<=0)return toast('Enter a valid tuner frequency',true);const label=`${(frequencyHz/1e6).toFixed(6)} MHz ${mode.toUpperCase()}`,profile={schemaVersion:1,id:crypto.randomUUID(),name:`Saved · ${label}`,summary:'Saved from the GP-SDR tuner',ranges:[],channels:[{id:crypto.randomUUID(),name:label,frequencyHz,bandwidthHz:Number($('#tuner-bandwidth').value)*1000,mode,decoder:null,enabled:true,priority:5}],deviceAssignments:[{id:crypto.randomUUID(),deviceID:null,role:'channelBank',target:'Saved channel'}],p25Systems:[],settings:{noiseMarginDB:8,revisitSeconds:20,recordAudio:true,recordIQForUnknown:true,transcribeVoice:false,maxRecordingDays:30},builtIn:false};try{await api('/api/profiles',{method:'POST',body:JSON.stringify(profile)});toast('Channel saved to Profiles');await refreshAll();}catch(error){toast(error.message,true);}});
$('#tuner-history').addEventListener('change',event=>{const item=tunerHistory.find(entry=>String(entry.frequencyHz)===event.target.value);if(!item)return;$('#tuner-frequency').value=(item.frequencyHz/1e6).toFixed(6);$('#tuner-readout').textContent=(item.frequencyHz/1e6).toFixed(6);$('#tuner-mode').value=item.mode;$('#tuner-bandwidth').value=item.bandwidthHz/1e3;drawSpectrum();});
for(const [id,key] of [['display-floor','floor'],['display-ceiling','ceiling']]){const control=$('#'+id);control.value=displayPrefs[key];control.addEventListener('change',()=>{displayPrefs[key]=Number(control.value);saveDisplayPrefs();});}
$('#display-peak-reset').addEventListener('click',()=>{[$('#spectrum'),$('#tuner-spectrum'),$('#mapper-spectrum')].forEach(canvas=>spectrumPeaks.delete(canvas));drawSpectrum();toast('Peak hold reset');});
renderTunerHistory();
applyMasterAudio(false);
saveDisplayPrefs();
