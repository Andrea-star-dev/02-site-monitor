# 웹페이지 가격/게시글 변경 감지 알리미

[![CI](https://github.com/Junoh-Lee111/02-site-monitor/actions/workflows/ci.yml/badge.svg)](https://github.com/Junoh-Lee111/02-site-monitor/actions/workflows/ci.yml)

지정한 웹페이지의 특정 부분(가격, 공고 제목 등)을 주기적으로 확인하고,
값이 바뀌면 텔레그램으로 알려주는 도구입니다.

## 이런 분께 유용합니다

- 경쟁사나 특정 상품의 가격이 바뀌면 바로 알고 싶다면
- 특정 게시판에 새 글/공고가 올라오면 알림을 받고 싶다면
- 매번 사이트를 열어서 눈으로 확인하는 게 번거롭다면

## 무엇을 만들어 주나요

1. **변경 알림** — 텔레그램 봇 토큰을 설정하면 값이 바뀔 때 텔레그램 메시지로 알려줍니다.
   설정하지 않으면 콘솔(터미널)에 출력되므로, 텔레그램 없이도 바로 사용해볼 수 있습니다.
2. **변경 이력** — 모든 값과 변경 기록이 SQLite 파일 하나에 쌓입니다. 나중에 "언제 얼마였는지"를
   돌아볼 수 있습니다.
3. **매너 있는 수집** — 대상 사이트의 `robots.txt`를 확인해서 수집이 금지된 경로는 건너뛰고,
   같은 사이트에 너무 자주 요청을 보내지 않도록 간격을 둡니다.

> 결과 스크린샷 자리
>
> - [ ] 콘솔에 변경 알림이 출력된 화면
> - [ ] 텔레그램으로 온 알림 메시지 화면

## 감시 대상 설정

`config/targets.yaml.example`을 `config/targets.yaml`로 복사해서 감시하고 싶은
페이지를 등록합니다.

```yaml
targets:
  - name: "toscrape-book-light-in-attic-price"
    url: "https://books.toscrape.com/catalogue/a-light-in-the-attic_1000/index.html"
    selector: "p.price_color"      # 브라우저 개발자도구로 확인한 CSS 셀렉터
    check_interval: "30m"           # 참고용 권장 주기 (아래 "주기적으로 실행하기" 참고)
```

`selector`는 브라우저에서 F12(개발자도구) → 확인하려는 값을 우클릭 → "검사"로
해당 요소를 찾은 뒤, 클래스명이나 태그명을 CSS 셀렉터 형태로 적으면 됩니다.

## 사용 방법 (1) — Go로 직접 실행

```bash
go build -o monitor ./cmd/monitor
./monitor --targets config/targets.yaml --db data/site_monitor.db
```

실행하면 각 대상의 현재 값을 보여주고, 이전에 저장된 값과 다르면 "변경됨"으로 표시합니다.
처음 실행할 때는 비교할 이전 값이 없으므로 "최초 등록"으로만 표시되고 알림은 가지 않습니다.

## 사용 방법 (2) — Docker로 실행

```bash
docker build -t site-monitor .

docker run --rm \
  -v "$(pwd)/config/targets.yaml:/app/config/targets.yaml:ro" \
  -v "$(pwd)/data:/app/data" \
  -e TELEGRAM_BOT_TOKEN="..." \
  -e TELEGRAM_CHAT_ID="..." \
  site-monitor
```

`-v "$(pwd)/data:/app/data"`로 DB 파일을 컨테이너 밖에 보관해야, 컨테이너를 다시 실행해도
이전 기록이 유지됩니다. 컨테이너는 UID 10001로 실행되므로, 호스트의 `data` 폴더가 없다면
먼저 만들어두세요 (`mkdir -p data`). 권한 오류가 나면 `chmod 777 data`로 임시로 풀어도 됩니다.

## 텔레그램 알림 받기 (선택)

1. 텔레그램에서 [@BotFather](https://t.me/BotFather)에게 `/newbot`으로 봇을 만들고 토큰을 받습니다.
2. 만든 봇과 대화를 한 번 시작한 뒤, `https://api.telegram.org/bot<토큰>/getUpdates`에
   접속해 `chat.id` 값을 확인합니다.
3. 실행할 때 환경변수 `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`를 설정합니다.

두 값을 설정하지 않으면 알림은 자동으로 콘솔 출력으로 대체되므로, 텔레그램 없이도
안전하게 테스트할 수 있습니다.

## 주기적으로 실행하기

이 프로그램은 "한 번 점검하고 종료"하는 방식입니다. 계속 감시하려면 OS의 스케줄러로
주기적으로 실행하세요.

- macOS/Linux: `cron` 또는 `systemd timer`
- Docker: `docker run --rm ...`을 cron으로 주기적으로 호출, 또는 호스트의 작업 스케줄러 사용

예) 30분마다 실행하는 crontab 한 줄:

```
*/30 * * * * cd /path/to/02-site-monitor && ./monitor --targets config/targets.yaml --db data/site_monitor.db
```

## 샘플로 바로 테스트해보기

`config/targets.yaml.example`에는 `books.toscrape.com`(크롤링 연습용 공개 사이트)의 책
가격 2개가 이미 등록되어 있습니다. 복사해서 바로 실행해보면 실제 동작을 확인할 수 있습니다.

## 개발/테스트

```bash
go test ./...
```
