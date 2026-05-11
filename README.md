# ggoboong

`ggo`는 GitHub 이것저것 처리해주는 꼬붕 CLI입니다.

지금은 아주 단순합니다. GitHub issue를 읽고, 아직 `ggo`가 답변한 적이 없으면 정해진 댓글을 하나 남깁니다. LLM, DB, 서버, Webhook, OAuth는 아직 없습니다.

## 지금 하는 일

- GitHub App installation token으로 인증
- issue 조회
- issue comments 조회
- 이미 남긴 `ggo` 댓글이 있으면 중단
- 없으면 고정 답변 댓글 작성
- `--dry-run`이면 실제 작성 없이 댓글 본문만 출력

## 설치

이 저장소에서 설치 스크립트를 실행하면 `ggo`를 빌드해서 `~/.local/bin/ggo`로 복사합니다.

```bash
./install.sh
```

원하는 설치 위치가 있으면 `GGO_INSTALL_DIR`를 지정합니다.

```bash
GGO_INSTALL_DIR=/usr/local/bin ./install.sh
```

`~/.local/bin`이 `PATH`에 없다면 shell profile에 추가합니다.

```bash
export PATH="$HOME/.local/bin:$PATH"
```

개발 중에는 그냥 빌드해도 됩니다.

```bash
go build -o ggo ./cmd/ggo
```

## GitHub App 준비

`ggo`는 GitHub App ID `3675420`을 소스에 고정해서 사용합니다. 이 값은 secret이 아니므로 설정 파일이나 `.env`에 넣을 필요가 없습니다.

GitHub App에는 repository 권한이 필요합니다.

- `Issues`: `Read and write`
- `Metadata`: `Read-only`

App을 repository에 설치한 뒤 `installation_id`를 확인하고, private key PEM 파일을 내려받습니다.

## 설정 설치

PEM을 바이너리 안에 넣어서 빌드하지 마세요. private key는 로컬 파일로 보관하고, 현재 사용자만 읽을 수 있게 두는 편이 안전합니다.

`ggo login`은 `~/.ggo/ggo.yaml`을 만들고 PEM을 `~/.ggo/github-app.private-key.pem`로 복사합니다.

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

## 설정 파일

`ggo.yaml` 예시:

```yaml
github:
  installation_id: 987654
  private_key_path: ./github-app.private-key.pem

bot:
  dry_run: false
```

설정 파일은 아래 순서로 찾습니다.

1. `--config`로 넘긴 경로
2. `GGO_CONFIG` 환경변수 경로
3. 현재 디렉터리의 `./ggo.yaml`
4. `~/.ggo/ggo.yaml`

## 환경변수

`ggo`는 실행할 때 `.env`가 있으면 자동으로 읽습니다.

읽는 순서:

1. 현재 디렉터리의 `./.env`
2. `~/.ggo/.env`

지원하는 환경변수:

```bash
GGO_INSTALLATION_ID=987654
GGO_PRIVATE_KEY_PATH=./github-app.private-key.pem
GGO_DRY_RUN=false
GGO_CONFIG=./ggo.yaml
```

현재 프로젝트에서 쓰던 이름도 호환됩니다.

```bash
GGOBOONG_INSTALLATION_ID=987654
GGOBOONG_PRIVATE_KEY_PATH=./github-app.private-key.pem
```

환경변수는 `ggo.yaml` 값을 덮어씁니다.

## 실행

dry-run:

```bash
ggo run --owner my-org --repo my-repo --issue 123 --dry-run
```

실제 댓글 작성:

```bash
ggo run --owner my-org --repo my-repo --issue 123
```

특정 설정 파일을 쓰려면:

```bash
ggo run --owner my-org --repo my-repo --issue 123 --config ./ggo.yaml
```

## 댓글 중복 방지

`ggo`는 자기가 남긴 댓글인지 알아보기 위해 댓글 첫 줄에 숨김 HTML 주석을 넣습니다. 이건 GitHub 공식 기능이나 별도 설정 이름이 아니라, CLI 내부에서 중복 댓글을 피하려고 쓰는 작은 표식입니다.

```text
<!-- ggo:v1 -->
안녕하세요! ggo가 이 이슈를 확인했습니다.
```

기존 댓글 중 하나라도 같은 숨김 주석을 포함하면 새 댓글을 만들지 않고 종료합니다.
