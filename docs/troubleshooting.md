# 문제 해결

[프로젝트 소개](../README.md) · [사용법](usage.md) · [개발](development.md)

## 먼저 확인할 것

1. 실행한 파일의 위치와 버전을 확인합니다. 다른 폴더의 예전 앱이 실행 중이면 트레이 메뉴에서 종료합니다.
2. Windows는 숨겨진 아이콘 영역(∧), macOS는 메뉴바를 확인합니다.
3. 상세 화면이 열리면 **설정 → 앱 정보 → 연결 상태**에서 마지막 성공 시각과 오류를 확인합니다.
4. 화면이 열리지 않으면 아래 로그를 확인합니다. 연결 가능한 서비스가 없어도 앱은 연결 대기 상태로 유지됩니다.

## 실행이 차단될 때

### Windows

이 프로젝트의 로컬 빌드는 코드 서명을 하지 않습니다. 경고 문구에 따라 원인과 대응이 다릅니다.

| 표시 또는 상황 | 확인 방법 |
|---|---|
| Microsoft Defender SmartScreen의 PC 보호 경고 | 다운로드 출처와 파일을 확인하고 경고가 제공하는 선택지를 확인합니다. 모든 경고에 실행 허용 버튼이 있는 것은 아닙니다. |
| Smart App Control 또는 Application Control에 의한 차단 | SmartScreen 경고와 구분합니다. Windows 보안의 해당 기능 또는 조직의 앱 제어 정책을 확인합니다. 조직 정책으로 관리되는 PC는 관리자에게 문의합니다. |
| 빌드 성공 후 테스트 실행 파일도 차단됨 | Go 컴파일 실패와 실행 정책 차단을 구분합니다. 직접 빌드한 파일도 자동으로 허용되는 것은 아닙니다. |

