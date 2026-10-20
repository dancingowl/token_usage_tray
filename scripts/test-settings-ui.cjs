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
        if (method === 'PUT') {
          if (state.failSettings) { state.failSettings--; return reply({ error: 'fixture settings failure' }, 500); }
          state.settings = { ...clone(request.postDataJSON()), theme: state.settings.theme }; return reply(state.settings);
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
async function enabled(page, selector, value) {
  await page.waitForFunction(({ selector, value }) => document.querySelector(selector)?.disabled === !value,
    { selector, value });
}
async function save(page) {
  await page.locator('#save-settings').click();
  await page.waitForFunction(() => document.querySelector('#settings-form').getAttribute('aria-busy') !== 'true');
}
async function selectTheme(page, theme) {
  await page.locator('#theme-toggle').click();
  await page.locator(`[data-theme-choice="${theme}"]`).click();
  await enabled(page, '#theme-toggle', true);
}
async function main() {
  const screenshotDirectory = process.env.SCREENSHOT_DIR && path.resolve(process.env.SCREENSHOT_DIR);
  if (screenshotDirectory) fs.mkdirSync(screenshotDirectory, { recursive: true });
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
    await test('unified structure, dirty revert and cancel', {}, async ({ page, state }) => {
      assert.equal(await page.locator('#save-startup').count(), 0);
      assert.equal(await page.locator('#settings-form #startup-enabled').count(), 1);
      assert.equal((await page.locator('#save-settings').textContent()).trim(), '저장');
      assert.equal((await page.locator('#reset-settings').textContent()).trim(), '변경 취소');
      assert.equal(await page.locator('#settings details, #advanced-settings, #other-alerts, #theme-setting, [name="theme"]').count(), 0);
      assert.equal(await page.locator('#system-details').evaluate(el => el.tagName), 'SECTION');
      assert.match(await page.locator('#system-details h2#system-title').textContent(), /앱 정보/);
      assert.equal(await page.locator('#poll-help').evaluate(el => el.tagName), 'P');
      assert.equal(await page.locator('#poll-help').isVisible(), true);
      assert.equal(await page.locator('#poll-minutes').isVisible(), true);
      assert.match(await page.locator('#poll-minutes').locator('xpath=ancestor::section[1]').textContent(), /일반/);
      assert.equal(await page.locator('header #theme-toggle').count(), 1);
      assert.equal(await page.locator('#theme-menu').getAttribute('role'), 'menu');
      assert.equal(await page.locator('#theme-menu [role="menuitemradio"][data-theme-choice]').count(), 3);
      await enabled(page, '#save-settings', false);
      await page.locator('#settings-form').evaluate(form => form.requestSubmit());
      assert.deepEqual(state.mutations, [], 'unchanged form submission is a no-op');
      await page.locator('#startup-enabled').check(); await enabled(page, '#save-settings', true);
      await page.locator('#startup-enabled').uncheck(); await enabled(page, '#save-settings', false);
      assert.equal(await page.locator('input[type="radio"][name="icon_style"]').count(), 2);
      await page.locator('input[name="icon_style"][value="donut"]').focus();
      await page.keyboard.press('ArrowRight');
      assert.equal(await page.locator('input[name="icon_style"][value="number"]').isChecked(), true);
      await enabled(page, '#save-settings', true);
      await page.locator('#startup-enabled').check();
      const reads = state.reads.length;
      await page.locator('#reset-settings').click();
      await enabled(page, '#save-settings', false);
      assert.equal(await page.locator('#startup-enabled').isChecked(), false);
      assert.equal(await page.locator('input[name="icon_style"][value="donut"]').isChecked(), true);
      assert.equal(state.reads.length, reads, 'cancel restores local baselines without fetching');
      assert.deepEqual(state.mutations, []);
    });
    await test('startup-only save', {}, async ({ page, state }) => {
      await page.locator('#startup-enabled').check(); await save(page);
      await enabled(page, '#save-settings', false);
      assert.deepEqual(state.mutations, [{ path: '/api/startup', method: 'POST', body: { enabled: true } }]);
    });
    await test('settings-only save preserves all service rules and interval', {}, async ({ page, state }) => {
      const originalRules = clone(state.settings.alert_rules);
      await field(page, 'poll_minutes').fill('2.5');
      assert.equal(await field(page, 'poll_minutes').getAttribute('min'), '1');
      assert.equal(await field(page, 'poll_minutes').getAttribute('max'), '60');
      assert.equal(await field(page, 'poll_minutes').getAttribute('step'), 'any');
      await save(page); await enabled(page, '#save-settings', false);
      assert.equal(state.mutations.length, 1); assert.equal(state.mutations[0].method, 'PUT');
      assert.equal(state.settings.poll_seconds, 150);
      assert.deepEqual(state.settings.alert_rules, originalRules);
      await field(page, 'poll_minutes').fill('3'); await enabled(page, '#save-settings', true);
      await field(page, 'poll_minutes').fill('2.5'); await enabled(page, '#save-settings', false);
    });
    await test('custom integer seconds survive unrelated save', { settings: { ...baseline(), poll_seconds: 61 } }, async ({ page, state }) => {
      assert.equal(Math.round(Number(await field(page, 'poll_minutes').inputValue()) * 60), 61);
      await page.locator('input[name="icon_style"][value="number"]').check(); await save(page);
      await enabled(page, '#save-settings', false); assert.equal(state.settings.poll_seconds, 61);
    });
    await test('mixed save settings first and startup partial failure retry', { failStartup: 1 }, async ({ page, state }) => {
      await page.locator('input[name="icon_style"][value="number"]').check();
      await page.locator('#startup-enabled').check(); await save(page);
      await enabled(page, '#save-settings', true);
      assert.deepEqual(state.mutations.map(m => m.path), ['/api/settings', '/api/startup']);
      assert.equal(await page.locator('#startup-enabled').isChecked(), true);
      assert.equal(await page.locator('input[name="icon_style"][value="number"]').isChecked(), true);
      await save(page); await enabled(page, '#save-settings', false);
      assert.deepEqual(state.mutations.map(m => m.path), ['/api/settings', '/api/startup', '/api/startup']);
    });
    await test('settings failure retains edits and cancel restores saved baseline', { failSettings: 1 }, async ({ page, state }) => {
      await page.locator('input[name="icon_style"][value="number"]').check(); await save(page);
      await enabled(page, '#save-settings', true);
      assert.equal(await page.locator('input[name="icon_style"][value="number"]').isChecked(), true);
      await page.locator('#reset-settings').click(); await enabled(page, '#save-settings', false);
      assert.equal(await page.locator('input[name="icon_style"][value="donut"]').isChecked(), true);
      assert.equal(state.mutations.length, 1);
    });
    await test('mixed settings failure still saves startup and retries only settings', { failSettings: 1 }, async ({ page, state }) => {
      await page.locator('input[name="icon_style"][value="number"]').check();
      await page.locator('#startup-enabled').check(); await save(page);
      await enabled(page, '#save-settings', true);
      assert.deepEqual(state.mutations.map(m => m.path), ['/api/settings', '/api/startup']);
      assert.equal(state.startup.enabled, true);
      await save(page); await enabled(page, '#save-settings', false);
      assert.deepEqual(state.mutations.map(m => m.path), ['/api/settings', '/api/startup', '/api/settings']);
      await page.locator('#startup-enabled').uncheck();
      await page.locator('#reset-settings').click(); await enabled(page, '#save-settings', false);
      assert.equal(await page.locator('#startup-enabled').isChecked(), true, 'cancel uses latest successful startup baseline');
    });
    await test('startup success response with wrong enabled state retains desired edit for retry', { mismatchStartup: 1 }, async ({ page, state }) => {
      await page.locator('#startup-enabled').check(); await save(page);
      await enabled(page, '#save-settings', true);
      assert.equal(await page.locator('#startup-enabled').isChecked(), true, 'retain requested value');
      assert.equal(state.startup.enabled, false, 'mismatched response does not update fixture baseline');
      assert.equal(await page.locator('#settings-message').evaluate(el => el.classList.contains('error')), true,
        'mismatched response must show an error instead of success');
      assert.doesNotMatch(await page.locator('#settings-message').textContent(), /저장했습니다/);
      await save(page); await enabled(page, '#save-settings', false);
      assert.deepEqual(state.mutations, [1, 2].map(() => ({ path: '/api/startup', method: 'POST', body: { enabled: true } })));
      assert.equal(state.startup.enabled, true);
    });
    await test('invalid disconnected service rule focuses visible input without mutation', {}, async ({ page, state }) => {
      await page.waitForFunction(() => document.querySelector('#alert-rules .alert-service[data-agent="claude"] [name="claude5h_threshold"]'));
      assert.equal(await field(page, 'claude5h_threshold').isVisible(), true);
      await field(page, 'claude5h_threshold').fill('100');
      await page.locator('#settings-form').evaluate(form => form.requestSubmit());
      await page.waitForFunction(() => document.activeElement?.name === 'claude5h_threshold');
      assert.deepEqual(state.mutations, [], 'invalid input must block every write');
      await field(page, 'claude5h_threshold').fill('23');
      await save(page); await enabled(page, '#save-settings', false);
      assert.equal(state.settings.alert_rules.claude5h.threshold, 23);
    });
    await test('theme saves separately while preserving draft and cancel keeps theme', {}, async ({ page, state }) => {
      await page.locator('input[name="icon_style"][value="number"]').check();
      await page.locator('#startup-enabled').check();
      await field(page, 'poll_minutes').fill('2.5');
      await selectTheme(page, 'dark');
      await page.waitForFunction(() => document.documentElement.dataset.theme === 'dark');
      assert.deepEqual(state.mutations, [{ path: '/api/theme', method: 'PUT', body: { theme: 'dark' } }]);
      assert.equal(await page.locator('input[name="icon_style"][value="number"]').isChecked(), true);
      assert.equal(await page.locator('#startup-enabled').isChecked(), true);
      assert.equal(await field(page, 'poll_minutes').inputValue(), '2.5');
      await enabled(page, '#save-settings', true);
      await page.locator('#reset-settings').click(); await enabled(page, '#save-settings', false);
      assert.equal(await page.locator('input[name="icon_style"][value="donut"]').isChecked(), true);
      assert.equal(await page.locator('#startup-enabled').isChecked(), false);
      assert.equal(await field(page, 'poll_minutes').inputValue(), '3');
      assert.equal(state.settings.theme, 'dark');
      assert.equal(await page.locator('html').getAttribute('data-theme'), 'dark');
      assert.equal(await page.locator('[data-theme-choice="dark"]').getAttribute('aria-checked'), 'true');
      await page.locator('input[name="icon_style"][value="number"]').check(); await save(page);
      assert.equal(state.settings.theme, 'dark', 'full settings save preserves separately saved theme');
    });
    await test('theme save failure retains previous theme and unrelated draft', { settings: { ...baseline(), theme: 'light' }, failTheme: 1 }, async ({ page, state }) => {
      await page.waitForFunction(() => document.documentElement.dataset.theme === 'light');
      await field(page, 'poll_minutes').fill('2.5');
      await selectTheme(page, 'dark');
      assert.equal(state.settings.theme, 'light');
      assert.equal(await page.locator('#action-message').evaluate(el => el.classList.contains('error')), true);
      assert.match(await page.locator('#action-message').textContent(), /테마/);
      assert.equal(await page.locator('html').getAttribute('data-theme'), 'light');
      assert.equal(await page.locator('[data-theme-choice="light"]').getAttribute('aria-checked'), 'true');
      assert.equal(await field(page, 'poll_minutes').inputValue(), '2.5');
      await enabled(page, '#save-settings', true);
      assert.deepEqual(state.mutations, [{ path: '/api/theme', method: 'PUT', body: { theme: 'dark' } }]);
    });
    for (const [name, options] of [['startup read failure', { failStartupRead: true }],
      ['unsupported startup', { startup: { supported: false, enabled: false } }]]) {
      await test(`${name} leaves settings usable`, options, async ({ page, state }) => {
        await enabled(page, '#startup-enabled', false);
        await page.locator('input[name="icon_style"][value="number"]').check(); await save(page);
        await enabled(page, '#save-settings', false);
        assert.deepEqual(state.mutations.map(m => m.path), ['/api/settings']);
      });
    }
    await test('notification visibility, connected service ordering and narrow viewport overflow', {}, async ({ page }) => {
      await field(page, 'alerts_enabled').uncheck();
      assert.equal(await page.locator('#notification-options').isVisible(), false);
      await field(page, 'alerts_enabled').check();
      assert.equal(await page.locator('#notification-options').isVisible(), true);
      await page.waitForFunction(() => document.querySelector('#alert-rules .alert-service')?.dataset.agent === 'codex');
      assert.equal(await page.locator('#alert-rules .alert-service[data-agent]').count(), 3);
      assert.equal(await page.locator('#alert-rules .alert-rule[data-rule]').count(), 5);
      assert.equal(await page.locator('#alert-rules [name$="_threshold"]').count(), 5);
      for (const key of keys) assert.equal(await field(page, `${key}_threshold`).isVisible(), true);
      for (const width of [485, 320]) {
        await page.setViewportSize({ width, height: 900 });
        const dimensions = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth,
          viewport: document.documentElement.clientWidth }));
        assert.ok(dimensions.scroll <= dimensions.viewport + 1, `${width}px overflow: ${JSON.stringify(dimensions)}`);
        assert.equal(await page.locator('#save-settings').isVisible(), true);
      }
    });
    if (screenshotDirectory) {
      const { page, context } = await fixture(browser);
      try {
        await page.waitForFunction(() => document.querySelector('#alert-rules [name="codex7d_threshold"]'));
        for (const [width, filename] of [[1000, 'settings-desktop.png'], [320, 'settings-mobile320.png']]) {
          await page.setViewportSize({ width, height: 900 });
          await page.evaluate(() => {
            window.scrollTo(0, 0);
          });
          await page.screenshot({ path: path.join(screenshotDirectory, filename), fullPage: true });
        }
      } finally { await context.close(); }
    }
    console.log(`PASS: ${passed} isolated settings UI scenarios; all API mutations mocked.`);
  } finally { await browser.close(); }
}
main().catch(error => { console.error(error); process.exitCode = 1; });
