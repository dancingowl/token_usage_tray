// Run with CREDIT_UI_FIXTURES_DIR containing isolated rendered pages and history.json.
// Every request is intercepted; no user settings, credentials or running app are touched.
'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE_PATH || 'playwright');
const directory=process.env.CREDIT_UI_FIXTURES_DIR;
if(!directory)throw new Error('Set CREDIT_UI_FIXTURES_DIR to the isolated fixture directory.');
const baseHistory=JSON.parse(fs.readFileSync(path.join(directory,'history.json'),'utf8'));
async function main(){
 const browser=await chromium.launch({headless:true,...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{})});
 async function fixture(scenario='normal',summary=false){
  const context=await browser.newContext({viewport:{width:1100,height:1000}}),page=await context.newPage(),errors=[];
  const html=fs.readFileSync(path.join(directory,scenario+'.html'),'utf8');
  const meta=name=>html.match(new RegExp('<meta name="'+name+'" content="([^"]*)"'))[1];
  let history=structuredClone(baseHistory);
  page.on('pageerror',error=>errors.push(error.message));
  await context.route('**/*',route=>{
   const url=new URL(route.request().url());
   if(url.pathname==='/api/meta')return route.fulfill({json:{instance_id:meta('usage-tray-instance'),ui_revision:meta('usage-tray-ui'),theme:'dark'}});
   if(url.pathname==='/api/history')return route.fulfill({json:history});
   if(url.pathname.startsWith('/api/'))return route.fulfill({status:404,json:{error:'Unexpected fixture request'}});
   return route.fulfill({contentType:'text/html; charset=utf-8',body:html});
  });
  await page.goto('http://127.0.0.1:47124/'+(summary?'?view=summary':''));
  if(!summary)await page.waitForSelector('.dash-history-chart');
  return {context,page,errors,setHistory(value){history=value;}};
 }
 try{
  const f=await fixture(),{page}=f;
  try{
   assert.equal(await page.locator('.card-credit').textContent(),'크레딧 잔액 60,087.10');
   assert.equal(await page.locator('.ct .card-plan').textContent(),' · Pro');
   assert.equal(await page.locator('.card-notes').count(),0);
   const resetBox=await page.locator('.cs').boundingBox(),creditBox=await page.locator('.card-details .card-credit').boundingBox();
   assert.ok(Math.abs(resetBox.y-creditBox.y)<4 && creditBox.x>resetBox.x+resetBox.width,'desktop reset and credit must share the second row');
   await page.screenshot({path:path.join(directory,'quota-card-1100.png'),fullPage:true});
   const previousURL=page.url(),previousItem=await page.locator('#history-window').inputValue();
   await page.locator('.card-credit').click();
   assert.equal(page.url(),previousURL);assert.equal(await page.locator('#history-window').inputValue(),previousItem);
   assert.equal(await page.locator('a.card-credit').count(),0);
   await page.locator('#history-window').selectOption('credits');
   assert.equal(await page.locator('#history-window').inputValue(),'credits');
   assert.equal(await page.locator('#history-agent').inputValue(),'codex');
   assert.equal(await page.locator('#usage-plan').isVisible(),false);
   assert.match(await page.locator('#credit-history-summary').textContent(),/112\.90 크레딧/);
   assert.equal(await page.locator('.dash-history-chart polyline').count(),1,'normal polling gaps must still form a line');
   assert.ok(!(await page.locator('.dash-history-chart').textContent()).includes('%'));
   const point=page.locator('.dash-chart-point').last();await point.focus();
   assert.equal(await page.locator('.history-tooltip').isVisible(),true);
   assert.match(await page.locator('.history-tooltip').textContent(),/60,087\.10 크레딧/);
   await page.keyboard.press('Escape');assert.equal(await page.locator('.history-tooltip').isVisible(),false);
   const box=await point.boundingBox();
   await page.locator('.dash-history-chart').dispatchEvent('pointerdown',{pointerType:'touch',clientX:box.x+box.width/2,clientY:box.y+box.height/2});
   assert.equal(await page.locator('.history-tooltip').isVisible(),true);
   await page.locator('#history-days').selectOption('30');
   await page.waitForFunction(()=>document.querySelector('.dash-history-chart')?.getAttribute('aria-label')?.includes('크레딧'));
   for(const width of [1100,485,320]){
    await page.setViewportSize({width,height:1000});
    assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth+1),'horizontal overflow at '+width);
    await page.screenshot({path:path.join(directory,'credits-'+width+'.png'),fullPage:true});
   }
   await page.locator('#history-agent').selectOption('claude');
   assert.equal(await page.locator('#history-window option[value="credits"]').count(),0);
   assert.equal(await page.locator('#usage-plan').isVisible(),true);
   assert.deepEqual(f.errors,[]);
   console.log('PASS real browser: plain balance, shared graph, numeric axis, changes, keyboard/touch, periods and responsive widths');
  }finally{await f.context.close();}
  for(const [scenario,text] of [['zero','0.00'],['unknown','확인 불가'],['stale','확인 불가'],['unlimited','무제한']]){
   const f=await fixture(scenario);try{
    assert.ok((await f.page.locator('.card-credit').textContent()).includes(text),scenario);
    if(scenario==='zero'){
     const data=structuredClone(baseHistory);
     for(const sample of data.samples)if(sample.window==='credits')sample.balance=0;
     f.setHistory(data);await f.page.locator('#history-window').selectOption('credits');
     await f.page.locator('#history-days').selectOption('1');
     await f.page.waitForFunction(()=>document.querySelector('.dash-chart-point')?.getAttribute('aria-label')?.includes('0.00 크레딧'));
     assert.equal(await f.page.locator('.dash-history-chart polyline').count(),1);
     assert.match(await f.page.locator('#credit-history-summary').textContent(),/감소량 0\.00 크레딧/);
    }
    assert.deepEqual(f.errors,[]);
   }finally{await f.context.close();}
  }
  const compact=await fixture('normal',true);try{
   await compact.page.locator('.card-credit').click();
   assert.equal(new URL(compact.page.url()).search,'?view=summary');
   assert.equal(await compact.page.locator('#history-details').isVisible(),false);
   assert.deepEqual(compact.errors,[]);
  }finally{await compact.context.close();}
  console.log('PASS real browser: zero, missing, stale, unlimited and non-interactive compact summary');
  const multiple=await fixture('multiple');try{
   assert.equal(await multiple.page.locator('.card').count(),3);
   assert.match(await multiple.page.locator('.card-notes').textContent(),/캐시 · 12분 전/);
   for(const width of [1100,894,485,320]){
    await multiple.page.setViewportSize({width,height:1000});
    assert.ok(await multiple.page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth+1),'multiple cards overflow at '+width);
    const reset=await multiple.page.locator('.card[data-agent="codex"] .cs').last().boundingBox();
    const credit=await multiple.page.locator('.card-credit').boundingBox();
    assert.ok(credit.y>=reset.y+reset.height-1 || credit.x>=reset.x+reset.width-1,'credit overlaps reset at '+width);
    if(width===485)await multiple.page.screenshot({path:path.join(directory,'quota-cards-485.png'),fullPage:true});
   }
   assert.deepEqual(multiple.errors,[]);
  }finally{await multiple.context.close();}
  console.log('PASS real browser: two-row card, inline plan, cached warning and multiple-card responsive wrapping');
  const offline=await fixture();try{
   await offline.context.route('**/api/**',route=>route.abort());
   await offline.page.evaluate(()=>{
    const expires=Number(document.querySelector('.card-credit').dataset.creditExpires);
    Date.now=()=>expires+1;document.dispatchEvent(new Event('visibilitychange'));
   });
   assert.match(await offline.page.locator('.card-credit strong').textContent(),/확인 불가/);
  }finally{await offline.context.close();}
  console.log('PASS real browser: offline page expires old balances without a server response');
 }finally{await browser.close();}
}
main().catch(error=>{console.error(error);process.exitCode=1;});
