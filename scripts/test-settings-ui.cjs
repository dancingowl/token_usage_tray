// Run against a running app: node scripts/test-settings-ui.cjs
// Optional: BASE_URL, CHROME_PATH, PLAYWRIGHT_MODULE_PATH (module or directory), SCREENSHOT_DIR.
// Every API and state request is mocked; no real settings or registry writes occur.
'use strict';
const assert = require('node:assert/strict');
const { createRequire } = require('node:module');
const path = require('node:path');
const fs = require('node:fs');
function playwright() {
  if (process.env.PLAYWRIGHT_MODULE_PATH) {
    const location = path.resolve(process.env.PLAYWRIGHT_MODULE_PATH);
    try { return require(location); } catch (error) {
      try { return createRequire(path.join(location, '__settings_test.cjs'))('playwright'); }
      catch { throw error; }
    }
  }
  try { return require('playwright'); } catch {
    try { return require('playwright-core'); } catch {
      throw new Error('Install playwright or set PLAYWRIGHT_MODULE_PATH; optionally set CHROME_PATH.');
    }
  }
}
const baseURL = process.env.BASE_URL || 'http://127.0.0.1:47113';
const keys = ['claude5h', 'claude7d', 'codex5h', 'codex7d', 'antigravity'];
const clone = value => JSON.parse(JSON.stringify(value));
const baseline = () => ({
  icon_style: 'donut', display_window: 'week', poll_seconds: 180, theme: 'system',
  visible_agents: { claude: true, codex: true, antigravity: false },
  alerts_enabled: true, notify_limits: true,
  alert_rules: Object.fromEntries(keys.map((key, index) => [key, {
    enabled: index % 2 === 0, threshold: 17 + index, recovery: index % 2 !== 0,
  }])),
});
const status = { agents: ['Claude', 'Codex', 'Antigravity'].map((name, index) => ({
  name, status: index ? 'waiting' : 'live', label: index ? '연결된 계정 없음' : '정상 수집',
  source: index ? '' : 'api', last_success: index ? null : '2026-10-15T00:00:00Z',
  age_minutes: index ? -1 : 0, error: '',
})), refreshing: false, scanning: false };
async function fixture(browser, options = {}) {
  const context = await browser.newContext({ viewport: { width: 1000, height: 900 } });
  const page = await context.newPage();
  const state = { settings: baseline(), startup: { supported: true, enabled: false },
    mutations: [], reads: [], errors: [], ...options };
  page.on('pageerror', error => state.errors.push(error.message));
  await context.route('**/*', async route => {
    const request = route.request(), url = new URL(request.url()), method = request.method();
    const reply = (data, code = 200) => route.fulfill({ status: code,
      contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname.startsWith('/api/') || url.pathname === '/state.json') {
      if (method === 'GET') state.reads.push(url.pathname);
      else state.mutations.push({ path: url.pathname, method, body: request.postDataJSON() });
      if (url.pathname === '/api/settings') {
        if (method === 'GET') return reply(state.settings);
        if (method === 'PATCH') {
          if (state.settingsDelay) await new Promise(resolve => setTimeout(resolve, state.settingsDelay));
          if (state.failSettings) { state.failSettings--; return reply({ error: 'fixture settings failure' }, 500); }
          const merge = (target, patch) => { for (const [key, value] of Object.entries(patch)) { if (value && typeof value === 'object') merge(target[key], value); else target[key] = value; } };
          merge(state.settings, request.postDataJSON()); return reply(state.settings);
        }
      }
      if (url.pathname === '/api/theme') {
        if (method === 'GET') return reply({ theme: state.settings.theme });
        if (method === 'PUT') {
          if (state.failTheme) { state.failTheme--; return reply({ error: 'fixture theme failure' }, 500); }
          state.settings.theme = request.postDataJSON().theme; return reply({ theme: state.settings.theme });
        }
      }
      if (url.pathname === '/api/meta') return reply({ instance_id: 'settings-fixture', ui_revision: 'settings-fixture', theme: state.settings.theme });
      if (url.pathname === '/api/startup') {
        if (method === 'GET') return reply(state.startup, state.failStartupRead ? 500 : 200);
        if (method === 'POST') {
          if (state.failStartup) { state.failStartup--; return reply({ error: 'fixture startup failure' }, 500); }
          if (state.mismatchStartup) {
            state.mismatchStartup--;
            return reply({ ...state.startup, enabled: !request.postDataJSON().enabled });
          }
          state.startup.enabled = request.postDataJSON().enabled; return reply(state.startup);
        }
      }
      if (url.pathname === '/api/status') return reply(status);
      if (url.pathname === '/api/auto-update') return reply({ supported: false, state: 'disabled' });
      if (url.pathname === '/api/update') return reply({ current_version: 'dev' });
      if (url.pathname === '/api/history') return reply({ samples: [], forecasts: [] });
      if (url.pathname === '/state.json') return reply({
        claude: { available: false, short: { left: -1, resetAt: 0 }, week: { left: -1, resetAt: 0 } },
        codex: { available: true, short: { left: -1, resetAt: 0 }, week: { left: 75, resetAt: Math.floor(Date.now() / 1000) + 86400 } },
        antigravity: { available: false, short: { left: -1, resetAt: 0 }, week: { left: -1, resetAt: 0 } },
      });
      return reply({ error: 'Unexpected mocked endpoint' }, 404);
    }
    // Only the page and its read-only assets can reach the running server.
    if (method !== 'GET' || url.origin !== new URL(baseURL).origin) return route.abort();
    return route.continue();
  });
  await page.goto(new URL('/settings', baseURL).href);
  await page.waitForFunction(() => {
    const field = document.querySelector('[name="poll_minutes"]');
    return field && !field.disabled && !field.closest('fieldset[disabled]') && field.value !== '';
  });
  await page.waitForFunction(() => !document.querySelector('#startup-state').textContent.includes('확인하고') ||
    !!document.querySelector('#startup-message')?.textContent ||
    document.querySelector('#retry-startup')?.hidden === false);
  return { page, state, context };
}
const field = (page, name) => page.locator(`[name="${name}"]`);
async function settled(page) {
  await page.waitForFunction(() => document.querySelector('#settings-form').getAttribute('aria-busy') === 'false');
}
async function selectTheme(page, theme) {
  await page.locator('#theme-toggle').click();
  await page.locator(`[data-theme-choice="${theme}"]`).click();
  await page.waitForFunction(() => !document.querySelector('#theme-toggle').disabled);
}
async function main() {
  const browser = await playwright().chromium.launch({ headless: true,
    ...(process.env.CHROME_PATH ? { executablePath: process.env.CHROME_PATH } : {}) });
  let passed = 0;
  async function test(name, options, run) {
    const f = await fixture(browser, options);
    try { await run(f); assert.deepEqual(f.state.errors, [], 'uncaught browser errors');
      console.log(`PASS ${name}`); passed++;
    } finally { await f.context.close(); }
  }
  try {
    await test('automatic settings structure and immediate toggles', {}, async ({page,state}) => {
      assert.equal(await page.locator('#save-settings,#reset-settings,#save-startup').count(), 0);
      assert.equal(await page.locator('#settings-form #startup-enabled').count(), 1);
      assert.equal(await page.locator('header #theme-toggle').count(), 1);
      await page.locator('#settings-form').evaluate(form => form.requestSubmit());
      assert.equal(state.mutations.length, 0);
      await field(page,'visible_claude').uncheck();await settled(page);
      await field(page,'display_window').selectOption('urgent');await settled(page);
      await page.locator('#startup-enabled').check();await settled(page);
      assert.deepEqual(state.mutations.map(m=>[m.path,m.method,m.body]),[
        ['/api/settings','PATCH',{visible_agents:{claude:false}}],
        ['/api/settings','PATCH',{display_window:'urgent'}],
        ['/api/startup','POST',{enabled:true}],
      ]);
      assert.match(await page.locator('#auto-update-state').textContent(),/지원하지/);
    });
    await test('valid numbers debounce, invalid edits stay local, blur validates', {}, async ({page,state}) => {
      await field(page,'poll_minutes').fill('2');await field(page,'poll_minutes').fill('2.5');await settled(page);
      assert.deepEqual(state.mutations.map(m=>m.body),[{poll_seconds:150}]);
      await field(page,'claude5h_threshold').fill('100');await settled(page);
      assert.equal(state.mutations.length,1);
      assert.equal(await field(page,'claude5h_threshold').getAttribute('aria-invalid'),'true');
      await field(page,'claude5h_threshold').fill('');await settled(page);assert.equal(state.mutations.length,1);
      await field(page,'claude5h_threshold').fill('23');await field(page,'poll_minutes').focus();await settled(page);
      assert.equal(state.settings.alert_rules.claude5h.threshold,23);
      assert.equal(state.mutations.length,2,'change and blur must not duplicate writes');
      assert.equal(state.settings.alert_rules.claude7d.threshold,18);
    });
    await test('new edits during in-flight request retained and preserve external fields', {settingsDelay:150}, async ({page,state}) => {
      await field(page,'display_window').selectOption('short');
      await page.locator('input[name="icon_style"][value="number"]').check();
      await field(page,'display_window').selectOption('urgent');
      state.settings.visible_agents.antigravity=true;state.settings.theme='dark';
      await settled(page);
      assert.equal(state.settings.display_window,'urgent');assert.equal(state.settings.icon_style,'number');
      assert.equal(await field(page,'display_window').inputValue(),'urgent');
      assert.equal(state.settings.visible_agents.antigravity,true);assert.equal(state.settings.theme,'dark');
      assert.ok(state.mutations.every(m=>Object.keys(m.body).length===1));
    });
    await test('failed leaf retries independently while other changes save', {failSettings:1}, async ({page,state}) => {
      await field(page,'display_window').selectOption('short');await settled(page);
      assert.equal(await field(page,'display_window').inputValue(),'short');
      assert.equal(await page.locator('#retry-settings').isVisible(),true);
      await page.locator('#startup-enabled').check();await settled(page);
      assert.equal(state.startup.enabled,true);assert.equal(state.settings.display_window,'week');
      await page.locator('#retry-settings').click();await settled(page);
      assert.equal(state.settings.display_window,'short');assert.equal(await page.locator('#retry-settings').isVisible(),false);
      assert.deepEqual(state.mutations.map(m=>m.path),['/api/settings','/api/startup','/api/settings']);
    });
    for(const failure of ['failStartup','mismatchStartup'])await test('startup failure or mismatch retries requested state '+failure,{[failure]:1},async({page,state})=>{
      await page.locator('#startup-enabled').check();await settled(page);
      assert.equal(await page.locator('#startup-enabled').isChecked(),true);
      assert.equal(await page.locator('#retry-settings').isVisible(),true);
      await page.locator('#retry-settings').click();await settled(page);assert.equal(state.startup.enabled,true);
    });
    await test('theme independent from number validation and failed settings', {failSettings:1}, async ({page,state})=>{
      await field(page,'display_window').selectOption('short');await settled(page);
      await field(page,'poll_minutes').fill('');
      await selectTheme(page,'dark');assert.equal(state.settings.theme,'dark');
      assert.equal(await field(page,'poll_minutes').inputValue(),'');
      await page.locator('#retry-settings').click();await settled(page);
      assert.equal(state.settings.display_window,'short');assert.equal(state.settings.theme,'dark');
      assert.ok(state.mutations.filter(m=>m.path==='/api/settings').every(m=>!('theme' in m.body)));
    });
    for (const [name, options] of [['startup read failure', { failStartupRead: true }],
      ['unsupported startup', { startup: { supported: false, enabled: false } }]]) {
      await test(name+' leaves settings usable', options, async ({ page, state }) => {
        assert.equal(await page.locator('#startup-enabled').isDisabled(),true);
        await field(page,'display_window').selectOption('short');await settled(page);
        assert.equal(state.settings.display_window,'short');
      });
    }
    await test('disabled notifications keep invalid disconnected rules visible and mobile fits',{},async({page})=>{
      await field(page,'claude5h_threshold').fill('100');await field(page,'alerts_enabled').uncheck();await settled(page);
      assert.equal(await field(page,'claude5h_threshold').isVisible(),true);
      await field(page,'claude5h_threshold').fill('23');await settled(page);
      assert.equal(await page.locator('#notification-options').isVisible(),false);
      for(const width of [485,320]){
        await page.setViewportSize({width,height:900});
        const size=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,width:document.documentElement.clientWidth}));
        assert.ok(size.scroll<=size.width+1,JSON.stringify(size));
      }
    });
    if(process.env.SCREENSHOT_DIR){
      const directory=path.resolve(process.env.SCREENSHOT_DIR);fs.mkdirSync(directory,{recursive:true});
      const {page,context}=await fixture(browser);
      try{for(const width of [1000,320]){await page.setViewportSize({width,height:900});await page.screenshot({path:path.join(directory,`settings-${width}.png`),fullPage:true});}}
      finally{await context.close();}
    }
    console.log(`PASS: ${passed} isolated immediate settings UI scenarios; all API mutations mocked.`);
  } finally { await browser.close(); }
}
main().catch(error => { console.error(error); process.exitCode = 1; });
