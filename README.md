# ggo

GitHub App installation token으로 특정 GitHub issue를 읽고, marker가 포함된 봇 댓글이 없을 때 고정 댓글을 작성하는 Go CLI입니다.

## 기능

- YAML 설정 파일 로드
- GitHub App JWT 생성
- installation access token 발급
- issue 조회
- issue comments 조회
- 기존 댓글에 marker가 있으면 중단
- dry-run 모드에서 작성 예정 댓글 출력
- dry-run이 아니면 issue comment 생성

## GitHub App 생성

`ggo`는 GitHub App ID `3675420`을 소스에 고정해서 사용합니다. 이 값은 secret이 아니므로 설정 파일이나 `.env`에 넣을 필요가 없습니다.

1. GitHub에서 `Settings` > `Developer settings` > `GitHub Apps` > `New GitHub App`으로 이동합니다.
2. App 이름과 Homepage URL을 입력합니다. 이 CLI는 서버, Webhook, OAuth를 사용하지 않으므로 Webhook은 비활성화해도 됩니다.
3. `Repository permissions`에서 다음 권한을 설정합니다.
   - `Issues`: `Read and write`
   - private repository에서 사용할 경우 `Metadata`: `Read-only`는 기본으로 필요합니다.
4. App을 생성한 뒤 `App ID`를 확인합니다.
5. `Private keys`에서 private key를 생성하고 PEM 파일을 로컬에 저장합니다.
6. `Install App`에서 대상 organization 또는 repository에 App을 설치합니다.
7. 설치 후 URL 또는 API 응답에서 `installation_id`를 확인합니다.

## 설정 파일 위치

설정 파일 이름은 `ggo.yaml`입니다. 위치는 홈 디렉터리에 고정하지 않고 아래 순서로 찾습니다.

1. `--config`로 넘긴 경로
2. `GGO_CONFIG` 환경변수 경로
3. 현재 디렉터리의 `./ggo.yaml`
4. `~/.ggo/ggo.yaml`

개인용 전역 CLI로 쓸 때는 `~/.ggo/ggo.yaml`을 추천합니다. 프로젝트마다 다른 설정을 쓰고 싶으면 해당 프로젝트 디렉터리에 `ggo.yaml`을 두면 됩니다.

`ggo.yaml` 예시:

```yaml
github:
  installation_id: 987654
  private_key_path: ./github-app.private-key.pem

bot:
  marker: "<!-- ggo:v1 -->"
  dry_run: true
```

`private_key_path`가 상대 경로이면 설정 파일이 있는 디렉터리를 기준으로 해석합니다.

## 환경변수와 .env

`ggo`는 실행할 때 `.env`가 있으면 자동으로 읽습니다. 이미 shell에 설정된 환경변수는 `.env` 값으로 덮어쓰지 않습니다.

읽는 순서:

1. 현재 디렉터리의 `./.env`
2. `~/.ggo/.env`

지원하는 환경변수:

```bash
GGO_INSTALLATION_ID=987654
GGO_PRIVATE_KEY_PATH=./github-app.private-key.pem
GGO_MARKER="<!-- ggo:v1 -->"
GGO_DRY_RUN=false
GGO_CONFIG=./ggo.yaml
```

현재 프로젝트에서 쓰던 이름도 호환됩니다.

```bash
GGOBOONG_INSTALLATION_ID=987654
GGOBOONG_PRIVATE_KEY_PATH=./github-app.private-key.pem
```

환경변수는 `ggo.yaml` 값을 덮어씁니다. 환경변수의 private key 경로가 상대 경로이면 현재 실행 디렉터리를 기준으로 해석됩니다.

## 전역 설치

이 저장소에서 설치 스크립트를 실행하면 `ggo`를 빌드해서 `~/.local/bin/ggo`로 복사합니다.

```bash
./install.sh
```

원하는 설치 위치가 있으면 `GGO_INSTALL_DIR`로 바꿀 수 있습니다.

```bash
GGO_INSTALL_DIR=/usr/local/bin ./install.sh
```

`~/.local/bin`이 `PATH`에 없다면 shell profile에 추가합니다.

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## 로컬 설정 설치

PEM을 바이너리 안에 넣어서 빌드하지 마세요. private key는 로컬 파일로 보관하고, 권한을 좁게 둔 뒤 CLI가 읽게 하는 편이 안전합니다.

`ggo login`은 GitHub App 값을 받아 `~/.ggo/ggo.yaml`을 만들고 PEM을 `~/.ggo/github-app.private-key.pem`로 복사합니다.

```bash
ggo login \
  --installation-id 987654 \
  --private-key ./github-app.private-key.pem
```

이미 `.env`에 값이 있으면 플래그를 생략할 수 있습니다.

```bash
ggo login --force
```

생성되는 파일:

```text
~/.ggo/ggo.yaml
~/.ggo/github-app.private-key.pem
```

둘 다 현재 사용자만 읽고 쓸 수 있도록 저장합니다.

## 실행

의존성을 내려받고 빌드합니다.

```bash
go mod tidy
go build -o ggo ./cmd/ggo
```

dry-run 실행:

```bash
./ggo run --owner my-org --repo my-repo --issue 123 --config ggo.yaml --dry-run
```

실제 댓글 작성:

```bash
./ggo run --owner my-org --repo my-repo --issue 123 --config ggo.yaml
```

현재 디렉터리에 `ggo.yaml`이 있거나 `~/.ggo/ggo.yaml`을 만들어 둔 경우 `--config`를 생략할 수 있습니다.

```bash
./ggo run --owner my-org --repo my-repo --issue 123
```

설정 파일의 `bot.dry_run`이 `true`이면 `--dry-run` 플래그가 없어도 댓글을 작성하지 않습니다. CLI의 `--dry-run`은 항상 dry-run을 켜는 방향으로만 동작합니다.

## 댓글 중복 방지

작성되는 댓글 본문 첫 줄에 marker가 포함됩니다.

```text
<!-- ggo:v1 -->
안녕하세요! ggo가 이 이슈를 확인했습니다.
```

기존 댓글 중 하나라도 같은 marker를 포함하면 새 댓글을 만들지 않고 종료합니다.
