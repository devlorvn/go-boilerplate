# go-boilerplate

Boilerplate microservice Go: gRPC services sau một HTTP gateway, đóng gói thành **một binary duy nhất**. Cấu trúc học theo một monorepo production thật (bootstrap kiểu servicer, config YAML nhiều lớp, RBAC theo từng method).

```
            HTTP :8000                     gRPC
client ───────────────► gateway ───────┬──────────► auth :50051 ──► Postgres (auth_db), Redis, Kafka
                        (gin, rate     │
                         limit, CORS)  └──────────► file :50052 ──► Postgres (file_db), S3
```

| Service | Vai trò | Port | Monitor (`/livez`, `/readyz`, `/metrics`) |
|---|---|---|---|
| `auth` | Đăng ký, đăng nhập, JWT (HS256), refresh token xoay vòng, profile | gRPC 50051 | :2112 |
| `file` | Upload/download qua S3, presigned URL, worker dọn file PENDING | gRPC 50052 | :2113 |
| `gateway` | REST → gRPC, rate limit bằng Redis, chặn token sớm | HTTP 8000 | :2114 |

## Bắt đầu nhanh

Yêu cầu: Go 1.26+, Docker, `openssl`.

**Cách 1: chạy service bằng `go run`** (hạ tầng trong Docker):

```bash
make dev-up        # Postgres, Redis, Kafka, S3 (SeaweedFS) qua docker compose
make secrets       # tạo configs/secrets/{auth,file}.yaml với JWT secret ngẫu nhiên
make migrate-up    # chạy migration cho auth và file
make run-all       # chạy auth, file, rồi gateway
```

**Cách 2: toàn bộ trong Docker** (giống môi trường deploy):

```bash
make up-app        # build image, migrate, chạy auth, file, gateway; chờ tới khi healthy
make down-app
```

Thử nhanh:

```bash
curl -s localhost:8000/api/v1/auth/register -d '{"email":"alice@example.com","password":"password123","name":"Alice"}'
TOKEN=$(curl -s localhost:8000/api/v1/auth/login -d '{"email":"alice@example.com","password":"password123"}' | jq -r .access_token)
curl -s localhost:8000/api/v1/auth/me -H "Authorization: Bearer $TOKEN"
```

S3 local là SeaweedFS ở `localhost:8333`, bucket `my-app-bucket` được tạo sẵn bởi service `s3-init`. Ở cách 2, presigned URL trỏ tới `http://s3:8333` (tên host trong mạng Docker); muốn gọi từ máy thật thì đổi host thành `localhost:8333`.

## Binary và lệnh

Mọi service nằm trong một binary `cmd/server` (cobra):

```bash
go run ./cmd/server gserver auth|file|gateway [--config-dir configs]
go run ./cmd/server migrate auth|file up
go run ./cmd/server migrate auth|file down --steps 1
go run ./cmd/server healthcheck auth|file|gateway [--path /readyz]   # exit 0 nếu probe trả 200
go run ./cmd/server --version
```

| Lệnh make | Tác dụng |
|---|---|
| `make run-<svc>` | Chạy một service (`run-auth`, `run-file`, `run-gateway`) |
| `make run-all` | Chạy cả ba, Ctrl+C dừng hết |
| `make migrate-up` | Migrate up cho mọi service có DB |
| `make migrate-down-<svc> STEPS=N` | Rollback N bước |
| `make build` | Build static binary ra `bin/server` (gắn version từ git) |
| `make test` / `make cover` | Test với `-race` / kèm coverage |
| `make lint` / `make fmt` | golangci-lint / format (gofmt + goimports) |
| `make gen-proto` | `buf generate` → `pkg/genproto` |
| `make lint-proto` / `make breaking` | `buf lint` / kiểm tra thay đổi proto phá client so với `origin/main` |
| `make tools` | Cài golangci-lint, buf, protoc-gen-go, protoc-gen-go-grpc đúng version CI dùng |
| `make docker-build` | Build image |
| `make up-app` / `down-app` | Toàn bộ stack trong Docker |
| `make dev-up` / `dev-down` | Chỉ hạ tầng local |

## Cấu hình

Mỗi service đọc theo thứ tự, lớp sau ghi đè lớp trước:

```
configs/common.yaml  →  configs/<svc>.yaml  →  configs/secrets/<svc>.yaml  →  biến môi trường
```

- `configs/secrets/*.yaml` bị gitignore, chỉ dùng ở local. Mẫu ở `configs/secrets/example.yaml`.
- Tên biến môi trường = đường dẫn key viết hoa, `.` đổi thành `_`. Ví dụ: `POSTGRES_PASSWORD`, `JWT_SECRET`, `STORAGE_ENDPOINT`, `COMMON_LOG_LEVEL`. Trên cloud, secret đi qua env var.
- `KAFKA_BROKERS=""` tắt Kafka (auth vẫn chạy, chỉ không publish event).
- `jwt.secret` **bắt buộc** và phải giống nhau giữa `auth` và `file`. Không có giá trị mặc định.

