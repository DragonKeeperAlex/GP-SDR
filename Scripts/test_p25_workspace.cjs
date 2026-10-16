const fs=require('fs'),vm=require('vm'),assert=require('node:assert/strict');
const source=fs.readFileSync(require('path').join(__dirname,'../server/web/app.js'),'utf8');
function section(start,end){return source.slice(source.indexOf(start),source.indexOf(end,source.indexOf(start)));}
const inherited={id:'assignment',role:'voice-1',deviceID:'rtl',target:'site-west',custom:'keep'};
const state={editingProfile:{deviceAssignments:[inherited],settings:{p25Receivers:{rtl:{gainDB:36}}}},p25Recordings:[],p25RecordingSearch:'old',p25RecordingsBusy:false,p25RecordingsNext:'',devices:[]};
const row={dataset:{id:'assignment'},querySelector:selector=>({value:({'[data-key="role"]':'voice','[data-key="deviceID"]':'rtl','[data-key="target"]':'site-west'})[selector]})};
const controls={'#profile-name':{value:'Saved'},'#profile-summary':{value:'test'},'#record-audio':{checked:true},'#record-iq':{checked:false},'#transcribe-voice':{checked:true},'#recording-retention':{value:'30'},'#p25-sample-rate':{value:'0'}};
const requests=[];
const context={state,$:id=>controls[id],$$:selector=>selector==='#assignments-editor .editor-row'?[row]:[],api:url=>new Promise((resolve,reject)=>requests.push({url,resolve,reject})),updateP25RecordingsView(){},console};
vm.createContext(context);
vm.runInContext(section('function collectEditor()','\nfunction downloadJSON'),context);
const saved=context.collectEditor();assert.equal(saved.deviceAssignments[0].target,'site-west');assert.equal(saved.deviceAssignments[0].custom,'keep');assert.equal(saved.settings.p25Receivers.rtl.gainDB,36);
vm.runInContext(section('let p25RecordingsRequest=0;','\nfunction updateP25RecordingsView'),context);
(async()=>{
 const first=context.refreshP25Recordings();state.p25RecordingSearch='new';const second=context.refreshP25Recordings();
 requests[1].resolve({calls:[{id:'new'}],nextBefore:'new'});await second;
 requests[0].resolve({calls:[{id:'stale'}]});await first;assert.equal(state.p25Recordings[0].id,'new');
 const older=context.refreshP25Recordings(true);assert.match(requests[2].url,/before=new/);requests[2].resolve({calls:[{id:'new'},{id:'old'}]});await older;
 assert.deepEqual(Array.from(state.p25Recordings,x=>x.id),['new','old']);assert.equal(state.p25RecordingsExpanded,true);
 const failed=context.refreshP25Recordings();requests[3].reject(new Error('offline'));await failed;assert.match(state.p25RecordingsError,/offline/);assert.equal(state.p25RecordingsBusy,false);assert.equal(state.p25Recordings.length,2);
 console.log('P25 assignment preservation, tuning preservation, stale-search rejection, cursor paging and retry checks passed');
})().catch(error=>{console.error(error);process.exitCode=1});
// Capability rendering must not expose HackRF gain controls on Pluto or RTL.
Object.assign(context,{escapeHTML:s=>String(s||''),receiverLabel:d=>d.name});
vm.runInContext(section('function p25RateOptions(','\nasync function startP25Workspace'),context);
vm.runInContext(section('function renderP25ReceiverSetup(','\nfunction collectP25ReceiverSettings'),context);
state.devices=[{id:'hackrf',name:'HackRF',kind:'HackRF',sampleRateLimit:20000000},{id:'pluto',name:'Pluto',kind:'PlutoSDR',sampleRateLimit:20000000},{id:'rtl',name:'RTL',kind:'RTL-SDR',sampleRateLimit:2400000}];
Object.assign(state,{p25DeviceID:'',p25ReceiverIDs:[],p25TuningDrafts:{}});
const profile={id:'site',deviceAssignments:state.devices.map(device=>({deviceID:device.id})),settings:{}};
for(const device of state.devices){
 state.p25TuningDeviceID=device.id;
 const html=context.renderP25ReceiverSetup(profile,{state:'ready'});
 assert(html.includes('data-p25-setting="ppmCorrection"'));
 if(device.kind==='HackRF'){assert(html.includes('value="8000000"'));assert(!html.includes('value="5000000"'));assert(html.includes('data-p25-setting="lnaGainDB"'));}
 else{if(device.kind==='RTL-SDR')assert(!html.includes('value="2880000"'));assert(html.includes('data-p25-setting="gainDB"'));assert(!html.includes('data-p25-setting="ampEnabled"'));assert(!html.includes('data-p25-setting="vgaGainDB"'));}
}
console.log('P25 receiver capability and HackRF minimum UI checks passed');
vm.runInContext(section('function renderP25Diagnostics(','\nfunction p25RateOptions'),context);
Object.assign(context,{timeAgo:()=> '20 seconds ago',formatFrequency:value=>String(value),escapeHTML:value=>String(value??'').replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;')});
const diagnostics=context.renderP25Diagnostics({state:'running',telemetryState:'available',receivers:[{state:'searching'}],systems:[{system:'<site>',controlMessages:0,voiceUpdates:58,lastControlAt:'2026-10-15'}]});
assert(diagnostics.includes('No recent control decode'));assert(diagnostics.includes('startup preference'));assert(diagnostics.includes('Voice frequency updates'));assert(diagnostics.includes('not unique calls'));assert(diagnostics.includes('&lt;site&gt;'));assert(!diagnostics.includes('<site>'));
assert(context.renderP25Diagnostics({state:'error'}).includes('decoder exited'));
assert(context.renderP25Diagnostics({state:'running',telemetryState:'unavailable'}).includes('does not establish RF failure'));
const receiverHTML=context.renderP25ReceiverStatus({receivers:[{name:'RTL',state:'idle',frequencyErrorHz:26}]});
assert(receiverHTML.includes('26 Hz'));assert(receiverHTML.includes('No current receiver attribution'));assert(!receiverHTML.includes('None this session'));
console.log('P25 loss, recovery, counter scope and frequency-offset diagnostics checks passed');
Object.assign(context,{recordingPlayer:{dataset:{},paused:true}});
vm.runInContext(section('function renderP25RecordingCards(','\nlet p25RecordingsRequest'),context);
state.p25Recordings=[{id:'missing',label:'Dispatch',audioAvailable:false,mediaIssues:['Missing recording'],durationSeconds:0}];
let cards=context.renderP25RecordingCards();assert(cards.includes('Audio unavailable'));assert(cards.includes('button disabled'));assert(cards.includes('Duration unknown'));
state.p25Recordings=[{id:'saved',audioAvailable:true,audioPath:'fixture.wav',recordedDurationSeconds:2,durationSeconds:.001,transcriptionState:'failed',transcriptionError:'model offline'}];
cards=context.renderP25RecordingCards();assert(cards.includes('2.0 s'));assert(cards.includes('Retry transcription'));assert(cards.includes('model offline'));
console.log('P25 archive missing-audio, measured duration and retry presentation checks passed');
