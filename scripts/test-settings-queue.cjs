// Pure queue regression checks: no browser, settings files, registry or network.
'use strict';
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const source=fs.readFileSync(require('node:path').join(__dirname,'..','dashboard.go'),'utf8');
const script=source.match(/const dashboardJS = `([\s\S]*?)`\s*$/)[1];
const section=(start,end)=>script.slice(script.indexOf(start),script.indexOf(end,script.indexOf(start)));
const context=vm.createContext({assert,setTimeout,clearTimeout});
vm.runInContext(`
const settingsQueue=new Map(),failedSettings=new Map(),numberTimers=new Map(),settingVersions=new Map(),settingSignatures=new Map();
let settingsSaving=false,settingsApplied=false,mutationCount=0,savedSettings={},savedStartup=null,pausedUntil=0;
const names={claude:'Claude'},ruleNames={claude5h:'Claude 5h'};
const form={addEventListener(){}};const $=()=>({hidden:false});let statuses=0;function settingsStatus(){statuses++;}function renderPauseState(){}
class HTMLInputElement {
 constructor(name,value,type='number'){this.name=name;this.value=value;this.type=type;this.attributes={};this.checked=false;}
 checkValidity(){return this.type!=='number' || this.value!=='' && Number.isFinite(Number(this.value)) && Number(this.value)>=1 && Number(this.value)<=(this.name==='poll_minutes'?60:99) && (this.name==='poll_minutes' || Number.isInteger(Number(this.value)));}
 setAttribute(key,value){this.attributes[key]=value;}removeAttribute(key){delete this.attributes[key];}
}
class HTMLSelectElement extends HTMLInputElement{}
let calls=[],failKey='',release=null,hold=false;let active=0,maxActive=0;
async function api(path,method,body){calls.push({path,method,body});active++;maxActive=Math.max(maxActive,active);try{if(hold){hold=false;await new Promise(resolve=>release=resolve);}if(failKey && body[failKey]!==undefined){failKey='';throw new Error('fixture failure');}return path==='/api/startup'?{supported:true,...body}:body;}finally{active--;}}
${section('function hasPendingSettings()', 'function updateSettingsControls()')}
${section('function settingPatch(', 'function flushNumberChanges()')}
function flushNumberChanges(){}
${section('function flushClosingSettings()', "window.addEventListener('pagehide'")}
async function settle(){while(hasPendingSettings())await new Promise(resolve=>setTimeout(resolve,5));}
`,context);
async function main(){
 await vm.runInContext(`(async()=>{
  hold=true;queueSetting({key:'display_window',patch:{display_window:'short'}});
  queueSetting({key:'icon_style',patch:{icon_style:'number'}});
  queueSetting({key:'display_window',patch:{display_window:'week'}});
  queueSetting({key:'display_window',patch:{display_window:'urgent'}});
  assert.equal(calls.length,1);release();await settle();
  assert.equal(maxActive,1,'settings and startup mutations must be serialized');
  assert.deepEqual(calls.map(call=>call.body),[{display_window:'short'},{icon_style:'number'},{display_window:'urgent'}]);
  assert.equal(mutationCount,0);assert.equal(failedSettings.size,0);
 })()`,context);
 console.log('PASS serial requests and newest queued edits survive in-flight responses');
 await vm.runInContext(`(async()=>{
  calls=[];hold=true;queueSetting({key:'display_window',patch:{display_window:'short'}});
  queueSetting({key:'display_window',patch:{display_window:'urgent'}});
  queueSetting({key:'icon_style',patch:{icon_style:'donut'}});
  flushClosingSettings();
  assert.deepEqual(calls.map(call=>call.body),[{display_window:'short'}], 'pagehide cannot bypass the active request');
  assert.equal(settingsQueue.get('display_window').patch.display_window,'urgent');
  assert.equal(settingsQueue.has('icon_style'),true,'all later actions stay in the same queue');
  release();await settle();
  assert.deepEqual(calls.map(call=>call.body),[{display_window:'short'},{display_window:'urgent'},{icon_style:'donut'}], 'same leaf remains serialized through teardown');
  calls=[];hold=true;queueSetting({key:'startup',patch:{enabled:true}});
  queueSetting({key:'startup',patch:{enabled:false}});flushClosingSettings();
  assert.equal(calls.length,1,'startup teardown cannot dispatch overlapping state');
  release();await settle();assert.equal(calls.length,2);assert.equal(calls[1].body.enabled,false);
 })()`,context);
 console.log('PASS pagehide never bypasses serial requests for settings or startup');

 await vm.runInContext(`(async()=>{
  calls=[];failKey='notify_limits';queueSetting({key:'notify_limits',patch:{notify_limits:false}});await settle();
  assert.equal(failedSettings.size,1);queueSetting({key:'startup',patch:{enabled:true}});await settle();
  assert.equal(savedStartup.enabled,true);assert.equal(failedSettings.size,1);
  for(const action of Array.from(failedSettings.values()))queueSetting({...action});await settle();
  assert.equal(failedSettings.size,0);assert.deepEqual(calls.map(call=>call.path),['/api/settings','/api/startup','/api/settings']);
  calls=[];hold=true;failKey='display_window';queueSetting({key:'display_window',patch:{display_window:'short'}});
  queueSetting({key:'display_window',patch:{display_window:'week'}});release();await settle();
  assert.equal(failedSettings.size,0,'failure of older action cannot mark newer successful edit failed');
 })()`,context);
 console.log('PASS failed leaf retries separately, independent startup succeeds, old failures ignored');
 await vm.runInContext(`(async()=>{
  calls=[];const input=new HTMLInputElement('poll_minutes','2');
  settingsInput({target:input,type:'input'});assert.equal(hasPendingSettings(),true);assert.equal(calls.length,0);
  input.value='2.5';settingsInput({target:input,type:'input'});await settle();
  assert.deepEqual(calls.map(call=>call.body),[{poll_seconds:150}]);
  settingsInput({target:input,type:'change'});await settle();assert.equal(calls.length,1,'blur/change cannot duplicate a numeric write');
  for(const value of ['', '0', '100', 'invalid']){input.value=value;settingsInput({target:input,type:'input'});await settle();}
  assert.equal(calls.length,1);assert.equal(input.attributes['aria-invalid'],'true');
  const rule=new HTMLInputElement('claude5h_threshold','23.5');settingsInput({target:rule,type:'input'});await settle();assert.equal(calls.length,1);
  rule.value='23';settingsInput({target:rule,type:'input'});settingsInput({target:rule,type:'change'});await settle();
  assert.deepEqual(calls[1].body,{alert_rules:{claude5h:{threshold:23}}});assert.equal(rule.attributes['aria-invalid'],undefined);
 })()`,context);
 console.log('PASS debounced valid numbers, invalid and intermediate input excluded, blur flush deduplicates');
}
main().catch(error=>{console.error(error);process.exitCode=1;});
