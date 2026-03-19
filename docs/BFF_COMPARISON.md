# OLake TUI vs BFF — 기능 비교 및 테스트 체크리스트

> 생성일: 2026-03-19
> olake-tui 커밋: f7e006b / BFF 참조: `_olake-ui-ref/server/`

## 아키텍처 차이

| 항목 | BFF (olake-ui server) | olake-tui |
|------|----------------------|-----------|
| DB 접근 | Beego ORM | database/sql (lib/pq) |
| Temporal | temporal wrapper 패키지 | go.temporal.io/sdk 직접 사용 |
| 암호화 | AES-256-GCM (utils/encryption.go) | ✅ 동일 구현 |
| 인증 | bcrypt + session cookie | ✅ bcrypt (DB 직접 조회) |
| 테이블 네이밍 | `olake-{runMode}-{entity}` | ✅ 동일 |
| Soft Delete | Beego ORM (deleted_at) | ✅ 수동 구현 |

---

## 기능별 비교

### ✅ 완전 구현

| BFF 기능 | TUI 메서드 | 비고 |
|----------|-----------|------|
| Login | `Login()` | bcrypt 검증 동일 |
| ListSources | `ListSources()` | job_count 포함 |
| GetSource | `GetSource()` | - |
| CreateSource | `CreateSource()` | 암호화 포함 |
| UpdateSource | `UpdateSource()` | - |
| DeleteSource | `DeleteSource()` | soft-delete + job count 체크 |
| ListDestinations | `ListDestinations()` | job_count 포함 |
| GetDestination | `GetDestination()` | - |
| CreateDestination | `CreateDestination()` | - |
| UpdateDestination | `UpdateDestination()` | - |
| DeleteDestination | `DeleteDestination()` | soft-delete + job count 체크 |
| ListJobs | `ListJobs()` | source/dest JOIN 포함 |
| GetJob | `GetJob()` | - |
| CreateJob | `CreateJob()` | ✅ Temporal 스케줄 생성 포함 |
| DeleteJob | `DeleteJob()` | Temporal 스케줄 삭제 포함 |
| TriggerSync | `TriggerSync()` | schedule.Trigger() |
| CancelJob | `CancelJob()` | workflow cancel |
| ActivateJob | `ActivateJob()` | DB + schedule pause/unpause |
| UpdateJobMeta | `UpdateJobMeta()` | name + frequency |
| GetJobTasks | `ListJobTasks()` | Temporal workflow history |
| GetTaskLogs | `GetTaskLogs()` | 디스크 로그 파일 읽기 |
| TestSourceConnection | `TestSource()` | Temporal check workflow |
| TestDestConnection | `TestDestination()` | Temporal check workflow |
| DiscoverStreams | `DiscoverStreams()` | Temporal discover workflow |
| ClearDestination | `ClearDestination()` | 2단계 confirm + schedule 교체 |
| GetSettings | `GetSettings()` | project-settings 테이블 |
| UpdateSettings | `UpdateSettings()` | UPSERT |
| ValidateSchema | `ValidateSchema()` | 테이블 존재 여부 체크 |

### ⚠️ 부분 구현 (차이 있음)

| BFF 기능 | TUI 상태 | 차이점 |
|----------|---------|--------|
| CreateJob | ✅ 구현 | BFF: job 이름 유일성 검사 (`IsJobNameUniqueInProject`) → TUI: **없음** |
| CreateJob | ✅ 구현 | BFF: source/dest를 upsert (이미 있으면 재사용) → TUI: sourceID/destID 직접 전달 |
| UpdateJob | ⚠️ 부분 | BFF: streams, source, dest, advanced_settings 전부 업데이트 + clear-dest 진행 중 차단 → TUI: name/frequency만 (`UpdateJobMeta`) |
| DeleteSource/Dest | ✅ 구현 | BFF: 연관 job을 cascade cancel → TUI: job 존재 시 삭제 거부 (더 보수적) |
| ClearDestination | ✅ 구현 | BFF: `RecoverFromClearDestination` 복구 메커니즘 → TUI: **없음** |
| ClearDestination | ✅ 구현 | BFF: `GetClearDestinationStatus` 상태 조회 → TUI: **없음** |
| ListJobs | ✅ 구현 | BFF: lastRunState를 Temporal에서 실시간 조회 → TUI: DB에 캐시된 값만 표시 |

### ❌ 미구현

