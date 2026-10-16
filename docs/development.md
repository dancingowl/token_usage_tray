# 개발 및 검증

[프로젝트 소개](../README.md) · [사용법](usage.md) · [문제 해결](troubleshooting.md)

## 환경과 브랜치

소스를 받으려면 **Git**, 빌드·Go 검증에는 **Go 1.26 이상**이 필요합니다(`go.mod`: `go 1.26.0`).
Windows는 **PowerShell 5.1 이상**을 사용합니다. 기본 빌드는 `CGO_ENABLED=0`이므로 C 컴파일러가 필요 없습니다.
Go 1.26 기준 실행 환경은 Windows 10 이상이며 이 저장소의 Windows 빌드는 x64입니다([Go 최소 요건](https://go.dev/wiki/MinimumRequirements)).
macOS는 Go 외에 **Xcode Command Line Tools**가 필요합니다(`xcode-select --install`).
메뉴바 Cocoa 연동에 cgo를 쓰므로 이 저장소의 macOS 빌드 절차는 Mac에서 실행합니다.
Node.js와 npm은 아래 JavaScript 검증·이미지 재생성 도구에만 필요하며 배포 exe 실행에는 필요 없습니다.

이 문서 작성 시 체크아웃은 `working/windows-development`, 원격 기본 브랜치 표시는 `origin/HEAD -> origin/main`입니다.
다음 명령은 Windows 개발 브랜치를 명시적으로 받습니다. 저장소 기본 브랜치를 바꾸는 명령이 아닙니다.
`main`의 코드와 개발 브랜치의 코드가 같다고 가정하지 말고 수정·검증할 브랜치를 먼저 확인합니다.

```powershell
git clone --branch working/windows-development https://github.com/dancingowl/token_usage_tray.git
cd token_usage_tray
git branch --show-current
go version
```

macOS 실행 가능 버전은 사용한 Go 툴체인의 공식 요건에 따라 달라집니다.
Go 1.26은 macOS 12 Monterey 이상, Go 1.27부터는 macOS 13 Ventura 이상이 필요합니다.
근거: [Go 최소 요건](https://go.dev/wiki/MinimumRequirements), [Go 1.26 Darwin 안내](https://go.dev/doc/go1.26#darwin).
`build.sh`의 `LSMinimumSystemVersion=11.0`은 이 요건과 일치하지 않으며 macOS 11 실행을 보장하지 않습니다.
번들 메타데이터와 툴체인 지원 범위는 후속 정합성 점검이 필요합니다.

## Windows 빌드와 기본 검증

저장소 루트의 새 PowerShell에서 실행합니다. 빌드 결과는 **Windows x64**용 `dist\usage-tray.exe`입니다.

```powershell
$env:CGO_ENABLED = '0'
go test ./...
go test fyne.io/systray
go vet ./...
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

`go test ./...`는 vendor 패키지의 테스트를 자동 포함하지 않으므로 systray를 별도로 검사합니다.
Windows 기본 경로에서는 `-race`를 사용하지 않습니다. `CGO_ENABLED=0`으로는 race detector를 실행할 수 없습니다.
빌드 후 실행까지 원하면 마지막 명령에 `-Run`을 붙입니다. 빌드만 하거나 일반 실행하는 것으로 자동 실행이 등록되지는 않습니다.
로그인 자동 실행은 [사용법](usage.md)의 설치 절차 또는 설정 화면에서 별도로 적용합니다.

## macOS 빌드와 기본 검증

Mac에서 위 clone 절차와 같은 저장소·브랜치를 받은 뒤 루트에서 실행합니다.

```bash
go version
xcode-select -p
export CGO_ENABLED=1
go test ./...
go test fyne.io/systray
go vet ./...
bash ./build.sh
open dist/UsageTray.app
```

현재 `build.sh`는 Git 파일 모드가 `100644`이므로 직접 빌드할 때 **`bash ./build.sh`**로 실행합니다.
기본값은 `lipo`가 있을 때 arm64·amd64 유니버설 빌드이며, 결과는 `dist/UsageTray.app`과 `dist/usage-tray`입니다.
`UNIVERSAL=0 bash ./build.sh`는 현재 환경의 단일 아키텍처로 빌드합니다.
번들에 `LSUIElement=true`를 넣고 `sips`·`iconutil`로 앱 아이콘을 만들며 ad-hoc 서명을 시도합니다.
실행에는 계정 연결이 필요합니다. 배포 파일의 권한·격리 문제는 [문제 해결](troubleshooting.md)을 참고합니다.

## 실제 계정으로 확인하기

먼저 기존 Usage Tray를 트레이 메뉴에서 종료합니다. 개발용 데이터 경로는 **상태·캐시·설정만 분리**합니다.
기존 로그인 자격증명과 `127.0.0.1:47113`~`47115` 포트는 공유하므로 앱 두 개를 동시에 실행하지 않습니다.
아래 수집 명령은 실제 계정의 API를 조회할 수 있으며 결과를 개발용 폴더에 저장합니다.

```powershell
$env:CGO_ENABLED = '0'
$env:USAGE_TRAY_DATA_DIR = Join-Path $env:TEMP 'token-usage-tray-dev'
go run . -once
go run . -tokens
.\dist\usage-tray.exe
```

배포 exe는 `-H windowsgui`로 콘솔 창 없이 빌드합니다. `dist\usage-tray.exe -once` 대신 **`go run . -once`**로 출력을 봅니다.
각 옵션의 동작은 아래 표를 참고합니다. 자격증명 없이 이미지 생성만 확인하려면 `.\dist\usage-tray.exe -icons dist\icons`를 사용합니다.
앱을 트레이 메뉴에서 종료한 뒤 `Remove-Item Env:USAGE_TRAY_DATA_DIR`로 경로 설정을 해제합니다.
개발용 파일은 위에서 지정한 임시 폴더에 남습니다. 필요 없을 때 해당 폴더만 확인하고 삭제합니다.
macOS에서는 `export USAGE_TRAY_DATA_DIR="${TMPDIR}token-usage-tray-dev"`로 지정하고 `go run . -once`로 확인합니다.
종료 후 `unset USAGE_TRAY_DATA_DIR`로 해제합니다. 새 PowerShell 창을 쓰면 검증용 `CGO_ENABLED` 등도 분리할 수 있습니다.

## CLI 플래그

| 플래그 | 동작과 영향 |
|---|---|
| `-once` | 실제 수집·상태 저장 후 요약·툴팁 내용을 출력하고 종료합니다. 트레이·HTTP 서버는 띄우지 않습니다. 연결 가능한 에이전트가 없으면 안내 후 종료합니다. |
| `-tokens` | 감지된 Claude·Codex의 로컬 토큰 집계를 JSON으로 출력하고 종료합니다. 트레이·HTTP 서버는 띄우지 않습니다. |
| `-only claude` / `-only codex` / `-only antigravity` | 해당 에이전트만 처리합니다. 일반 실행 또는 `-once`·`-tokens`와 함께 쓸 수 있습니다. Antigravity 연결은 macOS만 지원합니다. |
| `-icons <폴더>` | 자격증명 없이 아이콘 견본 PNG·ICO를 저장하고 종료합니다. 트레이·HTTP 서버는 띄우지 않습니다. |
| `-notify` | 자격증명 없이 OS 알림을 한 번 요청하고 약 3초 후 종료합니다. 트레이·HTTP 서버는 띄우지 않으며 OS 알림 설정에 따라 표시 여부가 달라집니다. |
| `-install` | Windows 시작프로그램 또는 macOS 로그인 항목을 등록하고 앱을 실행합니다. Windows는 트레이 표시 고정도 적용합니다. macOS에서는 `UsageTray.app` 내부 바이너리로 실행합니다. |
| `-uninstall` | 로그인 자동 실행 등록을 해제하고 종료합니다. 실행 중인 앱은 종료하지 않으며 실행 파일·데이터·Windows 아이콘 표시 설정은 삭제하지 않습니다. |
| `-promote` | Windows 전용입니다. 기존 트레이 항목에 `IsPromoted=1`을 기록하고 종료합니다. 자동 실행을 등록하지 않으며 표시 적용에는 트레이 재실행이 필요할 수 있습니다. |
| `-restart-explorer` | Windows에서 `-promote`와 함께 사용할 때만 적용됩니다. 앱 트레이와 Explorer를 종료·재실행하므로 작업표시줄이 잠시 사라집니다. 단독 사용은 효과가 없습니다. |

`-install`·`-uninstall`·`-promote`·`-icons`·`-notify`는 자격증명 감지 전에 처리됩니다.
서로 다른 종료용 옵션을 겹쳐 쓰면 앞선 처리만 실행됩니다. 일반 실행과 `-Run`은 자동 실행을 등록하지 않습니다.
설치·자동 실행 해제는 [사용법](usage.md), 파일·데이터 삭제와 실행 문제는 [문제 해결](troubleshooting.md)을 참고합니다.

## JavaScript 검증 도구

저장소에는 npm 패키지 설치를 강제하는 앱 런타임이 없습니다. Node.js는 이 절의 개발 도구에만 사용합니다.
`scripts/test-history-bins.cjs`는 Node 내장 모듈로 `dashboard.go`의 순수 표시 함수를 추출해 검증합니다.
서버·브라우저·Playwright가 필요 없으며 저장소 루트에서 실행합니다.

```powershell
node scripts/test-history-bins.cjs
```

설정 UI는 `scripts/test-settings-ui.cjs`로 검증합니다. Node.js와 `playwright` 또는 `playwright-core`, Chromium 계열 브라우저가 필요합니다.
앱의 `/settings` HTML과 읽기 전용 정적 자산을 받아야 하므로 **별도 터미널에서 앱 서버를 먼저 실행**합니다.
기존 앱을 종료한 후 앞 절의 데이터 경로를 지정하고 빌드한 exe를 일반 실행합니다. 계정이 없으면 서버도 뜨지 않습니다.
기본 주소는 `http://127.0.0.1:47113`이며 실제 실행 중인 에이전트 포트가 다르면 `BASE_URL`을 맞춥니다.

의존성을 저장소 밖에 설치하는 Windows 예시입니다. npm 사용이 가능한 개발용 터미널에서 실행합니다.

```powershell
$uiTools = Join-Path $env:TEMP 'usage-tray-ui-tools'
npm install --prefix $uiTools playwright
& (Join-Path $uiTools 'node_modules\.bin\playwright.cmd') install chromium
$env:PLAYWRIGHT_MODULE_PATH = Join-Path $uiTools 'node_modules\playwright'
$env:BASE_URL = 'http://127.0.0.1:47113'
node scripts/test-settings-ui.cjs
```

브라우저를 따로 내려받지 않고 설치된 Chrome을 쓸 경우 `install chromium` 대신 실제 실행 파일 경로를 지정합니다.
예: `$env:CHROME_PATH = 'C:\Program Files\Google\Chrome\Application\chrome.exe'`.
`PLAYWRIGHT_MODULE_PATH`는 모듈 경로나 `node_modules/playwright`를 찾을 수 있는 설치 루트 경로를 받습니다.
생략하면 스크립트가 `playwright`, 다음으로 `playwright-core`를 일반 Node 모듈 탐색으로 찾습니다.
macOS도 같은 변수 이름을 `export`로 지정하고 설치한 Playwright CLI로 `install chromium` 후 Node 스크립트를 실행합니다.
선택 변수 `SCREENSHOT_DIR`은 데스크톱·320px 화면 스크린샷을 저장할 폴더입니다.

UI 테스트는 `/api/*`와 `/state.json` 요청을 모두 mock합니다. 설정·자동 실행 저장도 브라우저 안의 fixture만 변경합니다.
실제 앱으로 전달하는 요청은 같은 origin의 페이지·정적 자산 GET뿐이며 다른 요청은 차단합니다.
따라서 저장 순서·부분 실패 재시도·변경 취소·좁은 화면 등을 확인하지만 실제 레지스트리 등록이나 API 수집 성공을 검증하지 않습니다.
서버로 실행한 앱 자체의 수집은 계속 동작합니다. mock 범위와 실제 앱 동작을 혼동하지 않습니다.
검증 후 앱을 종료하고 테스트 터미널을 닫거나 `BASE_URL`, `PLAYWRIGHT_MODULE_PATH`, `CHROME_PATH`, `SCREENSHOT_DIR`를 해제합니다.

## 구현 위치와 OS 차이

| 위치 | 역할 |
|---|---|
| `main.go`, `tray_preferences.go`, `click_summary.go` | 수집 주기·트레이·표시 선택·클릭 요약 |
| `usage.go`, `paths.go`, `claude_desktop.go`, `antigravity.go` | 서비스 조회·자격증명 탐색·로컬 폴백 |
| `settings.go`, `alerts.go`, `history.go`, `tokens.go` | 설정·알림·30일 관측 기록·토큰 집계 |
| `service.go`, `dashboard.go`, `page.go` | 루프백 API·상세 화면·설정 화면 |
| `platform_windows.go`, `platform_unix.go`, `startup_settings_*.go` | OS 기능·자동 실행 등록 |
| `icon.go`, `icon_text_*.go`, `assets.go` | 잔량 아이콘·앱 아이콘 자산 |
| `update.go` | 버전 비교·사용자 요청 시 릴리스 조회 |

systray는 프로세스당 아이콘 하나를 제공합니다. Windows는 여러 에이전트가 있으면 자신을 `-only <agent>`로 다시 실행합니다.
상태 파일도 에이전트별로 나누고 상세 화면은 여러 프로세스의 정보를 합칩니다.
`-only claude`는 47113, `-only codex`는 47114, `-only antigravity`는 47115를 사용합니다.
Antigravity 연결은 macOS만 지원하며 기본 단일 에이전트 실행과 macOS 통합 실행은 47113을 사용합니다.
macOS는 번들 본체가 종료되는 프로세스 분리 방식에서 메뉴바를 유지할 수 없어 한 프로세스가 가장 부족한 에이전트를 표시합니다.

Windows `-install`은 실행 파일 경로로 `HKCU\Control Panel\NotifyIconSettings` 항목을 찾아 `IsPromoted=1` 기록 후 트레이를 다시 띄웁니다.
셸의 아이콘 등록 시점에 값을 읽게 하는 순서이며 항목이 없으면 먼저 앱을 실행해 생성을 기다립니다.

Claude는 statusline의 2분 이내 캐시를 우선 사용하고 직접 조회는 Claude 3분·Codex 2분의 최소 간격을 둡니다.
Claude의 429·503 응답은 5분부터 최대 30분까지 지수 백오프합니다. 실패하면 직전 정상값과 경과 시간을 유지합니다.
Claude·Codex 조회 경로는 비공개 API이며 로컬 기록으로 폴백할 수 있습니다. 화면의 출처·수집 상태를 함께 확인합니다.

## vendor 패치 유지

현재 모듈은 `fyne.io/systray v1.11.0`, `golang.org/x/sys v0.48.0`, 간접 의존성 `github.com/godbus/dbus/v5 v5.1.0`입니다.
`vendor/fyne.io/systray`에는 upstream 외에 다음 Windows 커스텀 기능과 테스트가 들어 있습니다.

- `systray_windows.go`, `tooltip_windows.go`: 클릭·더블클릭 구분, 네이티브 요약 카드·수명·갱신 처리.
- `tooltip_rows*.go`, `tooltip_style_windows.go`: 한도별 행·초기화 정보·색상·DPI별 배치.
- `popup_actions.go`, `tooltip_actions_windows.go`, `tooltip_button_windows.go`: 새로고침·설정 버튼과 액션 채널.
- `tooltip_icon_windows.go`, `icons/`: Lucide PNG 마스크 임베딩·DPI 축소·버튼 상태별 색상.
- `visibility_windows.go`, `tooltip_other.go`: 아이콘 숨김·복원, 다른 OS의 확장 API 호환 처리.

Go의 기본 vendor 모드를 유지합니다. **`go mod vendor`는 로컬 패치를 덮어쓸 수 있고 `-mod=mod`는 패치를 우회합니다.**
의존성 갱신 전 패치·테스트·아이콘·라이선스 전체를 Git diff/별도 패치로 보존하고 새 upstream에 다시 적용해 검토합니다.
업데이트 후 `go test ./...`, `go test fyne.io/systray`, `go vet ./...`와 양 OS 빌드·수동 동작을 확인합니다.
외부 라이브러리의 README·라이선스도 그대로 보존합니다.

## 아이콘 재생성

앱 아이콘과 잔량 트레이 아이콘은 별개입니다. 앱 아이콘 원본은 `assets/icon.svg`이고 [Lucide Gauge](https://lucide.dev/icons/gauge)를 바탕으로 합니다.
[출처·ISC 라이선스](../assets/LICENSE.lucide.txt)를 보존합니다. SVG 변경 후 `icon.png`(1024px), `icon.ico`(16·20·24·32·48·64·128·256px),
`favicon32.png`(32px)를 다시 만들고 파생 PNG의 `License` 메타데이터에도 라이선스를 넣습니다.
Windows 리소스는 기존 절차대로 한 번 재생성합니다. 일반 빌드에는 커밋된 `rsrc_windows.syso`를 사용합니다.

```powershell
go run github.com/akavel/rsrc@latest -ico assets\icon.ico -arch amd64 -o rsrc_windows.syso
```

클릭 카드 버튼의 SVG·PNG는 `vendor/fyne.io/systray/icons/`에 있습니다.
[아이콘 원본·리비전](../vendor/fyne.io/systray/icons/README.md)과 [ISC·제삼자 고지](../vendor/fyne.io/systray/icons/LICENSE)를 함께 보존합니다.
저장소 밖 개발 폴더에 `sharp`를 설치하고 `SHARP_MODULE_PATH`를 모듈 경로로 지정합니다.
생성 결과는 투명 128×128 PNG이며 빌드·실행 시에는 Node.js·sharp·SVG 렌더러가 필요 없습니다.

```powershell
$iconTools = Join-Path $env:TEMP 'usage-tray-icon-tools'
npm install --prefix $iconTools sharp
$env:SHARP_MODULE_PATH = Join-Path $iconTools 'node_modules\sharp'
node scripts/render-tray-action-icons.cjs
```

재생성 후 터미널을 닫거나 `Remove-Item Env:SHARP_MODULE_PATH`로 도구 경로를 해제합니다.

## 버전과 CI

로컬 빌드의 기본 버전은 `dev`이며 업데이트 버전 비교를 하지 않습니다. 버전은 `main.appVersion`에 링크합니다.
Windows는 `powershell -ExecutionPolicy Bypass -File .\build.ps1 -Version v1.2.3`, macOS는 `VERSION=v1.2.3 bash ./build.sh`로 지정합니다.
[build workflow](../.github/workflows/build.yml)는 `main` push, `v*` 태그 push, PR, 수동 실행을 받습니다.
현재 개발 브랜치의 일반 push만으로 실행되는 설정은 아닙니다. CI는 Go `stable`을 사용하므로 툴체인·OS 요건을 함께 확인합니다.
Windows는 앱·vendor 테스트와 아이콘 생성 스모크, macOS는 앱 테스트·유니버설 번들·아이콘 스모크를 수행합니다.
별도 Linux job은 vendor를 제외한 Go 파일 서식과 `go vet ./...`를 검사합니다.
Windows exe·macOS zip artifact는 7일 보관하고 `v*` 태그에서는 릴리스에 첨부합니다. 태그 빌드는 태그명을 버전으로 사용합니다.
