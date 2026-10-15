# token_usage_tray

[inging88/usage_tray](https://github.com/inging88/usage_tray)를 원작자의 허락을 받아
[dancingowl/token_usage_tray](https://github.com/dancingowl/token_usage_tray)라는 독립 저장소로 옮긴 프로젝트입니다.
기존 코드와 커밋 이력을 보존했으며, 원본 프로젝트에 대한 출처를 유지합니다.

Claude Code · Codex · Antigravity 의 남은 사용량을 **작업표시줄(Windows) · 메뉴바(macOS)** 에 띄운다.

Windows는 Claude Code · Codex를 지원하며, Antigravity 연결은 macOS에서만 지원한다.

- 기본 도넛은 링 길이가 남은 %, 20% 밑이면 빨강. 색이 어느 에이전트인지 알려 준다
- 우클릭 메뉴 → **아이콘 모양**에서 **도넛 / 숫자만**을 선택한다
- **윈도우는 에이전트마다 아이콘 하나**, **맥은 아이콘 하나에 가장 급한 에이전트** ([왜 다른가](#아이콘-개수가-os-마다-다른-이유))
- 마우스를 올리면 남은량과 리셋 시각을 보여 준다. Windows는 **한 번 클릭하면 요약 툴팁**, **더블클릭하면 상세 화면(브라우저)**을 연다
- 남은량이 임계값을 넘으면 알림 한 번, 창이 초기화돼 다시 쓸 수 있게 되면 또 한 번
- **쓰는 에이전트만 나온다.** 하나도 없으면 실행되지 않는다

바뀐 점과 연결 조건은 [UPGRADE.md](UPGRADE.md) 를 먼저 읽는다.

별도의 로그인 설정 없이 이미 로그인해 둔 자격증명을 읽고 자기 계정의 자기 사용량을 보여 준다.
아이콘 모양은 우클릭 메뉴에서 선택하면 데이터 폴더의 `settings.json`에 자동으로 저장한다.

## 다운로드

[Releases](https://github.com/dancingowl/token_usage_tray/releases) 에서 자기 OS 파일을 받는다.
릴리스가 아직 없다면 아래 직접 빌드 절차를 사용한다.

| OS | 파일 |
|---|---|
| Windows | `usage-tray.exe` |
| macOS | `UsageTray-macos-universal.zip` |

---

# Windows

## 트레이 조작과 아이콘 모양

- 한 번 클릭: 한도마다 2줄로 정리한 요약 카드를 7초간 띄운다. 첫 줄은 서비스·한도명과 굵은 잔량, 둘째 줄은 초기화 날짜·시각과 남은 시간이다. 잔량은 20% 이하에서 주황색, 10% 이하에서 빨간색으로 강조한다. 더블클릭과 구분하기 위해 Windows의 더블클릭 대기 시간 뒤에 표시하며, 다시 한 번 클릭하면 닫힌다.
- 더블클릭: 모든 에이전트를 합친 상세 화면을 브라우저에서 연다.
- 오른쪽 클릭: 갱신·설정·아이콘 모양·종료 메뉴를 연다.

5시간 한도의 사용량과 초기화 정보가 모두 없으면 해당 항목은 표시하지 않는다. 남은량이 0%인 한도는 표시한다.
요약 아래의 **갱신 / 상세 / 설정** 버튼으로 바로 조작한다. 카드 위에 마우스를 두면 유지되고, 벗어난 뒤 약 7초 후 닫힌다.

**아이콘 모양**에서 아래 중 하나를 선택하면 바로 적용한다.

| 모양 | 표시 |
|---|---|
| 도넛 | 기존 링 게이지 (기본값) |
| 숫자만 | 남은 퍼센트 숫자. 예: `64`는 64% 남음 |

숫자도 남은량이 20% 미만이면 빨강이며, 조회된 값이 없으면 `-`를 표시한다.
Windows 숫자는 Segoe UI 글꼴을 사용한다.
선택은 앱 재시작 후에도 유지하며, Windows에서 에이전트별 프로세스가 실행 중이면 모든 아이콘에 같은 모양을 반영한다.

## 설정·수집 상태·사용 계획

트레이 **설정** 또는 상세 화면의 **설정**에서 변경한다. 저장하면 재시작 없이 각 에이전트에 반영된다.

상세 화면은 **한도 현황(`/`) · 토큰 통계(`/tokens`) · 설정(`/settings`)**으로 나뉜다.
한도 현황은 잔량과 초기화까지 남은 시간을 강조하고 잔여량 추이를 바로 보여준다.
추이 그래프는 실제 기록 범위에 맞춰 시간 간격을 자동 조절하고 구간별 마지막 잔량을 표시한다.
초기화 전후와 기록이 없는 구간은 선을 잇지 않는다. 원본 기록과 소진 예상 계산은 유지하며, 자세한 안내는 제목 옆 `?`에서 확인한다.
토큰 통계에서는 일별·모델별·프로젝트별 집계를, 설정에서는 표시·알림·자동 실행과 앱 상태·업데이트를 확인한다.
화면 자동 갱신 중에도 선택한 이력 조건과 저장하지 않은 설정은 유지된다. 저장 전 다른 화면으로 이동하면 브라우저가 확인한다.

| 기능 | 동작 |
|---|---|
| 표시할 한도 | 주간 / 단기 / 가장 부족한 한도. 클릭 요약에 아이콘 기준도 표시한다. 선택한 한도 정보가 없으면 값 없음으로 표시한다 |
| 에이전트 표시 | 트레이에서 에이전트를 숨기거나 다시 표시한다. 수집과 상세 화면은 유지한다 |
| 갱신 주기 | 60~3600초. API 조회는 Claude 최소 3분, Codex 최소 2분의 안전 간격을 유지한다 |
| 알림 | 전체 켜기/끄기, 한도별 1~99% 기준과 회복 알림, Codex 한도 도달 알림을 설정한다 |
| 자동 실행 | 설정 화면에서 로그인 자동 실행을 등록·해제한다. 앱이나 Explorer를 재시작하지 않는다 |
| 수집 상태 | 정상 수집 / 기록·캐시 / 로그인 문제 / 수집 실패와 마지막 성공 시각을 구분한다 |
| 사용량 기록 | 실제 관측을 최대 30일 저장하고 1일·7일·30일 잔여 한도 그래프로 확인한다 |
| 소진 예상 | 현재 초기화 주기에서 3개 이상·30분 이상·1% 이상 감소한 관측을 바탕으로 계산한다 |
| 권장 사용 속도 | 초기화까지 남은 한도를 균등하게 사용하는 속도를 한도 퍼센트포인트/일로 표시한다 |
| 업데이트 | 버튼을 눌렀을 때 GitHub 최신 릴리스와 변경 내용을 확인하고 다운로드 페이지로 연결한다 |

에이전트를 모두 숨겨도 설정 진입용 아이콘 하나는 남긴다. 데이터가 오래됐거나 캐시·직전값이면 예측과 권장 속도는 숨긴다. 초기화·한도 회복·주기 변경 전후 기록은 예측에서 섞지 않는다. 예측은 최근 감소 속도가 계속된다는 가정이며 토큰 수를 추정하지 않는다.

업데이트 파일을 자동으로 설치하지는 않는다. 로컬 빌드는 `dev`로 표시하며 버전 비교를 하지 않는다. 릴리스 버전 빌드는 `./build.ps1 -Version v1.2.3` 또는 macOS에서 `VERSION=v1.2.3 ./build.sh`처럼 버전을 전달한다. 태그 CI 빌드는 태그 버전을 사용한다.

## 설치 — 한 줄

`usage-tray.exe` 를 **계속 둘 폴더**에 놓고(경로가 바뀌면 다시 걸어야 한다) 한 줄만 친다.

```powershell
# PowerShell 은 현재 폴더를 PATH 로 찾지 않는다 — `.\` 를 붙이거나 전체 경로를 쓴다
.\usage-tray.exe -install
```

이 한 줄이 셋을 한다.

1. 시작프로그램에 바로가기를 만든다 → 다음 로그인부터 저절로 뜬다
2. 아이콘을 오버플로(∧)에서 작업표시줄로 꺼낸다(`IsPromoted=1`)
3. 트레이를 지금 띄운다

떼려면 `.\usage-tray.exe -uninstall`. 자동 실행만 떼고 아이콘 설정은 건드리지 않는다.

"Windows 의 PC 보호" 가 뜨면 `추가 정보` → `실행` — 서명이 없어서 나는 경고다.

## 어떻게 동작하나

새 트레이 아이콘은 기본으로 오버플로 안에 숨는다. Windows 11 은 아이콘별 표시 여부를
`HKCU\Control Panel\NotifyIconSettings\<id>\IsPromoted` 로 기억한다(`<id>` 는 실행 파일 경로
해시라 각 항목의 `ExecutablePath` 로 찾는다). 그 항목은 아이콘이 한 번 뜬 뒤에야 생기므로,
`-install` 은 항목이 없으면 앱을 먼저 띄워 만들고 기다린다.

**순서가 핵심이다.** 값을 먼저 박고 트레이를 새로 띄우면 셸이 아이콘 등록 시점에 그 값을 읽어
바로 작업표시줄에 올린다 — explorer 를 재시작할 일이 없다. 값만 따로 넣으려면 `-promote`,
셸이 값을 안 읽는 드문 경우에만 `-promote -restart-explorer`(작업표시줄이 몇 초 사라진다).

손으로 하려면 설정 → 개인 설정 → 작업 표시줄 → 시스템 트레이 아이콘에서 켜거나,
오버플로를 열어 아이콘을 작업표시줄로 끌어다 놓는다.

## 직접 빌드

Go 1.26+ 만 있으면 된다. 컴파일러는 필요 없다(`CGO_ENABLED=0`).

```powershell
git clone https://github.com/dancingowl/token_usage_tray.git
cd token_usage_tray
powershell -ExecutionPolicy Bypass -File build.ps1 -Run
```

`-Run`을 빼면 빌드만 한다. 결과는 `dist\usage-tray.exe`이며, 일반 실행은 자동 시작을 등록하지 않는다.

### 수정 전 검증

Go 설치 후 새 PowerShell에서 아래를 실행한다. 기존 UsageTray는 종료한 상태에서 확인한다.
개발용 데이터 폴더는 저장 데이터만 분리하며, 로그인 정보와 로컬 포트는 그대로 사용한다.

```powershell
$env:CGO_ENABLED = '0'
$env:USAGE_TRAY_DATA_DIR = Join-Path $env:TEMP 'token-usage-tray-dev'
go test ./...
go vet ./...
powershell -ExecutionPolicy Bypass -File .\build.ps1
go run . -once
.\dist\usage-tray.exe
```

배포 exe는 콘솔 창을 만들지 않도록 빌드하므로, 출력 확인에는 `go run . -once`를 사용한다.
`-once`는 사용량을 조회하고 상태 파일을 저장한 뒤 종료하며, 트레이와 HTTP 서버는 띄우지 않는다.
검증 후에는 트레이 메뉴에서 종료하고 `Remove-Item Env:USAGE_TRAY_DATA_DIR`로 데이터 경로 설정을 해제한다.

앱 아이콘은 어두운 둥근 사각형 안의 민트색 계기판이다. 실행 파일·바로가기·상세 화면에 사용하며,
트레이의 도넛·숫자 잔여량 표시는 별개다. 계기판은 [Lucide Gauge](https://lucide.dev/icons/gauge)를
바탕으로 만들었으며 출처와 ISC 라이선스는 `assets/LICENSE.lucide.txt`에 보존한다.

앱 아이콘은 `rsrc_windows.syso`로 exe에 포함되어 있어 바로 빌드하면 된다.
그림을 바꾸려면 `assets/icon.svg`를 수정하고 `assets/icon.png`(1024px),
`assets/icon.ico`(16·20·24·32·48·64·128·256px), `assets/favicon32.png`(32px)를
다시 만든 뒤 아래를 한 번 돌린다. 파생 PNG의 `License` 메타데이터에도 라이선스를 포함한다.

```powershell
go run github.com/akavel/rsrc@latest -ico assets\icon.ico -arch amd64 -o rsrc_windows.syso
```

## 파일이 놓이는 곳

`%LOCALAPPDATA%\usage-tray\`에 상태·토큰 집계·캐시와 `usage-tray.log`, 아이콘 모양 설정 `settings.json`을 저장한다.
에이전트가 둘 이상이면 프로세스를 나누어 `state-<agent>.json` · `status-<agent>.txt` ·
`tokens-<agent>.json` · `tokens-cache-<agent>.json`을 사용한다. 에이전트가 하나인 일반 실행은
접미사 없이 `state.json` · `status.txt` · `tokens.json` · `tokens-cache.json`을 사용한다.
`history.json` 또는 `history-<agent>.json`에는 사용량 관측만 저장한다. `heartbeat-<agent>`는 에이전트별 프로세스 생존 표시이다. 설정·사용량 기록에는 로그인 토큰을 저장하지 않는다.
`-only`를 지정하면 하나만 실행해도 에이전트 접미사가 붙는다. 지울 때는 실행 파일과
이 폴더를 삭제하면 된다. `USAGE_TRAY_DATA_DIR`을 주면 그 폴더를 대신 쓴다(검증용).

---

# macOS

## 실행

```bash
unzip UsageTray-macos-universal.zip
xattr -dr com.apple.quarantine UsageTray.app   # 서명이 ad-hoc 이라 한 번 필요하다
mv UsageTray.app /Applications/               # 계속 둘 자리로 옮긴다
/Applications/UsageTray.app/Contents/MacOS/usage-tray -install
```

마지막 줄이 로그인 항목에 등록하고 앱을 띄운다. 메뉴바에 링이 생기고, Dock 아이콘은 없다
(`LSUIElement`). 처음 한 번 맥이 **'시스템 이벤트' 제어 권한**을 물으면 허용한다 — 로그인
항목을 넣는 데 필요하다. 거절했다면 시스템 설정 → 개인정보 보호 및 보안 → 자동화 에서 켠다.

떼려면 `-uninstall`. 앱을 옮겼다면 새 자리에서 `-install` 을 다시 한 번 돌린다(옛 항목은
자동으로 지운다).

손으로 하려면 시스템 설정 → 일반 → 로그인 항목 → `+` → `UsageTray.app`

## 직접 빌드

**맥에서 빌드해야 한다.** 메뉴바(NSStatusItem)가 Cocoa API 라 cgo 를 거치고, cgo 는 크로스
빌드가 안 된다. Xcode Command Line Tools 만 있으면 된다 — `xcode-select --install`.

```bash
git clone https://github.com/dancingowl/token_usage_tray.git
cd token_usage_tray
./build.sh          # dist/UsageTray.app (arm64 + amd64 유니버설)
open dist/UsageTray.app
```

`build.sh` 가 `assets/icon.png` 에서 `AppIcon.icns` 를 구워 번들에 넣는다(`sips` · `iconutil`
둘 다 맥 기본 도구다). Finder·로그인 항목에 뜨는 그림이고, 메뉴바 링과는 무관하다.

## 파일이 놓이는 곳

`~/Library/Application Support/usage-tray/` — 맥은 한 프로세스로 도니 접미사 없이
`state.json` · `status.txt` · `tokens.json` · `usage-tray.log` 다.

## 맥에서만 걸리는 것

Claude 자격증명이 Keychain 에 있을 수 있다. 파일에서 못 찾으면 후보 서비스명
(`Claude Code-credentials` · `Claude Code` · `claude-code`)을 시도한다. 그래도 안 되면:

```bash
security dump-keychain | grep -i -A2 claude | head -20
export USAGE_TRAY_CLAUDE_KEYCHAIN_SERVICE="찾은 이름"
```

---

# 공통

## 값은 어디서 오나

| 에이전트 | 엔드포인트 |
|---|---|
| Claude | `api.anthropic.com/api/oauth/usage` |
| Codex | `chatgpt.com/backend-api/codex/usage` |
| Antigravity | 로컬에서 도는 Antigravity 언어 서버 (`127.0.0.1` 루프백, macOS 만) |

Claude · Codex 는 비공개 경로다. 실패하면 로컬 기록으로 폴백하고 화면에 출처를 표시한다.
Antigravity 는 앱이 떠 있을 때만 잡히고, 자격증명을 루프백 밖으로 보내지 않으며 리디렉션도
따르지 않는다. 모델별 잔여 한도만 읽고 토큰 수나 과거 사용량은 수집하지 않는다.

호출은 아낀다 — statusline 이 받아 둔 캐시가 2분 이내면 그걸 쓰고, 직접 부를 때도 최소
간격(Claude 3분 · Codex 2분)을 지키며, 429 를 받으면 5분부터 두 배씩 최대 30분 쉰다.
실패해도 화면을 비우지 않고 직전 정상값에 나이를 붙인다.

## 자격증명을 찾는 자리

| | Windows | macOS |
|---|---|---|
| Claude | `%LOCALAPPDATA%\Claude Code\credentials.json` → `~\.claude\.credentials.json` | `~/.claude/.credentials.json` → `~/Library/Application Support/Claude Code/credentials.json` → Keychain |
| Codex | `~\.codex\auth.json` | `~/.codex/auth.json` |

`CLAUDE_CODE_OAUTH_TOKEN` 이 있으면 그게 우선이고, `CLAUDE_CONFIG_DIR` 을 주면 `~/.claude`
대신 그 폴더를 본다.

맥에서는 파일과 Keychain 에서 못 찾으면 Claude 데스크톱 앱의 OAuth 캐시
(`~/Library/Application Support/Claude/config.json`)를 마지막으로 본다. Electron safeStorage
로 잠긴 **지정된 필드만** 메모리에서 푼다 — 브라우저 쿠키나 Keychain 전체를 뒤지지 않고,
복호화 키와 토큰은 파일에도 로그에도 남기지 않는다. macOS 가 `Claude Safe Storage` 접근을
물으면 허용해야 한다. 그마저 막히면 같은 앱의 24시간 이내 사용 기록
(`plan-usage-history.json`)을 '앱 기록' 으로 표시한다 — 실시간 조회가 아니고 리셋 시각이
없어 미확인으로 나온다.

## 아이콘 개수가 OS 마다 다른 이유

트레이 API 는 프로세스당 아이콘 1개만 준다. 링을 여러 개 내려면 프로세스를 갈라야 한다.

**윈도우는 가른다.** 실행 파일이 자기 자신을 `-only claude` · `-only codex` 로 띄우고 부모는
빠진다. 작업표시줄에는 아이콘이 나란히 붙어도 문제가 없고, 에이전트마다 링 하나가 한눈에
낫다. 상세 화면은 어느 쪽을 눌러도 전부 합쳐 보여 준다
(claude 47113 · codex 47114). 에이전트 하나만 사용하는 기본 실행은 47113을 사용한다.
`-only antigravity`의 47115는 macOS에서만 사용한다.

**맥은 못 가른다.** `.app` 본체가 자식을 띄우고 빠지는 순간 LaunchServices 가 앱이 끝난 것으로
보고, 번들 밖에서 도는 자식은 `NSStatusItem` 을 못 얻는다 — 메뉴바에 **아무것도 안 뜬다.**
0.3.0 이전 맥 빌드가 실행은 되는데 메뉴바가 비어 있던 이유가 이것이다. 그래서 맥은 한
프로세스가 링 하나를 띄우고 그 링에 **남은량이 가장 적은** 에이전트를 그린다. 툴팁과 상세
화면(47113)에는 쓰는 에이전트를 모두 담는다.

어느 쪽이든 `-only` 로 띄우면 그 에이전트 하나만 맡는다.

## 플래그

| 플래그 | 하는 일 |
|---|---|
| `-once` | 한 줄 요약과 툴팁 내용을 출력하고 끝낸다 |
| `-tokens` | 토큰 집계를 JSON 으로 출력 |
| `-only claude` / `-only codex` / `-only antigravity` | 한쪽만 다룬다 |
| `-icons <폴더>` | 아이콘 견본을 PNG/ICO 로 떨어뜨린다 |
| `-notify` | 알림을 한 번 띄워 본다 |
| `-install` | 로그인 자동 실행 + 아이콘 꺼내기 + 지금 띄우기를 한 번에 |
| `-uninstall` | `-install` 로 건 자동 실행을 뗀다 |
| `-promote` | 아이콘을 작업표시줄에 고정한다(윈도우). `-restart-explorer` 와 함께 쓰면 explorer 까지 재시작 |

## 문제 해결

| 증상 | 조치 |
|---|---|
| 아무것도 안 뜬다 | 자격증명이 없다. `usage-tray -once` 로 확인 |
| 링이 오버플로(∧) 안에만 있다 | `-install` 을 한 번 돌린다(윈도우) |
| 로그인해도 안 뜬다 | 실행 파일을 옮겼다면 새 자리에서 `-install` 을 다시 돌린다 |
| 이미 실행 중이라고 나온다 | 그 포트를 쓰는 인스턴스가 이미 있다 |
| Codex 가 `기록 N분 전` 으로만 나온다 | 토큰이 만료됐을 수 있다. `codex` 를 한 번 실행 |
| 값이 안 변한다 | 최소 간격 탓이다(3분). 우클릭 → 지금 갱신 |
| 토큰 표가 비어 있다 | 첫 스캔이 아직이다. `-tokens` 로 직접 돌려 본다 |

로그는 데이터 폴더의 `usage-tray.log` 이고 실패 사유만 쌓인다.