Khóa đáng chú ý:

| File | Key | Ý nghĩa |
|---|---|---|
| `common.yaml` | `common.env` | `local` bật gRPC reflection; image mặc định là `prod` |
| `common.yaml` | `common.drain_delay` | Thời gian `/readyz` báo DRAINING trước khi tắt server (xem [Health check](#health-check-và-graceful-shutdown)) |
| `auth.yaml` | `tokens.access_ttl`, `tokens.refresh_ttl` | Thời hạn access/refresh token |
| `file.yaml` | `storage.*` | Bucket, region, endpoint (để trống = S3 thật) |
| `file.yaml` | `cleanup.interval`, `cleanup.pending_timeout` | Chu kỳ worker và thời gian chờ trước khi đánh dấu FAILED |
| `gateway.yaml` | `upstreams.*`, `rate_limit.*` | Địa chỉ auth/file, số request mỗi cửa sổ |

## Cấu trúc thư mục

```
cmd/server/                 main.go (cobra), migrate.go
  <svc>/server.go           composition root: nối các tầng của service
configs/                    YAML theo lớp
migrations/<svc>/           SQL cho golang-migrate
proto/<pkg>/v1/             định nghĩa gRPC
pkg/genproto/               code sinh từ proto (không sửa tay)
internal/
  golibs/                   thư viện dùng chung
    bootstrap/              Register[T], vòng đời servicer, graceful shutdown
    configs/                Load[T], cấu hình nhiều lớp
    authz/                  Actor, Role, Policy (RBAC)
    errorx/                 lỗi có phân loại → mã gRPC
    interceptors/           recovery, logging, error, auth
    jwtutil/ database/ redis/ kafka/ healthcheck/ ...
  <svc>/
    configurations/         struct config của service
    domain/                 entity, quy tắc nghiệp vụ, lỗi domain (không import hạ tầng)
    application/
      ports.go              interface mà infrastructure phải cài đặt
      commands/             use case ghi
      queries/              use case đọc
      apptest/              fake in-memory cho unit test
    infrastructure/         postgres/, redis/, kafka/, s3/
    transport/              grpc/ (handler + Policy), worker/, http/ (gateway)
```

Hướng phụ thuộc: `transport` → `application` → `domain` ← `infrastructure`. Chỉ `cmd/server/<svc>` biết tất cả các tầng.

## Bootstrap

Mỗi service là một *servicer*, đăng ký trong `init()`:

```go
func init() { bootstrap.Register[configurations.Config]("auth", &server{}) }
```

Servicer cài các interface tùy chọn và bootstrap tự lo phần còn lại:

| Interface | Khi nào dùng |
|---|---|
| `GRPCServicer` | Đăng ký gRPC service, thêm interceptor riêng |
| `HTTPServicer` | Đăng ký route gin |
| `BackgroundServicer` | Job nền (ví dụ worker dọn file) |
| `Shutdowner` | Đóng kết nối khi tắt |

Luôn có sẵn:

- Monitor server với `/livez`, `/readyz` và `/metrics` (Prometheus).
- gRPC health check.
- Chuỗi interceptor: recovery → logging → error → interceptor của service.
- `Resources` khởi tạo lười: `DB()`, `Redis(ctx)`, `Kafka()`.

Listener mở đồng bộ nên cấu hình sai sẽ lỗi ngay lúc khởi động. Khi tắt, health chuyển sang NOT_SERVING trước rồi mới drain.

## Health check và graceful shutdown

| Endpoint | Ý nghĩa | Dùng cho |
|---|---|---|
| `/livez` | Luôn 200 khi process còn phục vụ được HTTP. Không kiểm tra dependency | Liveness probe (k8s), container health check (ECS, Docker) |
| `/readyz` | 200 khi mọi dependency đã mở (Postgres, Redis) đều ping được và service chưa bắt đầu tắt | Readiness probe (k8s), target group health check (ALB) |
| gRPC `grpc.health.v1.Health` | `SERVING` / `NOT_SERVING` | Load balancer hiểu gRPC |

Tách hai probe để DB chập chờn chỉ làm instance bị rút khỏi load balancer, không làm orchestrator restart hàng loạt.

Khi nhận SIGTERM:

1. `/readyz` trả 503 `DRAINING`, gRPC health chuyển `NOT_SERVING`.
2. Chờ `common.drain_delay` để load balancer kịp rút instance (đặt khoảng 10s khi đứng sau ALB, 0 ở local).
3. Dừng gRPC/HTTP server một cách graceful trong `common.shutdown_timeout`.
4. Gọi `Shutdown` của servicer, cuối cùng mới tắt monitor server.

Image distroless không có `curl`, nên health check của container chạy chính binary: `/app/server healthcheck <svc>`.

## Docker

`Dockerfile` multi-stage: build bằng `golang:1.26.4-alpine`, chạy trên `distroless/static:nonroot` (khoảng 66 MB, không shell, user 65532). Một image cho mọi service, chọn bằng tham số:

```bash
docker run IMAGE gserver auth
docker run IMAGE migrate auth up
```

Image chứa `configs/*.yaml` và `migrations/`, **không** chứa `configs/secrets/`. Secret và địa chỉ hạ tầng truyền qua env var, ví dụ `JWT_SECRET`, `POSTGRES_HOST`, `POSTGRES_PASSWORD`, `REDIS_HOST`, `KAFKA_BROKERS`.

## Phân quyền

Mỗi service khai báo bảng quyền theo method, mặc định **từ chối**:

```go
var Policy = authz.Policy{
    userv1.UserService_Login_FullMethodName:      authz.Public,
    userv1.UserService_GetProfile_FullMethodName: authz.Authenticated,
    // authz.Roles(authz.RoleAdmin) cho method chỉ admin
}
```

- `Policy.Validate(ServiceDesc)` chạy lúc khởi động: thêm RPC mà quên khai báo quyền thì service không lên.
- Gateway chuyển tiếp header `Authorization`; mỗi service tự xác thực lại token (zero-trust).
- Quyền theo dữ liệu nằm trong domain, ví dụ `File.AccessibleBy(actor)`: user chỉ thấy file của mình, admin thấy tất cả. File của người khác trả 404.

## Xử lý lỗi

Code nghiệp vụ trả lỗi `errorx` có phân loại; `ErrorInterceptor` đổi sang mã gRPC, gateway đổi tiếp sang HTTP:

| errorx | gRPC | HTTP |
|---|---|---|
| `InvalidArgument` | InvalidArgument | 400 |
| `Unauthenticated` | Unauthenticated | 401 |
| `PermissionDenied` | PermissionDenied | 403 |
| `NotFound` | NotFound | 404 |
| `AlreadyExists` | AlreadyExists | 409 |
| `FailedPrecondition` | FailedPrecondition | 412 |
| `Internal` / lỗi chưa phân loại | Internal | 500, chỉ trả `"internal error"` |

Chi tiết lỗi nội bộ chỉ ghi vào log server. Body lỗi luôn có dạng `{"code": "...", "message": "..."}`.

## HTTP API (gateway)

| Method | Path | Auth |
|---|---|---|
| POST | `/api/v1/auth/register` | — |
| POST | `/api/v1/auth/login` | — |
| POST | `/api/v1/auth/refresh-token` | — (refresh token chỉ dùng được 1 lần) |
| GET | `/api/v1/auth/me` | Bearer |
| GET | `/api/v1/files?limit=&offset=&status=` | Bearer |
| DELETE | `/api/v1/files?file_key=` | Bearer |
| GET | `/api/v1/files/presigned-url?file_key=&expires_in_seconds=` | Bearer |
| POST | `/api/v1/files/upload-presigned-url` | Bearer |

## Thêm một service mới

1. Viết `proto/<name>/v1/<name>.proto`, chạy `make lint-proto gen-proto`.
2. Tạo `internal/<name>/` theo layout ở trên: `domain`, `application` (ports, commands, queries), `infrastructure`, `transport/grpc` kèm `Policy`.
3. Tạo `configs/<name>.yaml` và (nếu cần DB) `migrations/<name>/`, thêm tên vào `DB_SERVICES` trong `Makefile`.
4. Tạo `cmd/server/<name>/server.go` gọi `bootstrap.Register`, rồi blank-import nó trong `cmd/server/main.go`.
5. Thêm service vào `docker-compose.yml` (profile `app`) nếu muốn chạy cùng stack.

## Test và chất lượng code

```bash
make tools         # một lần
make test lint lint-proto
```

- Use case được test bằng fake viết tay trong `application/apptest`, không cần Docker. Không dùng mockery: fake in-memory kiểm tra được hành vi (ví dụ refresh token chỉ dùng một lần) thay vì chỉ kiểm tra lời gọi, và không phải sinh lại khi interface đổi.
- `.golangci.yml` bật thêm gosec, errorlint, noctx, nilerr, gocritic, revive... và goimports với nhóm import `go-boilerplate`.

## CI

`.github/workflows/ci.yml` chạy trên mỗi push vào `main` và mỗi pull request:

| Job | Kiểm tra |
|---|---|
| `lint` | golangci-lint |
| `test` | `go mod tidy -diff`, `go test -race` với coverage |
| `proto` | `buf lint`, `buf breaking` so với nhánh đích (chỉ PR), code trong `pkg/genproto` khớp với `buf generate` |
| `docker` | Build image (không push), cache qua GitHub Actions |

## Hạn chế đã biết

- Upload qua presigned URL không có bước xác nhận, nên file ở trạng thái PENDING cho tới khi worker đánh dấu FAILED (sau `cleanup.pending_timeout`). Hướng sửa: RPC `ConfirmUpload` hoặc S3 Event Notification → SQS.
- CORS đang để `*` cùng `credentials: true`; cần đổi sang danh sách origin cụ thể khi có frontend.
- S3 dùng static credentials; trên AWS nên dùng credential chain mặc định (IAM role).
