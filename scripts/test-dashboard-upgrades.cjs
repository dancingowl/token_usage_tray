// Pure JavaScript regression checks. No browser, user settings, or network access.
'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const source = fs.readFileSync(path.join(__dirname, '..', 'dashboard.go'), 'utf8');
const dashboard = source.match(/const dashboardJS = `([\s\S]*?)`\s*$/)[1];
const section = (start, end) => dashboard.slice(dashboard.indexOf(start), dashboard.indexOf(end, dashboard.indexOf(start)));

async function main() {
  const metadata = vm.createContext({ assert });
  vm.runInContext(`
    const elements = Object.fromEntries(['app-update-message','app-update-banner','reload-app'].map(id=>[id,{}]));
    const $ = id => elements[id];
    let pendingSettings=false,settingsBusy=false,mutationCount=0,metadataBusy=false,reloadPending=false;
    function hasPendingSettings(){return pendingSettings;}
    let knownInstance='old',knownUI='ui',themeBusy=false,themeRequest=0;
    let reloads=0,theme='system';
    const document={hidden:false},location={reload(){reloads++;}},window={applyUsageTrayTheme(value){theme=value;}};
    function adoptTheme(value){window.applyUsageTrayTheme(value);}
    let response={instance_id:'new',ui_revision:'ui',theme:'dark'};
    async function api(){return response;}
    ${section('function showReloadNotice(', "$('reload-app').addEventListener")}
  `, metadata);
  await vm.runInContext(`(async()=>{
    pendingSettings=true;await checkAppMetadata();
    assert.equal(reloads,0);assert.equal(elements['reload-app'].disabled,true);assert.equal(reloadPending,true);
    assert.equal(theme,'dark','saved theme is independent of pending settings');
    pendingSettings=false;settingsBusy=true;await checkAppMetadata();assert.equal(reloads,0);
    settingsBusy=false;mutationCount=1;await checkAppMetadata();assert.equal(reloads,0);
    mutationCount=0;await checkAppMetadata();assert.equal(reloads,1);assert.equal(theme,'dark');
    response={instance_id:'old',ui_revision:'new-ui',theme:'light'};await checkAppMetadata();assert.equal(reloads,2);
    themeBusy=true;response.theme='dark';await checkAppMetadata();assert.equal(theme,'light');themeBusy=false;
    api=async()=>{themeRequest++;return response;};await checkAppMetadata();assert.equal(theme,'light','old metadata cannot replace a newer theme action');
    api=async()=>{throw new Error('server restarting');};await checkAppMetadata();assert.equal(metadataBusy,false);
  })()`, metadata);
  console.log('PASS restart detection, pending mutation protection, UI revision, retry');

  const themeActions = vm.createContext({ assert });
  vm.runInContext(`
    let themePreference='system',themeBusy=false,settingsBusy=false,themeRequest=0,mutationCount=0,settingsDirty=true;
    let savedSettings={poll_seconds:60},draft={poll_seconds:180},calls=[],focused=0,lastMessage='';
    const document={activeElement:null,body:{}};const trigger={focus(){focused++;document.activeElement=this;}};const $=()=>trigger;
    function closeThemeMenu(restore){if(restore)trigger.focus();}function renderThemeControl(){}function adoptTheme(value){themePreference=value;}
    function message(id,text){lastMessage=text;}
    let fail=false;async function api(path,method,body){calls.push({path,method,body});if(fail)throw new Error('save failed');return body;}
    ${section('async function saveTheme(', "$('theme-toggle').addEventListener")}
  `, themeActions);
  await vm.runInContext(`(async()=>{
    await saveTheme('dark');assert.equal(themePreference,'dark');assert.equal(calls.length,1);assert.equal(calls[0].path,'/api/theme');
    assert.deepEqual(Object.keys(calls[0].body),['theme']);assert.equal(settingsDirty,true);assert.equal(draft.poll_seconds,180);assert.equal(savedSettings.poll_seconds,60);
    assert.equal(mutationCount,0);assert.equal(themeBusy,false);await saveTheme('dark');assert.equal(calls.length,1);
    fail=true;await saveTheme('light');assert.equal(themePreference,'dark');assert.match(lastMessage,/저장하지 못/);assert.equal(themeBusy,false);assert.equal(mutationCount,0);
    settingsBusy=true;fail=false;await saveTheme('system');assert.equal(calls.length,3,'theme remains usable while settings load');
    settingsBusy=false;fail=false;themePreference='system';calls=[];
    const input={};api=async(path,method,body)=>{calls.push({path,method,body});document.activeElement=input;if(method==='PUT'){themeRequest++;themePreference='light';return {theme:'dark'};}return {theme:'light'};};
    await saveTheme('dark');assert.equal(themePreference,'light','late PUT must re-read a newer preference from another tab');assert.equal(calls.length,2);assert.equal(document.activeElement,input,'save completion must not steal editing focus');
  })()`, themeActions);
  console.log('PASS independent theme save, unrelated settings preservation and failed save');

  const refresh = vm.createContext({ assert, setTimeout: callback => callback() });
  vm.runInContext(`
    let refreshWatcher=0,snapshots=0,histories=0,busy=false,lastMessage='';const page='usage',compact=false;
    let jobs=[{refresh:{id:1,phase:'running',message:'running'}},{refresh:{id:2,phase:'completed',message:'completed'}}];
    async function loadStatus(){assert.ok(jobs.length,'must finish when another tab supersedes the job');return jobs.shift();}
    function refreshBusy(value){busy=value;}function message(id,text){lastMessage=text;}function timeLabel(value){return value;}
    async function refreshSnapshots(){snapshots++;}async function loadHistory(){histories++;}
    ${section('async function watchRefresh(', "document.querySelectorAll('[data-action]')")}
  `, refresh);
  await vm.runInContext(`(async()=>{
    await watchRefresh(1);assert.equal(busy,false);assert.equal(snapshots,1);assert.equal(histories,1);assert.equal(lastMessage,'completed');
    jobs=[{refresh:{id:3,phase:'waiting',message:'waiting',next_at:'next-time'}}];await watchRefresh(3);
    assert.match(lastMessage,/next-time/);assert.equal(snapshots,2);assert.equal(busy,false);
    jobs=[null];await assert.rejects(()=>watchRefresh(4),/수집 상태/);assert.equal(busy,false);
  })()`, refresh);
  console.log('PASS superseded refresh, waiting result, connection failure, button unlock');

  const preferences = vm.createContext({ assert });
  vm.runInContext(`
    const names={claude:'Claude',codex:'Codex',antigravity:'Antigravity'};
    const elements={'history-agent':{value:'codex'},'history-window':{value:'week',options:[]},'history-days':{value:'7'}};
    const $=id=>elements[id];let historyZoom=true,historySelectionTouched=false,historySelectionInitialized=false;
    let saved=null;const localStorage={getItem(){return saved;},setItem(key,value){saved=value;}};
    ${section('function storageRead(', 'restoreHistoryView();')}
    saved=JSON.stringify({agent:'claude',window:'short',days:30,zoom:false});restoreHistoryView();
    assert.equal(elements['history-agent'].value,'claude');assert.equal(elements['history-days'].value,'30');assert.equal(historyZoom,false);
    assert.equal(historySelectionTouched,true);saveHistoryView();assert.equal(JSON.parse(saved).window,'short');
    saved='invalid JSON';restoreHistoryView();assert.equal(elements['history-agent'].value,'claude');
    localStorage.getItem=()=>{throw new Error('blocked storage');};localStorage.setItem=localStorage.getItem;
    restoreHistoryView();saveHistoryView();
  `, preferences);
  console.log('PASS history selection restoration and unavailable browser storage');

  function tokenPage(search, saved) {
    const elements = Object.fromEntries(['token-days','token-model','token-project','token-reset','token-filter-form'].map(id => [id, {
      value: id === 'token-days' ? '14' : '', listeners: {},
      addEventListener(type, callback) { this.listeners[type] = callback; },
      requestSubmit() { this.submitted = true; },
    }]));
    const state = { saved, redirected: '', assigned: '' };
    const context = vm.createContext({ URLSearchParams, page: 'tokens',
      $: id => elements[id],
      location: { search, replace(url) { state.redirected = url; }, assign(url) { state.assigned = url; } },
      storageRead: () => state.saved, storageWrite: (key, value) => { state.saved = value; },
    });
    vm.runInContext(section("const tokenForm=$('token-filter-form');", "if(page==='settings'){loadSettings();"), context);
    return { elements, state };
  }
  const tokens = tokenPage('?days=7&model=chosen&all=1', { days: '30', model: 'old', project: 'old', all: false });
  assert.equal(tokens.state.redirected, '', 'explicit URLs override stored choices');
  tokens.elements['token-days'].value = '7';tokens.elements['token-model'].value = 'chosen';
  tokens.elements['token-filter-form'].listeners.change();
  assert.equal(tokens.elements['token-filter-form'].submitted, true);
  assert.equal(tokens.state.saved.days, '7');assert.equal(tokens.state.saved.model, 'chosen');
  tokens.elements['token-reset'].listeners.click();
  assert.equal(tokens.state.assigned, '/tokens?days=14');
  assert.equal(JSON.stringify(tokens.state.saved), JSON.stringify({ days: '14', model: '', project: '' }));
  assert.equal(tokenPage('', tokens.state.saved).state.redirected, '', 'reset remains effective on revisit');
  assert.equal(tokenPage('', { days: '30', model: 'a+b', project: 'space project', all: true }).state.redirected,
    '/tokens?days=30&model=a%2Bb&project=space+project', 'restore filters without obsolete all flag');
  assert.equal(tokenPage('', null).state.redirected, '');
  console.log('PASS immediate token filters, explicit URLs, saved filters and persistent reset');

  const feedback = vm.createContext({ assert });
  vm.runInContext(`
    const elements={'settings-message':{classList:{toggle(){}},textContent:''},'settings-feedback':{hidden:true}};
    const $=id=>elements[id];const failedSettings=new Map();let invalid=null;
    const form={querySelector(){return invalid;}};function updateSettingsControls(){}
    ${section('function message(', 'function textNode(')}
    ${section('function settingsStatus()', 'function renderPauseState(')}
    settingsStatus();assert.equal(elements['settings-feedback'].hidden,true);
    failedSettings.set('poll_seconds',{});settingsStatus();assert.equal(elements['settings-feedback'].hidden,false);
    assert.match(elements['settings-message'].textContent,/다시 시도/);
    failedSettings.clear();invalid={};settingsStatus();assert.equal(elements['settings-feedback'].hidden,false);
    invalid=null;settingsStatus();assert.equal(elements['settings-feedback'].hidden,true);
    assert.equal(elements['settings-message'].textContent,'');
  `, feedback);
  console.log('PASS settings feedback appears only for failed saves or invalid input');
}
main().catch(error => { console.error(error);process.exitCode=1; });
