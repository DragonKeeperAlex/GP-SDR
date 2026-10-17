const fs=require('fs'),vm=require('vm'),assert=require('node:assert/strict');
const source=fs.readFileSync(require('path').join(__dirname,'../server/web/app.js'),'utf8');
const line=source.split('\n').find(x=>x.startsWith("$('#local-ai-form').addEventListener('submit'"));
let submit,body;
const state={localAI:{config:{remoteEnabled:true,remoteEndpoint:'http://home-pc.local:11434',remoteModel:'remote-model',remoteMinimumConfidence:55}}};
const form={addEventListener:(event,callback)=>submit=callback};
const values={'#local-ai-enabled':{checked:true},'#local-ai-profile':{value:'lightweight'},'#local-ai-model':{value:'pi-model'},'#local-ai-context':{value:'2048'},'#local-ai-endpoint':{value:'http://127.0.0.1:11434'},'#local-ai-confidence':{value:'55'}};
const context={state,$:id=>id==='#local-ai-form'?form:values[id],api:async(url,options)=>{body=JSON.parse(options.body);return state.localAI},renderLocalAI(){},toast(){}};vm.createContext(context);vm.runInContext(line,context);
(async()=>{await submit({preventDefault(){}});assert.equal(body.remoteEnabled,true);assert.equal(body.remoteEndpoint,'http://home-pc.local:11434');assert.equal(body.remoteModel,'remote-model');assert.equal(body.model,'pi-model');console.log('Local settings preserve configured home-compute settings');})().catch(error=>{console.error(error);process.exitCode=1});

const computeLine=source.split('\n').find(x=>x.startsWith("$('#compute-mode').addEventListener('change'"));
let change,selection;
const control={value:'local',disabled:false,addEventListener:(event,fn)=>change=fn};
const computeState={localAI:{config:{computeMode:'auto',remoteEndpoint:'http://192.168.1.2:11435',remoteModel:'home'}}};
const computeContext={state:computeState,$:()=>control,api:async(url,options)=>{assert.equal(url,'/api/local-ai/compute');selection=JSON.parse(options.body);return {config:{...computeState.localAI.config,computeMode:selection.mode}}},renderLocalAI(){},toast(){}};
vm.createContext(computeContext);vm.runInContext(computeLine,computeContext);
(async()=>{await change({target:control});assert.deepEqual(selection,{mode:'local'});assert.equal(control.disabled,false);assert.equal(computeState.localAI.config.remoteModel,'home');control.value='deferred';computeContext.api=async()=>{throw Error('offline')};await change({target:control});assert.equal(control.value,'local');assert.equal(control.disabled,false);console.log('Compute selector saves only the mode and restores selection on failure');})().catch(error=>{console.error(error);process.exitCode=1});
