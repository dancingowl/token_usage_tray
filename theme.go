package main

import (
	"image/color"

	"fyne.io/systray"
)

// App surfaces follow the application preference. The taskbar is owned by the
// OS and uses its own preference even when the app is forced light or dark.
func resolvedTheme(theme string, systemDark bool) string {
	if theme == "dark" || (theme != "light" && systemDark) {
		return "dark"
	}
	return "light"
}

func applyTrayTheme(theme string) {
	systray.SetClickTooltipTheme(resolvedTheme(theme, systemAppDark()))
}

// Polling this inexpensive stamp lets the UI react to OS theme changes even
// when collection is paused or its network interval is long.
func trayThemeStamp(theme string) uint8 {
	var stamp uint8
	if resolvedTheme(theme, systemAppDark()) == "dark" {
		stamp |= 1
	}
	if systemTaskbarDark() {
		stamp |= 2
	}
	return stamp
}

func trayNumberColor(brand color.NRGBA, dark bool) color.NRGBA {
	if dark {
		switch brand {
		case colCodex:
			return color.NRGBA{107, 232, 196, 255} // Mint, against the dark taskbar.
		case colClaude:
			return color.NRGBA{239, 167, 139, 255}
		case colAntigravity:
			return color.NRGBA{166, 187, 255, 255}
		}
	}
	if brand == colCodex {
		return colCodexNumber
	}
	return brand
}

func themeScriptTag() string { return "<script id=\"theme-init\">" + themeJS + "</script>" }

// Run in the head before styles/body to avoid a light flash on dark pages.
// API refreshes call applyUsageTrayTheme with the saved preference. Local
// storage gives subsequent navigation and other open tabs the same first paint.
const themeJS = `(function(){
'use strict';
const key='usage-tray-theme', media=window.matchMedia('(prefers-color-scheme: dark)');
let preference='system';
function normalize(value){return value==='light'||value==='dark'?value:'system';}
function paint(){document.documentElement.dataset.theme=preference==='system'?(media.matches?'dark':'light'):preference;document.documentElement.style.colorScheme=document.documentElement.dataset.theme;}
function announce(){window.dispatchEvent(new CustomEvent('usage-tray-theme-change',{detail:preference}));}
window.getUsageTrayTheme=function(){return preference;};
window.applyUsageTrayTheme=function(value){const next=normalize(value),changed=next!==preference;preference=next;paint();try{localStorage.setItem(key,preference);}catch(_){}if(changed)announce();};
try{preference=normalize(localStorage.getItem(key));}catch(_){}paint();
window.addEventListener('storage',function(event){if(event.key===key){preference=normalize(event.newValue);paint();announce();}});
if(media.addEventListener)media.addEventListener('change',paint);else if(media.addListener)media.addListener(paint);
})();`

// Lucide sun/moon/monitor geometry, unchanged from revision
// 7b0196e249ca09ca27c584c497aca9a8cab8a7f5 (ISC; THIRD_PARTY_NOTICES.txt).
const themeSun = `<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/>`
const themeMoon = `<path d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401"/>`
const themeMonitor = `<rect width="20" height="14" x="2" y="3" rx="2"/><path d="M8 21h8M12 17v4"/>`
const themeSVGStart = `<svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">`
const themeControlHTML = `<div class="theme-control" id="theme-control"><button type="button" id="theme-toggle" class="theme-toggle" aria-haspopup="menu" aria-controls="theme-menu" aria-expanded="false" aria-label="테마: 시스템 설정" title="테마: 시스템 설정">` + themeSVGStart + themeMonitor + `</svg></button><div id="theme-menu" class="theme-menu" role="menu" aria-label="테마" hidden>` +
	`<button type="button" role="menuitemradio" aria-checked="true" tabindex="-1" data-theme-choice="system">` + themeSVGStart + themeMonitor + `</svg>시스템 설정<span class="theme-check" aria-hidden="true">✓</span></button>` +
	`<button type="button" role="menuitemradio" aria-checked="false" tabindex="-1" data-theme-choice="light">` + themeSVGStart + themeSun + `</svg>밝게<span class="theme-check" aria-hidden="true">✓</span></button>` +
	`<button type="button" role="menuitemradio" aria-checked="false" tabindex="-1" data-theme-choice="dark">` + themeSVGStart + themeMoon + `</svg>어둡게<span class="theme-check" aria-hidden="true">✓</span></button></div></div>`

const themeControlCSS = `
.theme-control{position:relative}.theme-toggle{width:36px;padding:8px}.theme-menu{position:absolute;right:0;top:calc(100% + 8px);z-index:30;min-width:164px;padding:5px;border:1px solid var(--line);border-radius:9px;background:var(--panel);box-shadow:0 5px 18px #0002}
.theme-menu button{width:100%;justify-content:flex-start;gap:10px;min-height:36px;padding:8px 10px;border:0;background:transparent;color:var(--tx);font-size:13px;font-weight:400;white-space:nowrap}.theme-check{margin-left:auto;visibility:hidden}.theme-menu [aria-checked="true"] .theme-check{visibility:visible}
`

