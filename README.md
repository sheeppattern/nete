# zk — AI 에이전트용 제텔카스텐 메모리 CLI

AI 에이전트가 지식을 **원자적 노트**로 저장하고, **관계 타입과 가중치가 있는 양방향 연결**로 구조화하며, **프로젝트 단위로 격리**하여 관리하는 CLI 도구. Concrete/Abstract 노트 레이어를 통해 사실 기록과 인사이트를 체계적으로 분리한다.

> **zk** is a CLI tool that lets AI agents store knowledge as **atomic notes**, structure them with **typed and weighted bidirectional links**, and manage them in **project-scoped isolation**. It separates factual records (concrete) from insights (abstract) through a two-layer note system.

## 왜 필요한가

기존 AI 에이전트의 메모리는 단편적이다. "A를 안다"는 기억할 수 있지만, "A는 B를 뒷받침하고, B는 C와 모순된다"는 표현할 수 없다. 또한 단순 기록만 쌓이고, 기록 사이의 패턴이나 긴장을 도출하는 구조가 없다.

zk는 제텔카스텐 원칙을 AI 에이전트에 적용하여 **기억 사이의 관계를 구조화**하고, **사실(concrete)에서 인사이트(abstract)를 체계적으로 도출**한다.

> Conventional AI agent memory is flat — it can remember "A exists" but cannot express "A supports B, and B contradicts C." Records accumulate without structure for deriving patterns or tensions between them.
>
> zk applies Zettelkasten principles to AI agents, enabling **structured reasoning across memories** and **systematic derivation of insights (abstract) from facts (concrete)**.

```
Concrete Layer:  "MAU 500" ──supports──▶ "Redis 캐싱 적용"
                      │                        │
                  abstracts               contradicts
                      ▼                        ▼
Abstract Layer:  "성장 vs 리텐션 — 어느 쪽에 투자할 것인가?"
```

## 설치

Go가 설치된 환경에서:

> With Go installed:

```bash
go install github.com/sheeppattern/zk@latest
```

또는 바이너리 직접 빌드:

> Or build from source:

```bash
git clone https://github.com/sheeppattern/zk.git
cd zk
go build -o zk .
```