| BFF 기능 | 설명 | 중요도 |
|----------|------|--------|
| `GetSourceVersions` | 소스 커넥터 버전 목록 (Docker 이미지 태그) | 🟡 |
| `GetSourceSpec` | 커넥터 스펙 JSON (폼 필드 자동 생성) | 🟡 |
| `GetDestinationVersions` | 목적지 커넥터 버전 목록 | 🟡 |
| `GetDestinationSpec` | 목적지 커넥터 스펙 JSON | 🟡 |
| `GetAllReleasesResponse` | GitHub 릴리즈 목록 (업데이트 알림) | 🟢 |
| `GetClearDestinationStatus` | clear-dest 진행 상태 조회 | 🟡 |
| `RecoverFromClearDestination` | clear-dest 실패 시 스케줄 복원 | 🟡 |
| `DownloadTaskLogs` | 로그 파일 다운로드 | 🟢 |
| `UpdateJob` (full) | streams/source/dest/advancedSettings 업데이트 | 🔴 |
| Job name uniqueness check | 동일 프로젝트 내 중복 이름 체크 | 🟡 |
| Telemetry tracking | job/source 생성 시 텔레메트리 | 🟢 |
| `CheckClearDestCompatibility` | 소스 버전별 clear-dest 호환성 체크 | 🟡 |

---

## E2E 테스트 체크리스트

실제 OLake 환경에서 검증해야 할 항목들:

### 인증
- [ ] 유효한 credentials로 로그인 성공
- [ ] 잘못된 credentials로 로그인 실패 + 에러 메시지 확인
- [ ] 빈 username/password 클라이언트 검증 동작

### Sources
- [ ] 소스 목록 조회 (job count 정확성)
- [ ] PostgreSQL 소스 생성
- [ ] 소스 이름/config 수정
- [ ] 소스 삭제 (연관 job 없는 경우)
- [ ] 소스 삭제 거부 (연관 job 있는 경우)
- [ ] 소스 연결 테스트 (Temporal check workflow)
- [ ] 암호화된 config가 BFF에서도 복호화 가능한지 확인

### Destinations
- [ ] 목적지 목록 조회
- [ ] Iceberg/Parquet 목적지 생성
- [ ] 목적지 수정
- [ ] 목적지 삭제
- [ ] 목적지 연결 테스트

### Jobs
- [ ] Job 목록 조회 (source/dest name 매핑)
- [ ] Job 생성 → Temporal 스케줄 생성 확인 (`tctl schedule list`)
- [ ] Job 스케줄 트리거 (수동 sync)
- [ ] Job 취소 (running workflow cancel)
- [ ] Job 일시정지/재개 (schedule pause/unpause)
- [ ] Job 삭제 → Temporal 스케줄 삭제 확인
- [ ] Job 이름/frequency 수정
- [ ] Job 태스크 히스토리 조회
- [ ] 태스크 로그 조회 (pagination: older/newer)

### Streams
- [ ] Source에서 스트림 discover
- [ ] 스트림 선택/해제
- [ ] per-stream sync mode 설정
- [ ] cursor field 설정

### Settings
- [ ] Webhook URL 조회
- [ ] Webhook URL 저장

### Clear Destination
- [ ] Clear destination 실행 (2단계 confirm)
- [ ] 실행 후 Temporal 스케줄이 sync로 복원되는지 확인

### 교차 호환성
- [ ] TUI에서 생성한 source를 BFF UI에서 조회/수정 가능
- [ ] BFF에서 생성한 job을 TUI에서 조회/sync/cancel 가능
- [ ] 암호화 키 동일할 때 양방향 config 복호화
- [ ] TUI에서 soft-delete한 항목이 BFF에서도 보이지 않음

### 엣지 케이스
- [ ] Temporal 미연결 상태에서 TUI 실행 → DB 기능만 동작
- [ ] 잘못된 DB URL → 명확한 에러 메시지
- [ ] 잘못된 runMode → 거부됨
- [ ] 좁은 터미널 (80x24)에서 레이아웃 깨짐 없음
- [ ] 매우 긴 source/job 이름 → 잘림 처리

---

## 우선순위 구현 로드맵

### Phase 1 (필수 — BFF 호환성)
1. **`UpdateJob` 풀 구현** — streams, source, dest 전부 업데이트 가능하도록
2. **Job name uniqueness** — `CreateJob` 시 중복 이름 체크
3. **ListJobs lastRunState** — Temporal에서 실시간 상태 조회 (현재 DB 캐시만)

### Phase 2 (권장)
4. `GetClearDestinationStatus` — clear-dest 진행 상태 표시
5. `RecoverFromClearDestination` — 실패 시 자동 복원
6. `GetSourceSpec` / `GetDestinationSpec` — 커넥터 스펙 기반 동적 폼

### Phase 3 (있으면 좋음)
7. `GetSourceVersions` / `GetDestinationVersions` — 버전 선택기
8. `GetAllReleasesResponse` — 실제 릴리즈 데이터로 Updates 모달 채우기
9. 로그 파일 다운로드
10. Telemetry (선택적)
