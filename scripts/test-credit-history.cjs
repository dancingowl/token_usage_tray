// Exercise the shipped credit helpers, chart renderer, tooltip and selection flow.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const source = fs.readFileSync(path.join(__dirname, '../dashboard.go'), 'utf8');
const section = (start,end) => source.slice(source.indexOf(start),source.indexOf(end,source.indexOf(start)));
new vm.Script(source.slice(source.indexOf('const dashboardJS = `')+'const dashboardJS = `'.length,source.lastIndexOf('`')));
class Element {
 constructor(tag='div'){this.tag=tag;this.children=[];this.attrs={};this.listeners={};this.style={};this.dataset={};this.hidden=false;this.clientWidth=560;this.clientHeight=260;this.offsetWidth=180;this.offsetHeight=80;this.value='';this.textContent='';this.classList={remove(){},toggle(){}};}
 append(...children){for(const child of children){child.parent=this;this.children.push(child);}}
 replaceChildren(...children){this.children=[];this.textContent='';this.append(...children);}
 setAttribute(key,value){this.attrs[key]=String(value);if(key==='data-at')this.dataset.at=value;}
 addEventListener(type,callback){this.listeners[type]=callback;}
 contains(el){return this===el || this.children.some(child=>child.contains(el));}
 get options(){return this.children;}
 remove(){this.parent.children=this.parent.children.filter(child=>child!==this);}
 focus(){document.activeElement=this;if(this.listeners.focus)this.listeners.focus();}
 scrollIntoView(){this.scrolled=true;}
 getScreenCTM(){return {inverse(){return this;}};}
 getBoundingClientRect(){return {left:0,top:0};}
}
const document={activeElement:null};
const elements=Object.fromEntries(['history-context','history-agent','history-window','history-days','history-chart','history-overview','history-zoom','history-view-label','history-message','usage-plan','usage-plan-context','credit-history-summary','forecast-state','history-details','history-title'].map(id=>[id,new Element()]));
const $=id=>elements[id];
function textNode(tag,text,cls){const e=new Element(tag);e.textContent=text;e.className=cls;return e;}
function svgElement(tag,attrs){const e=new Element(tag);for(const [key,value] of Object.entries(attrs))e.setAttribute(key,value);return e;}
const DOMPoint=class{constructor(x,y){this.x=x;this.y=y;}matrixTransform(){return this;}};
const context=vm.createContext({assert,document,$,textNode,svgElement,DOMPoint,timeLabel:String,message:(id,text)=>$(id).textContent=text,localStorage:{getItem(){return null;},setItem(){}},names:{codex:'Codex',claude:'Claude',antigravity:'Antigravity'},compact:false,page:'usage'});
vm.runInContext(`let historyData=null,historyZoom=true,historySelectionTouched=false,historySelectionInitialized=false,dismissHistoryDetails=()=>{};${section('function storageRead(','restoreHistoryView();')}${section('function historyInterval(','async function loadHistory()')}`,context);
const {historyBins,historyScale,creditHistoryChanges,creditNumber,historyValid,historyWindowAllowed,syncHistoryWindows,drawHistory}=context;
const minute=60000,at=Date.UTC(2026,9,26,12);
const credit=(offset,balance,account='a',source='live')=>({agent:'codex',window:'credits',at:new Date(at+offset*minute).toISOString(),balance,credit_account:account,source,left:99});
assert.equal(creditNumber(60087.1),'60,087.10');assert.equal(creditNumber(0),'0.00');
for(const value of [null,undefined,-1,Infinity,NaN,'3'])assert.equal(historyValid(credit(0,value)),false);
assert.equal(historyValid(credit(0,0)),true);assert.equal(historyValid(credit(0,3,'a','unavailable')),false);
const input=Object.freeze([credit(0,100.25),credit(1,80.1),credit(2,null),credit(3,60.1),credit(4,50.1),credit(5,100.1,'b'),credit(6,120.2,'b')].map(Object.freeze));
const before=JSON.stringify(input),bins=historyBins(input,10*minute);
assert.deepEqual(Array.from(bins,b=>[b.s.balance,b.segment]),[[80.1,0],[50.1,1],[120.2,2]]);
assert.equal(JSON.stringify(input),before);
const changes=creditHistoryChanges(input,10*minute);assert.ok(Math.abs(changes.decrease-30.15)<1e-8);assert.ok(Math.abs(changes.increase-20.1)<1e-8);assert.equal(changes.pairs,3);
assert.equal(creditHistoryChanges([credit(0,100),credit(40,10)]).pairs,0);
for(const interval of [minute,10*minute,4*60*minute]){const totals=creditHistoryChanges(input,interval);assert.equal(totals.decrease,changes.decrease);assert.equal(totals.increase,changes.increase);}
assert.equal(historyBins([credit(0,100),credit(31,90)],4*60*minute).length,2);
assert.equal(historyValid(credit(0,3,'a','cache')),false);
assert.equal(historyBins([credit(0,100),credit(1,90,'b'),credit(2,80,'a')],10*minute).length,3);
assert.equal(historyBins([credit(0,null),credit(1,0)],minute)[0].s.balance,0);
for(const values of [[],[0],[.01],[60087.1],[Number.MAX_VALUE]]){
 const samples=values.map((value,i)=>credit(i,value)),scale=historyScale(samples,true);assert.ok(Number.isFinite(scale.max)&&scale.max>0);
 for(const tick of scale.ticks){assert.ok(Number.isFinite(tick.value));assert.ok(context.creditAxisLabel(tick.value).length<15);}for(const sample of samples)assert.ok(Number.isFinite(scale.ratio(sample)));
 assert.equal(scale.ticks[0].value,0);
}
function descendants(e){return [e,...e.children.flatMap(descendants)];}
function run(samples,zoom=true){
 $('history-agent').value='codex';$('history-window').value='credits';
 context.input={samples,forecasts:[{agent:'codex',window:'credits',ready:true}],displayDays:7,rangeEnd:at+60*minute};
 context.zoom=zoom;vm.runInContext('historyData=input;historyZoom=zoom;drawHistory();',context);
 const nodes=descendants($('history-chart'));
 for(const node of nodes)for(const key of ['cx','cy','x','y'])if(node.attrs[key]!==undefined)assert.ok(Number.isFinite(Number(node.attrs[key])),key+' must be finite');
 assert.equal($('usage-plan').hidden,true);assert.equal($('credit-history-summary').hidden,false);assert.equal($('forecast-state').children.length,0);
 assert.ok(nodes.filter(node=>node.tag==='text').every(node=>!node.textContent.includes('%')));
 return nodes;
}
run(input,true);const zoomSummary=$('credit-history-summary').textContent;
let nodes=run(input,false),dots=nodes.filter(node=>node.attrs.class==='dash-chart-point');
assert.equal($('credit-history-summary').textContent,zoomSummary,'zoom changes may not alter credit totals');
assert.equal($('history-context').textContent,'이력 · 잔액 변화');assert.ok(dots.length>=3);assert.ok(dots.every(dot=>dot.attrs.tabindex==='0' && dot.attrs['aria-label'].includes('크레딧') && !dot.attrs['aria-label'].includes('초기화')));
assert.match($('history-message').textContent,/6개 기록/);assert.match($('credit-history-summary').textContent,/감소량 30.15 크레딧 · 증가량 20.10/);
assert.equal(nodes.filter(node=>node.tag==='polyline').length,0,'binning may not reconnect failure or account boundaries');
dots[0].focus();const tip=descendants($('history-chart')).find(node=>node.id==='history-tooltip');assert.equal(tip.hidden,false);assert.match(tip.children[1].textContent,/크레딧/);assert.ok(tip.children.every(node=>!node.textContent.includes('초기화')));
dots[0].listeners.blur();assert.equal(tip.hidden,true);
const chartSVG=$('history-chart').children[0];chartSVG.listeners.pointerdown({clientX:Number(dots[0].attrs.cx),clientY:Number(dots[0].attrs.cy),pointerType:'touch'});assert.equal(tip.hidden,false);
chartSVG.listeners.pointermove({clientX:0,clientY:0,pointerType:'mouse'});assert.equal(tip.hidden,true);
for(const samples of [[],[credit(0,null)],[credit(0,0)],[credit(0,.01)],[credit(0,Number.MAX_VALUE)]])run(samples);
assert.match($('history-message').textContent,/기록이 1개/);
nodes=run([credit(0,100),credit(1,90),credit(2,null),credit(3,null)]);assert.ok(nodes.some(node=>node.attrs.class==='dash-chart-point'),'latest failure preserves older numeric records');
nodes=run([credit(0,100),credit(1,90),credit(40,50)],false);assert.equal(nodes.filter(node=>node.tag==='polyline').length,0,'empty time buckets remain disconnected');
// Switching back to quota restores percent chart, plan and tooltip reset metadata.
context.input={samples:[{agent:'codex',window:'week',at:new Date(at).toISOString(),left:80,source:'live'}],forecasts:[],displayDays:7,rangeEnd:at+minute};
$('history-window').value='week';vm.runInContext('historyData=input;drawHistory();',context);assert.equal($('usage-plan').hidden,false);assert.equal($('credit-history-summary').hidden,true);assert.ok(descendants($('history-chart')).some(node=>node.textContent==='100%'));
// Credit option is added only for Codex; quota choices survive agent changes.
$('history-window').replaceChildren(...['short','week'].map(value=>{const e=new Element('option');e.value=value;return e;}));
$('history-agent').value='codex';syncHistoryWindows('short');syncHistoryWindows('credits');assert.ok($('history-window').options.some(option=>option.value==='credits'));
$('history-agent').value='claude';syncHistoryWindows();assert.equal($('history-window').value,'week');assert.ok(!$('history-window').options.some(option=>option.value==='credits'));
$('history-agent').value='codex';vm.runInContext("syncHistoryWindows(historyQuotaWindows.codex)",context);assert.equal($('history-window').value,'short');
assert.equal(historyWindowAllowed('claude','credits'),false);assert.equal(historyWindowAllowed('codex','credits'),true);
console.log('PASS credit numeric scale, 0/decimal/missing/large balance, bins, gaps, account changes, increases, counts, renderer, hover/touch/keyboard, quota switching and item selection');

assert.equal(creditHistoryChanges([credit(0,100,''),credit(1,90,'')]).pairs,0,'unidentified accounts cannot be compared');
nodes=run([credit(0,100),credit(5,90),credit(10,80)]);
assert.equal(nodes.filter(node=>node.tag==='polyline').length,1,'poll cadence wider than zoom bins must still show the line');
vm.runInContext(`let historyRequest=0;const api=async()=>{throw new Error('fixture disconnected');};${section('async function loadHistory()',"$('history-agent').addEventListener('change'")}`,context);
$('history-days').value='30';$('credit-history-summary').textContent='previous period total';$('credit-history-summary').hidden=false;
vm.runInContext('loadHistory()',context).then(()=>{
 assert.equal($('credit-history-summary').hidden,true,'failed new period must hide previous totals');
 assert.equal($('credit-history-summary').textContent,'');
 $('history-agent').value='claude';$('history-window').value='week';drawHistory();
 assert.equal($('usage-plan').hidden,false,'selection context updates even with no loaded data');
 console.log('PASS period-request failure clears old totals and keeps selection context accurate');
}).catch(error=>{console.error(error);process.exitCode=1;});
