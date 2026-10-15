package main

// 앱 아이콘. 계기판 원본은 assets/icon.svg, 래스터 원본은 icon.png(1024px)다.
//
//	assets/icon.ico      → rsrc_windows.syso 로 exe 에 박힌다(탐색기·작업표시줄·Alt+Tab).
//	assets/favicon32.png → 상세 화면 파비콘. 여기서 바이너리에 심는다.
//	assets/icon.png      → build.sh 가 맥에서 .icns 로 굽는다(sips·iconutil).
//
// 트레이/메뉴바는 남은량을 도넛 또는 숫자로 표시한다(icon.go).
//
// SVG를 바꾸면 icon.png와 16/20/24/32/48/64/128/256px ICO, favicon32.png를 다시 만든다.
// Lucide 계기판의 출처·라이선스는 assets/LICENSE.lucide.txt에 보존한다.
//
//	go run github.com/akavel/rsrc@latest -ico assets/icon.ico -arch amd64 -o rsrc_windows.syso

import _ "embed"

//go:embed assets/favicon32.png
var faviconPNG []byte
