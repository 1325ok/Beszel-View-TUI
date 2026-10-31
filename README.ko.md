# Beszel View

[English](README.md) | [한국어](README.ko.md)

[Beszel](https://github.com/henrygd/beszel) Hub의 여러 서버 상태를 표시하는 비공식 CLI/TUI 클라이언트입니다.

- 전체 현황, 장비 상세, 장애 경고 화면 자동 순환
- 모든 장비의 CPU, 메모리, 디스크 사용량 확인
- 모든 장비를 한 화면에 표시하는 단일 모드

## 빠른 시작

예제 설정을 복사한 뒤 Hub URL과 인증정보를 입력합니다.

```bash
cp config.example.json config.json
```

Linux:

```bash
chmod +x beszel-monitor-linux-amd64
./beszel-monitor-linux-amd64 --config config.json
```

Windows:

```powershell
Copy-Item config.example.json config.json
.\beszel-monitor.exe --config config.json
```

Go로 설치:

```bash
go install github.com/1325ok/Beszel-View-TUI/cmd/beszel-monitor@latest
```

## 레이아웃과 화면 크기

| 레이아웃 | 용도 | 최소 크기 |
|---|---|---:|
| `small` | 작은 LCD, 핵심 정보 중심 | `38x13` |
| `large` | 표와 그래프가 포함된 관제 화면 | `80x22` |
| `auto` | `100x30` 이상이면 large, 아니면 small | `38x13` |

```bash
beszel-monitor --config config.json --layout small --size 50x16
beszel-monitor --config config.json --layout large --size 80x22
beszel-monitor --config config.json --layout auto --size auto
```

## 단일 모드

화면 순환 없이 한 페이지에 장비 상태를 표시합니다.

```bash
beszel-monitor --config config.json --single --size 80x24
```

`config.json`에서 `"single_mode": true`를 설정하거나 `BESZEL_SINGLE=true` 환경변수를 사용할 수도 있습니다. 장비가 화면에 모두 들어가지 않으면 `+N MORE`가 표시되며, 화면 높이를 늘리면 더 많은 행을 볼 수 있습니다.

> 목록 순환 및 페이지 기능은 추가 예정입니다.

## 주요 옵션

| 옵션 | 기본값 | 설명 |
|---|---:|---|
| `--layout` | `auto` | `small`, `large`, `auto` |
| `--size` | `auto` | `80x22` 형식의 터미널 크기 |
| `--single` | `false` | 단일 화면 모드 |
| `--refresh` | `3s` | API 갱신 주기 |
| `--rotation` | `5s` | 화면 전환 주기 |
| `--alert-duration` | `20s` | 경고 화면 표시 시간 |
| `--alert-blink` | `750ms` | 경고 점멸 주기 |

전체 옵션은 `beszel-monitor --help`로 확인할 수 있습니다.

## 환경변수

```text
BESZEL_HUB_URL
BESZEL_EMAIL
BESZEL_PASSWORD
BESZEL_TOKEN
BESZEL_LAYOUT
BESZEL_SIZE
BESZEL_SINGLE
BESZEL_REFRESH
BESZEL_ROTATION
BESZEL_ALERT_DURATION
BESZEL_ALERT_BLINK
BESZEL_INSECURE_TLS
```

CLI 옵션, 환경변수, `config.json` 순서로 우선 적용됩니다.

## 빌드

```bash
go test ./...
go vet ./...
go build -o beszel-monitor ./cmd/beszel-monitor
```

## 라이선스

MIT License. 자세한 내용은 [LICENSE](LICENSE)를 참고하세요.
