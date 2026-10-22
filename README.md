# Usage Tray · token_usage_tray

Claude Code·Codex·Antigravity의 **남은 사용 한도와 초기화 시각**을 Windows 작업표시줄 또는 macOS 메뉴바에서 확인하는 앱입니다. 브라우저 상세 화면에서는 한도 이력, 사용 전망, 로컬 기록 기반 토큰 통계를 제공합니다.

[inging88/usage_tray](https://github.com/inging88/usage_tray)를 원작자의 허락을 받아 [dancingowl/token_usage_tray](https://github.com/dancingowl/token_usage_tray)라는 독립 저장소로 옮겼습니다. 기존 코드와 커밋 이력, 원본 프로젝트에 대한 출처를 보존합니다.

## 지원 환경

| 환경 | 연결 서비스 | 아이콘과 배포 형식 |
|---|---|---|
| Windows | Claude Code, Codex | 서비스별 트레이 아이콘 · x64 단일 `usage-tray.exe` |
| macOS | Claude Code, Codex, Antigravity | 통합 메뉴바 아이콘 · Apple Silicon/Intel 유니버설 `UsageTray.app` |

이미 로그인한 서비스의 자격증명을 읽으므로 앱에서 별도로 로그인할 필요는 없습니다. 연결 가능한 서비스가 없어도 트레이에서 연결을 기다리며, 로그인 흔적을 약 15초마다 확인해 자동으로 연결합니다. Antigravity 연결은 macOS에서만 지원합니다.

실행 파일에는 Go나 Node.js 설치가 필요하지 않습니다. **소스 빌드에는 Go 1.26 이상**이 필요하고, macOS에는 Xcode Command Line Tools도 필요합니다. 지원 OS는 빌드한 Go 버전의 요구사항을 따릅니다. 자세한 조건은 [개발 문서](docs/development.md)를 참고하세요.

## 빠른 시작

실행 파일을 내려받으면 Go나 Node.js 설치 없이 사용할 수 있습니다.

- **Windows x64:** [UsageTray-windows-amd64.zip](https://github.com/dancingowl/token_usage_tray/releases/latest/download/UsageTray-windows-amd64.zip)을 계속 사용할 폴더에 풀고 `usage-tray.exe`를 실행합니다.
- **macOS Apple Silicon / Intel:** [UsageTray-macos-universal.zip](https://github.com/dancingowl/token_usage_tray/releases/latest/download/UsageTray-macos-universal.zip)을 풀고 `UsageTray.app`을 응용 프로그램 폴더로 옮겨 실행합니다.

변경 내용과 버전별 파일은 [Releases](https://github.com/dancingowl/token_usage_tray/releases)에서 확인합니다. 아래 소스 빌드 예시는 기본 브랜치 **`main`**을 사용합니다.

### Windows에서 빌드하고 실행

Git과 Go를 설치한 뒤 PowerShell에서 실행합니다.

```powershell
git clone --branch main https://github.com/dancingowl/token_usage_tray.git
cd token_usage_tray
powershell -ExecutionPolicy Bypass -File .\build.ps1 -Run
```

결과는 `dist\usage-tray.exe`입니다. 이후에는 이 파일을 실행하면 됩니다. `-Run`을 빼면 빌드만 하며, 일반 실행은 로그인 자동 실행을 등록하지 않습니다.

### macOS에서 빌드하고 실행

Git·Go·Xcode Command Line Tools가 설치된 macOS 터미널에서 실행합니다.

```bash
git clone --branch main https://github.com/dancingowl/token_usage_tray.git
cd token_usage_tray
bash ./build.sh
open dist/UsageTray.app
```

다운로드한 앱의 실행, 로그인 자동 실행, 종료 방법은 [사용법](docs/usage.md), 실행이 차단되거나 아이콘이 보이지 않으면 [문제 해결](docs/troubleshooting.md)을 참고하세요.

## 주요 기능

- **빠른 한도 확인:** 도넛 또는 숫자로 잔량을 표시합니다. Windows에서는 한 번 클릭해 요약을, 더블클릭해 상세 화면을 엽니다.
- **한도 이력:** 최근 1일·7일·30일을 보고 실제 기록이 있는 구간을 확대합니다.
- **사용 전망:** 관측 사용량·하루 환산 속도·초기화까지의 하루 예산을 구분합니다. 최근 사용 속도가 크게 달라지면 소진 시점 추정을 보류합니다.
- **토큰 통계:** 최근 7·14·30일을 같은 기준으로 집계하고 모델·프로젝트 필터와 전체 목록을 제공합니다. 잔여 한도 비율과는 별도 정보입니다.
- **설정과 알림:** 표시할 서비스·아이콘·수집 간격·알림·로그인 자동 실행·테마는 변경하면 바로 저장합니다. 숫자는 유효한 입력이 끝난 뒤 저장하며, 저장 상태와 실패 시 재시도를 안내합니다. 알림은 1시간 또는 오늘까지 일시 중지할 수 있습니다.
- **자동 업데이트:** Windows x64 정식 버전은 시작 직후와 6시간마다 GitHub 정식 릴리스를 확인하고 검증한 새 버전으로 교체·재시작합니다. macOS는 릴리스 페이지에서 수동으로 업데이트합니다.
- **화면 유지:** 그래프·토큰 조회 조건을 기억하고 앱 재시작을 감지해 화면을 갱신합니다.
- **갱신 상태:** 조회 중·완료·실패와 최소 간격·요청 제한에 따른 다음 조회 대기를 구분합니다.

## 문서

| 문서 | 내용 |
|---|---|
| [사용법](docs/usage.md) | 실행·트레이 조작·상세 화면·설정·알림·데이터 위치 |
| [개발](docs/development.md) | 빌드·테스트·아이콘 재생성·vendor 확장·버전과 CI |
| [문제 해결](docs/troubleshooting.md) | 실행 차단·아이콘·로그인·갱신 지연·진단·삭제 |
| [과거 macOS 확장판 기록](UPGRADE.md) | 기존 1.1.0 확장판의 변경 배경과 연결 조건 |

## 데이터와 출처

Claude·Codex의 사용 한도는 서비스의 비공개 조회 경로에, Antigravity는 실행 중인 로컬 앱에 의존합니다. 서비스 변경으로 조회가 실패하면 화면에 출처와 상태를 표시합니다. [연결 조건과 데이터 설명](docs/usage.md)을 확인하세요.

앱의 민트색 계기판과 요약 카드의 두 버튼은 [Lucide](https://lucide.dev/) 아이콘을 사용합니다. 계기판의 출처·라이선스는 [assets/LICENSE.lucide.txt](assets/LICENSE.lucide.txt), 버튼 아이콘은 [원본과 재생성 안내](vendor/fyne.io/systray/icons/README.md)에 보존합니다. 외부 라이브러리의 README와 라이선스는 `vendor/`에 원문으로 유지합니다.