`-ExecutionPolicy Bypass`는 PowerShell 스크립트 실행 옵션이며 exe에 대한 앱 제어 정책을 해제하지 않습니다. Microsoft는 Smart App Control에 개별 앱 예외 기능을 제공하지 않는다고 안내합니다. [Microsoft의 앱 및 브라우저 제어 안내](https://support.microsoft.com/en-us/windows/security/windows-security/app-browser-control-in-the-windows-security-app), [Smart App Control FAQ](https://support.microsoft.com/en-us/windows/security/threat-malware-protection/smart-app-control-frequently-asked-questions)를 참고하세요.

### macOS

제공 빌드 스크립트는 로컬 ad-hoc 서명을 시도하며 Apple 공증을 수행하지 않습니다. 개발자 확인 경고가 나오면 앱의 출처를 확인하고 **시스템 설정 → 개인정보 보호 및 보안**에서 해당 앱에 대한 허용 항목을 확인합니다. [Apple의 앱 실행 안내](https://support.apple.com/en-ie/102445)를 따르세요.

OS가 지원 범위보다 오래된 경우에는 보안 경고를 허용해도 해결되지 않습니다. [개발 문서의 Go·macOS 요구사항](development.md)을 확인하세요.

## 아이콘과 상세 화면

| 증상 | 확인 또는 조치 |
|---|---|
| 아이콘이 보이지 않음 | 숨겨진 아이콘 영역과 로그를 확인합니다. 계정이 없어도 연결 대기 아이콘은 남습니다. Antigravity 연결은 macOS 전용입니다. |
| 계정 연결 대기 | 해당 서비스에서 로그인한 뒤 약 15초 기다리거나 지금 갱신합니다. 아이콘이 나타나도 설치 흔적만 발견했을 수 있으므로 연결 상태의 인증 오류도 확인합니다. |
| Windows에서 아이콘이 ∧ 안에만 있음 | 아이콘을 작업표시줄로 끌어 놓거나 Windows의 작업표시줄 설정에서 표시합니다. `-install`은 자동 실행 등록도 함께 하므로 고정만 원하는 경우와 구분합니다. |
| Windows에서 아이콘이 여러 개 보임 | 여러 서비스를 사용하는 경우 서비스별 프로세스·아이콘은 정상입니다. 같은 서비스가 다른 exe 경로에서 중복 실행되는지는 확인합니다. |
| 상세 화면 연결이 거부됨 | 앱이 실행 중인지 확인하고 트레이에서 상세 화면을 다시 엽니다. 앱이 종료되면 로컬 웹 서버도 종료됩니다. Windows가 단일 프로세스에서 서비스별 프로세스로 전환하는 동안에는 잠시 연결이 끊길 수 있습니다. |
| 재시작 후 예전 화면이 남음 | 화면을 다시 보이게 하면 앱 변경을 확인합니다. 미저장 설정 안내가 있으면 저장하거나 취소한 뒤 화면 갱신을 누릅니다. |
| 그래프·토큰 조건이 초기화됨 | 같은 브라우저와 로컬 포트인지, 브라우저 저장소를 지우거나 차단했는지 확인합니다. 토큰 주소에 적힌 조건이 기억한 조건보다 우선합니다. |
| 테마를 바꾼 뒤 변경 취소해도 유지됨 | 상단 테마 아이콘의 선택은 즉시 별도로 저장됩니다. 변경 취소는 설정 폼의 미저장 변경만 되돌립니다. |
| 앱 테마와 트레이 숫자 색이 다름 | 상세 화면·Windows 요약 카드는 앱 테마, 숫자는 작업표시줄 밝기를 따릅니다. Windows의 앱 모드와 Windows 모드가 다르면 정상입니다. |
| 이미 실행 중이라는 기록이 있음 | 동일한 로컬 포트를 다른 인스턴스나 프로그램이 사용 중인지 확인합니다. 데이터 폴더만 바꿔도 포트는 분리되지 않습니다. |
| 로그인 시 실행되지 않음 | 실행 파일을 이동했다면 새 위치의 앱에서 자동 실행을 다시 저장합니다. Windows 시작프로그램 또는 macOS 로그인 항목의 비활성화 여부도 확인합니다. |

기본 상세 화면 주소는 `http://127.0.0.1:47113/`입니다. `-only claude`, `-only codex`, `-only antigravity`는 각각 47113·47114·47115 포트를 사용합니다. Windows에서 여러 서비스가 분리 실행되면 해당 서비스의 포트를 사용하며, 어느 상세 화면이든 데이터를 합쳐 보여 줍니다. 브라우저 주소를 추측하기보다 트레이에서 여는 것이 간단합니다.

## 사용량이 갱신되지 않거나 예측이 없을 때

| 증상 | 확인 또는 조치 |
|---|---|
| 새로고침해도 잔량이 같음 | 갱신 완료인지 조회 대기인지 먼저 확인합니다. Claude는 최소 3분, Codex·Antigravity는 최소 2분 간격을 유지하며 수동 갱신도 지킵니다. 대기 메시지와 설정의 다음 조회 시각을 확인합니다. 정수 퍼센트가 바뀔 만큼 사용하지 않았을 수도 있습니다. |
| Claude 요청 제한·일시 오류가 반복됨 | Claude의 HTTP 429·503에는 5분부터 최대 30분까지 재시도 대기가 적용됩니다. 연결 상태와 마지막 성공 시각을 확인합니다. |
| `기록`·`캐시`·로그인 확인 필요가 표시됨 | 실제 최신 조회와 구분합니다. 해당 서비스의 공식 앱 또는 CLI에서 로그인을 확인한 뒤 다시 조회합니다. |
| 토큰 통계가 비어 있음 | 초기 집계 완료와 로컬 대화 기록을 확인합니다. 선택한 7·14·30일과 모델·프로젝트 필터에 일치하는 기록이 있는지도 확인합니다. Antigravity 토큰 수는 집계하지 않습니다. |
| 모델·프로젝트가 일부만 보임 | 기본 표는 상위 6개입니다. 모델·프로젝트 전체 보기를 켜면 선택한 기간·필터의 전체 목록을 봅니다. |
| 예상 소진·환산 속도가 없음 | 같은 한도 주기의 기록 3개 이상, 관측 30분 이상, 잔량 감소 1%p 이상이 필요합니다. 오래된 관측·캐시·초기화 직후 등에는 예상이 숨겨집니다. |
| 사용 속도 변화로 소진 시점 추정 보류 | 최근 구간 속도가 전체 관측 평균과 크게 달라 신뢰하기 어려운 시점 계산을 보류한 상태입니다. 관측 사용량·최근 속도·하루 예산을 확인하고 이후 기록을 기다립니다. |
| 하루 환산 속도가 실제 하루 사용량보다 커 보임 | 짧은 관측 구간의 감소량을 24시간으로 환산한 값입니다. 실제 사용량과 시작·종료 시각은 **관측 기록**에서 확인합니다. |

알림이 오지 않으면 **설정 → 알림 일시 중지**의 종료 시각과 알림 전체·한도별 사용 여부를 확인합니다.
1시간 또는 오늘까지 중지는 종료 시각 후 자동 재개하며, 중지 중에 기존 알림 기준을 삭제하지 않습니다.

## 로그인과 자격증명

Claude Code 또는 Codex에서 로그인을 확인한 뒤 앱을 다시 조회합니다. 자격증명 파일을 직접 고치거나 토큰 값을 복사하는 대신 해당 서비스의 로그인 절차를 사용합니다.

| 서비스 | Windows에서 확인하는 위치 | macOS에서 확인하는 위치 |
|---|---|---|
| Claude Code | `%LOCALAPPDATA%\Claude Code\credentials.json` → `~/.claude/.credentials.json` | `~/.claude/.credentials.json` → `~/Library/Application Support/Claude Code/credentials.json` → Keychain |
| Codex | `~/.codex/auth.json` | `~/.codex/auth.json` |

`~`는 사용자 홈 폴더입니다. `CLAUDE_CONFIG_DIR`을 지정하면 Claude의 기본 `~/.claude` 경로를 대신합니다. `CLAUDE_CODE_OAUTH_TOKEN`이 설정되어 있으면 Claude 파일·Keychain보다 우선하므로 오래된 환경변수가 남아 있는지도 확인합니다.

macOS에서는 위 후보에서 토큰을 찾지 못하면 Claude 데스크톱 앱의 지정된 OAuth 캐시를 확인합니다. Keychain의 `Claude Safe Storage` 접근 권한이 필요할 수 있습니다. 사용 가능한 인증을 찾지 못하면 24시간 이내의 로컬 사용 기록으로 폴백할 수 있으며, 이 기록에는 초기화 시각이 없고 실시간 조회와 구분해 표시합니다.

macOS Claude Keychain 접근은 키체인 접근 앱에서 해당 서비스 항목과 권한을 확인합니다. 자동 탐색 후보(`Claude Code-credentials` · `Claude Code` · `claude-code`)와 다른 서비스명을 사용하는 경우 `USAGE_TRAY_CLAUDE_KEYCHAIN_SERVICE`로 지정할 수 있습니다. 이 변수에는 암호나 토큰 대신 서비스명을 넣습니다.

```bash
export USAGE_TRAY_CLAUDE_KEYCHAIN_SERVICE="확인한 서비스명"
/Applications/UsageTray.app/Contents/MacOS/usage-tray -once
```

기존 앱을 종료한 뒤 같은 터미널에서 이 진단 명령을 실행해 조회 결과를 확인합니다. 환경변수는 해당 터미널에서 실행한 프로세스에 적용되며 설정 파일에 저장되는 항목은 아닙니다.

## 로그와 진단 출력

기본 로그 위치는 Windows의 `%LOCALAPPDATA%\usage-tray\usage-tray.log`, macOS의 `~/Library/Application Support/usage-tray/usage-tray.log`입니다. `USAGE_TRAY_DATA_DIR`을 지정했다면 그 폴더를 확인합니다. 로그에는 시작·종료·서비스 감지와 조회 실패 등 진단 기록이 쌓입니다.

```powershell
# Windows: 최근 로그 확인
Get-Content (Join-Path $env:LOCALAPPDATA 'usage-tray\usage-tray.log') -Tail 50
```

```bash
# macOS: 최근 로그 확인
tail -n 50 "$HOME/Library/Application Support/usage-tray/usage-tray.log"
```

Windows 배포 exe는 콘솔 창을 만들지 않으므로 `-once`·`-tokens`의 표준 출력이 보이지 않을 수 있습니다. 배포 사용자에게는 로그·연결 상태·데이터 파일 확인을 권장합니다. 소스와 Go가 있는 경우 저장소 루트에서 다음을 실행합니다.

```powershell
go run . -once
go run . -tokens
```

`-once`는 사용량을 조회하고 상태를 저장한 뒤 종료하며, HTTP 서버와 트레이는 띄우지 않습니다. 문제를 보고할 때는 OS·앱 버전·빌드 방식·오류 시각과 관련 로그를 함께 제공하고, 로그인 토큰이나 자격증명 파일 내용은 첨부하지 않습니다.

## 자동 실행 해제와 삭제

1. 실행 파일을 지우기 전에 **설정 → 일반 → 로그인 시 실행**을 끄고 **저장**합니다. 또는 아래 OS별 명령으로 자동 실행을 해제합니다.
2. 트레이·메뉴바 메뉴에서 앱을 종료합니다. Windows에서 여러 서비스가 실행 중이면 각 프로세스가 종료됐는지 확인합니다.
3. exe 또는 `.app`을 삭제합니다. 사용 기록·설정도 지우려면 [사용법의 데이터 폴더](usage.md)에 안내된 폴더를 함께 삭제합니다.

```powershell
# Windows: exe가 있는 폴더에서 실행
.\usage-tray.exe -uninstall
```

```bash
# macOS: 설치 위치가 다르면 경로를 맞춥니다.
/Applications/UsageTray.app/Contents/MacOS/usage-tray -uninstall
```

`-uninstall`은 자동 실행만 해제하며 실행 중인 앱을 종료하거나 기록을 삭제하지 않습니다. Windows의 작업표시줄 표시 설정도 유지합니다.