// Keep theme overrides after pageCSS and dashboardCSS. These rules only alter
// colors: existing typography, icon masks, graph geometry and spacing remain.
const themeCSS = `
html{background:var(--bg)}
:root[data-theme="dark"]{color-scheme:dark;--bg:#14191f;--panel:#1d252e;--panel2:#242e39;--line:#3a4757;--tx:#edf2f8;--sub:#b2bfce;--dim:#b2bfce;--accent:#6be8c4;--warn:#f0c366;--bad:#ff9292}
:root[data-theme="dark"] .page-nav a[aria-current="page"]{color:var(--accent)}
:root[data-theme="dark"] select,:root[data-theme="dark"] input:is([type=number],[type=time],[type=date],[type=text]){color:var(--tx);background:var(--panel2);border-color:var(--line)}
:root[data-theme="dark"] button,:root[data-theme="dark"] .button{color:var(--tx);background:#2b3745;border-color:#47576a}
:root[data-theme="dark"] button:hover:not(:disabled),:root[data-theme="dark"] .button:hover{background:#37485b;border-color:#677c92}
:root[data-theme="dark"] button:active:not(:disabled),:root[data-theme="dark"] .button:active{background:#40546b;border-color:#8194a9}
:root[data-theme="dark"] button:disabled{color:#a0aebe;background:#242e39;border-color:var(--line)}
:root[data-theme="dark"] button.primary{background:#1d7266;border-color:#47a896;color:#fff}
:root[data-theme="dark"] button.primary:hover:not(:disabled){background:#258575;border-color:#6be8c4}
:root[data-theme="dark"] button.primary:active:not(:disabled){background:#195d54}
:root[data-theme="dark"] button.primary:disabled{background:#29433f;border-color:#3c5752;color:#b1bfba}
:root[data-theme="dark"] .button.subtle{color:var(--sub);background:transparent;border-color:transparent}
:root[data-theme="dark"] .button.subtle:hover{background:#2b3745;color:var(--tx);border-color:var(--line)}
:root[data-theme="dark"] :is(button,input,select,summary,a):focus-visible,:root[data-theme="dark"] .icon-option input:focus-visible+.icon-option-face{outline-color:var(--accent)}
:root[data-theme="dark"] .dash-message,:root[data-theme="dark"] .history-tooltip strong,:root[data-theme="dark"] .preview-number{color:var(--accent)}
:root[data-theme="dark"] .dash-message.error,:root[data-theme="dark"] .risk .pace-status{color:var(--bad)}
:root[data-theme="dark"] .history-help>summary{border-color:var(--line)}
:root[data-theme="dark"] .history-help[open]>summary,:root[data-theme="dark"] #history-zoom[aria-pressed="true"],:root[data-theme="dark"] .icon-option input:checked+.icon-option-face{color:var(--accent);background:#223f3a;border-color:#4a8779}
:root[data-theme="dark"] .history-help-content,:root[data-theme="dark"] .history-tooltip,:root[data-theme="dark"] .token-tooltip{background:var(--panel2);border-color:var(--line);box-shadow:0 4px 18px #0006}
:root[data-theme="dark"] .settings-footer,:root[data-theme="dark"] .icon-option-face{background:var(--panel)}
:root[data-theme="dark"] .settings-switch{background:#495869;border-color:#687a8f}
:root[data-theme="dark"] .settings-switch:checked{background:#287d6e;border-color:#62b5a2}
:root[data-theme="dark"] .service-option input,:root[data-theme="dark"] .alert-recovery input{accent-color:#6be8c4}
:root[data-theme="dark"] .ch-col[data-agent="Codex"] .ch-bar{background:#6be8c4}
:root[data-theme="dark"] .ch-col[data-agent="Claude"] .ch-bar{background:#efa78b}
:root[data-theme="dark"] .preview-donut{border-color:var(--accent);border-right-color:var(--line)}
:root[data-theme="dark"] .dash-history-chart text{fill:var(--sub)}
:root[data-theme="dark"] .dash-history-chart line[stroke="#dce3eb"]{stroke:var(--line)}
:root[data-theme="dark"] .dash-history-chart polyline,:root[data-theme="dark"] .history-overview polyline{stroke:var(--accent)}
:root[data-theme="dark"] .history-selected-point{fill:var(--panel2);stroke:var(--accent)}
:root[data-theme="dark"] .history-overview rect[fill="#edf1f4"]{fill:#18212b}
:root[data-theme="dark"] .history-zoom-window{fill:#2b5347;stroke:#78b9a4}
`