또는 [GitHub Releases](https://github.com/sheeppattern/zk/releases)에서 플랫폼별 바이너리를 다운로드할 수 있다.

> Pre-built binaries for all platforms are available at [GitHub Releases](https://github.com/sheeppattern/zk/releases).

## 빠른 시작

```bash
# 저장소 초기화 (6개 에이전트 도구용 skill 파일 자동 생성)
zk init

# 프로젝트 생성
zk project create "auth-migration" --description "인증 시스템 마이그레이션"

# Concrete 노트 생성 (사실 기록)
zk note create --title "JWT 토큰 구조" \
  --content "Access Token과 Refresh Token 분리 저장. Redis 권장." \
  --tags "jwt,auth,redis" --layer concrete --project P-XXXXXX

zk note create --title "세션 기반 인증의 한계" \
  --content "서버 확장 시 세션 공유 문제." \
  --tags "session,auth" --layer concrete --project P-XXXXXX

# 관계 연결 (중복 자동 방지)
zk link add N-AAAAAA N-BBBBBB --type contradicts --weight 0.8 --project P-XXXXXX

# Abstract 노트 생성 (인사이트)
zk note create --title "세션 vs JWT — 확장성과 복잡성의 트레이드오프" \
  --content "..." --layer abstract --project P-XXXXXX

# 인사이트 자동 제안
zk reflect --project P-XXXXXX
```

> ```bash
> # Initialize store (auto-generates skill files for 6 AI agent tools)
> zk init
>
> # Create project, concrete notes, links, abstract notes
> # Then use zk reflect to get insight suggestions
> ```

## 핵심 개념: Concrete/Abstract 레이어

모든 노트는 두 레이어 중 하나에 속한다:

> Every note belongs to one of two layers:

| 레이어 / Layer | 역할 / Role | 예시 / Example |
|------|------|---------|
| **concrete** (기본) | 사실, 관찰, 데이터 기록 | "MAU 500, 리텐션 23%" |
| **abstract** | 패턴, 긴장, 질문, 인사이트 | "성장 투자 vs 리텐션 개선 — 뭐가 먼저?" |

레이어 간 연결을 위한 전용 관계 타입:

> Dedicated relation types for cross-layer links:

| 관계 타입 / Type | 방향 / Direction | 의미 / Meaning |
|------|------|------|
| `abstracts` | concrete → abstract | "이 사실에서 이 인사이트가 도출됨" |
| `grounds` | abstract → concrete | "이 인사이트의 근거가 되는 사실" |

### `zk reflect` — 인사이트 자동 제안

concrete 노트를 분석하여 누락된 abstract 노트를 자동으로 제안한다:

> Analyzes concrete notes and suggests missing abstract notes:

```bash
zk reflect --project P-XXXXXX              # 인사이트 후보 출력
zk reflect --project P-XXXXXX --apply      # 제안을 자동으로 노트 생성
zk reflect --project P-XXXXXX --format md  # 마크다운 리포트
```

감지하는 패턴:

> Detected patterns:

- **긴장(tensions)**: contradicts 관계가 있지만 이를 종합하는 abstract 노트가 없는 쌍
- **추상화 필요한 허브(hubs)**: 4개 이상 연결된 concrete 노트에 상위 추상화가 없는 경우
- **고립 노트(orphans)**: 어디에도 연결되지 않은 노트
- **추상화 비율(ratio)**: concrete 대비 abstract 비율이 낮으면 인사이트 부족 경고

## 명령어 레퍼런스

### 초기화 및 설정

```bash
zk init                              # 저장소 초기화 + 에이전트 skill 파일 생성
zk init --path /custom/path          # 커스텀 경로
zk config show                       # 현재 설정 조회
zk config set <key> <value>          # 설정 변경
```

> Available config keys: `store_path`, `default_project`, `default_format`
>
> When `default_project` is set, it auto-applies when `--project` is not specified.

### 프로젝트 관리

```bash
zk project create <name> --description "설명"
zk project list
zk project get <id>       # 노트 수, 링크 수, 최근 활동 통계 포함
zk project delete <id>
```

### 노트 CRUD

```bash
zk note create --title "제목" --content "내용" --tags "t1,t2" --layer concrete --project <id>
zk note create --title "인사이트" --content "..." --layer abstract --project <id>
zk note create --title "제목" --template research --project <id>   # 템플릿 사용
zk note get <noteID> --project <id>
zk note list --project <id>
zk note list --layer abstract --project <id>       # 레이어 필터
zk note update <noteID> --title "새 제목" --project <id>
zk note delete <noteID> --project <id>             # 백링크 있으면 거부
zk note delete <noteID> --force --project <id>     # 강제 삭제 (trash로 이동)
zk note move <noteID> <targetProject> --project <sourceProject>
```

> Deleted notes are moved to `trash/`, not permanently removed.

### 링크 (관계 타입 + 가중치)

```bash
zk link add <src> <tgt> --type supports --weight 0.8 --project <id>
zk link add <src> <tgt> --type extends --project P-1 --target-project P-2   # 크로스 프로젝트
zk link remove <src> <tgt> --project <id>
zk link list <noteID> --project <id>
zk link list <noteID> --type supports              # 관계 타입 필터
zk link list <noteID> --sort-weight                # 가중치 순 정렬
zk link list <noteID> --depth 3 --project <id>     # BFS 깊이 탐색
```

> Duplicate links are auto-prevented. Cross-project backlinks are included in results.

#### 관계 타입

| 타입 | 의미 | 예시 |
|------|------|------|
| `related` | 일반적 관련 (기본값) | 같은 주제의 다른 관점 |
| `supports` | 뒷받침/근거 | 증거가 주장을 지지 |
| `contradicts` | 반박/모순 | 상충하는 의견 |
| `extends` | 확장/발전 | 아이디어를 더 발전시킴 |
| `causes` | 원인/결과 | 인과 관계 |
| `example-of` | 사례/예시 | 개념의 구체적 사례 |
| `abstracts` | 추상화 | concrete에서 abstract 도출 |
| `grounds` | 근거 | abstract의 근거가 되는 concrete |

> Types: related (default), supports, contradicts, extends, causes, example-of, abstracts, grounds

#### 가중치

| 범위 | 의미 |
|------|------|
| 0.8~1.0 | 매우 강한 관계 (핵심 연결) |
| 0.5~0.7 | 보통 관계 (참고 수준) |
| 0.1~0.4 | 약한 관계 (간접 연결) |

> 0.8–1.0: core connection / 0.5–0.7: reference level / 0.1–0.4: indirect

### 검색

```bash
zk search <query> --project <id>
zk search "Redis" --tags "cache" --relation supports --min-weight 0.5
zk search "인증" --layer abstract --sort relevance
zk search "data" --created-after 2026-01-01 --created-before 2026-12-31
```

| 옵션 | 설명 |
|------|------|
| `--tags` | 태그 필터 (AND 로직) |
| `--relation` | 특정 관계 타입을 가진 노트만 |
| `--min-weight` | 최소 가중치 이상인 링크를 가진 노트만 |
| `--status` | 상태 필터 (active/archived) |
| `--layer` | 레이어 필터 (concrete/abstract) |
| `--sort` | 정렬 기준 (relevance/created/updated) |
| `--created-after` | 이 날짜 이후 생성 (YYYY-MM-DD) |
| `--created-before` | 이 날짜 이전 생성 (YYYY-MM-DD) |

### 태그 관리

```bash
zk tag add <noteID> <tag1> [tag2...] --project <id>
zk tag remove <noteID> <tag1> [tag2...]
zk tag replace <oldTag> <newTag> --project <id>
zk tag list --project <id>
zk tag batch-add <tag> <noteID1> [noteID2...]
```

### 진단

```bash
zk diagnose --project <id>
```

끊어진 링크, 파싱 실패 파일, 고아 노트, 잘못된 관계 타입, 범위 초과 가중치를 검사하고 오류/경고를 구분하여 리포트한다.

> Checks: broken links, corrupted files, orphan notes, invalid relation types, out-of-range weights.

### 내보내기 / 가져오기

```bash
zk export --project <id> --format yaml --output backup.yaml
zk export --project <id> --notes N-AAA,N-BBB
zk import --file backup.yaml --project <id> --conflict skip
```

> Conflict resolution: `skip`, `overwrite`, `new-id`

### 스키마 자가 조회

```bash
zk schema              # 전체 리소스 목록
zk schema note         # 노트 필드 상세
zk schema link         # 링크 필드 상세
zk schema relation-types
```

> AI agents can discover data structures at runtime via `zk schema`.

### 에이전트 스킬 생성

`zk init` 실행 시 6개 AI 코딩 도구용 instruction 파일이 자동 생성된다:

> On `zk init`, instruction files for 6 AI coding tools are auto-generated:

| 도구 / Tool | 파일 / File | 레벨 / Level |
|------|------|------|
| Claude Code | `~/.claude/skills/zk/SKILL.md` | Global |
| Gemini CLI | `~/.gemini/instructions/zk.md` | Global |
| OpenAI Codex | `~/.codex/instructions/zk.md` | Global |
| Cursor | `.cursor/rules/zk.mdc` | Project |
| GitHub Copilot | `.github/copilot-instructions.md` | Project |
| Windsurf | `.windsurf/rules/zk.md` | Project |

```bash
zk skill generate                                    # global 파일만
zk skill generate --project-dir .                    # global + project 파일
zk skill generate --agents claude,cursor --project-dir .  # 선택적
```

### 버전 관리

```bash
zk version                 # 버전 확인
zk update                  # 최신 버전으로 업데이트
zk update --check          # 업데이트 확인만
zk uninstall               # 바이너리 삭제
zk uninstall --purge       # 바이너리 + 저장소 + skill 파일 전부 삭제
```

## 글로벌 옵션

| 옵션 | 설명 | 기본값 |
|------|------|--------|
| `--format` | 출력 형식 (json/yaml/md) | json |
| `--project` | 프로젝트 범위 지정 | (global) |
| `--verbose` | 디버그 출력 | false |
| `--quiet` | stderr 상태 메시지 억제 | false |

## 파이프라인 안전 출력

- **stdout**: 순수 데이터만 (JSON/YAML/Markdown)
- **stderr**: 상태 메시지, 에러, 디버그 정보
- `--quiet`로 stderr를 억제할 수 있다

> stdout = pure data, stderr = status/errors. Use `--quiet` to suppress stderr.

```bash
NOTES=$(zk search "Redis" --project P-XXX --format json --quiet 2>/dev/null)
NOTE_ID=$(zk note create --title "발견" --content "..." --quiet 2>/dev/null | jq -r '.id')
```

## 저장소 구조

```
~/.zk-memory/
├── config.yaml
├── projects/
│   └── {project-id}/
│       ├── project.yaml
│       └── notes/
│           └── {note-id}.md      # YAML frontmatter + Markdown body
├── global/
│   └── notes/                     # 프로젝트에 속하지 않는 범용 노트
├── trash/                          # 삭제된 노트 보관
└── templates/                      # 노트 템플릿 (.yaml)
```

> Notes use YAML frontmatter + Markdown body format. Human-readable and hand-editable.

### 노트 파일 형식

```markdown
---
id: N-72F576
title: JWT 토큰 구조
tags: [jwt, auth, redis]
layer: concrete
links:
  - target_id: N-CE12CD
    relation_type: contradicts
    weight: 0.8
metadata:
  created_at: 2026-03-20T13:39:56+09:00
  updated_at: 2026-03-20T13:40:13+09:00
  source: ""
  status: active
project_id: P-20FFD1
---
Access Token과 Refresh Token의 분리 저장 방식 검토. Redis에 Refresh Token 저장 권장.
```

## 기술 스택

- **언어**: Go 1.26
- **CLI 프레임워크**: [cobra](https://github.com/spf13/cobra)
- **설정 관리**: [viper](https://github.com/spf13/viper)
- **YAML**: gopkg.in/yaml.v3
- **ID 생성**: google/uuid

> Single binary, zero runtime dependencies.

## 라이선스

MIT

---

## 부록: 이 프로젝트는 Manyfast로 만들어졌습니다

이 프로젝트의 기획 문서(PRD, 요구사항, 기능, 스펙)는 [Manyfast](https://manyfast.io)로 작성 및 관리되었습니다.

> Planning documents (PRD, requirements, features, specs) were created and managed with [Manyfast](https://manyfast.io).

### 기획에서 개발까지

1. **Manyfast에서 PRD 작성**: 제품 목표, 사용자 문제, 솔루션, 차별점, KPI, 리스크 정의
2. **요구사항 12개 정의**: PRD 기반 요구사항 → 기능 → 스펙 계층 구조
3. **AI 에이전트와 협업 개발**: Manyfast CLI로 기획 문서를 읽고, 진행도를 실시간 추적
4. **MVP 완료**: 기획 문서 작성부터 전체 구현까지 약 **30~40분** 소요

> 1. PRD in Manyfast: goals, problems, solutions, KPIs, risks
> 2. 12 requirements defined: hierarchical requirement → feature → spec structure
> 3. AI-assisted development with real-time progress tracking via Manyfast CLI
> 4. MVP complete: ~30-40 minutes from planning to full implementation

### 최종 산출물

| 항목 | 수량 |
|------|------|
| 요구사항 (Requirements) | 12 (전체 done) |
| 기능 (Features) | 24 |
| 스펙 (Specs) | 58 |
| CLI 명령어 | 35+ |
| Go 소스 파일 | 17 |
| 테스트 | 46 |

> Manyfast Project ID: `5fc2a8ca-c59b-4fb3-a0c7-c5744137028b`
