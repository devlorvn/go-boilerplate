# go-boilerplate

Boilerplate microservice Go: gRPC services sau một HTTP gateway, đóng gói thành **một binary duy nhất**. Cấu trúc học theo một monorepo production thật (bootstrap kiểu servicer, config YAML nhiều lớp, RBAC theo từng method).

```
            HTTP :8000                     gRPC
client ───────────────► gateway ───────┬──────────► auth :50051 ──► Postgres (auth_db), Redis, Kafka
                        (gin, rate     │
                         limit, CORS)  └──────────► file :50052 ──► Postgres (file_db), S3
```

| Service | Vai trò | Port | Metrics/health |
|---|---|---|---|
| `auth` | Đăng ký, đăng nhập, JWT (HS256), refresh token xoay vòng, profile | gRPC 50051 | :2112 |
| `file` | Upload/download qua S3, presigned URL, worker dọn file PENDING | gRPC 50052 | :2113 |
| `gateway` | REST → gRPC, rate limit bằng Redis, chặn token sớm | HTTP 8000 | :2114 |

## Bắt đầu nhanh

Yêu cầu: Go 1.26+, Docker, `openssl`.

```bash
make dev-up        # Postgres, Redis, Kafka qua docker compose
make secrets       # tạo configs/secrets/{auth,file}.yaml với JWT secret ngẫu nhiên
make migrate-up    # chạy migration cho auth và file
make run-all       # chạy auth, file, rồi gateway
```

Thử nhanh:

```bash
curl -s localhost:8000/api/v1/auth/register -d '{"email":"alice@example.com","password":"password123","name":"Alice"}'
TOKEN=$(curl -s localhost:8000/api/v1/auth/login -d '{"email":"alice@example.com","password":"password123"}' | jq -r .access_token)
curl -s localhost:8000/api/v1/auth/me -H "Authorization: Bearer $TOKEN"
```

Service `file` cần S3. Chạy local thì trỏ `storage.endpoint` tới MinIO hoặc LocalStack (xem [Cấu hình](#cấu-hình)).

## Binary và lệnh

Mọi service nằm trong một binary `cmd/server` (cobra):

```bash
go run ./cmd/server gserver auth|file|gateway [--config-dir configs]
go run ./cmd/server migrate auth|file up
go run ./cmd/server migrate auth|file down --steps 1
```

| Lệnh make | Tác dụng |
|---|---|
| `make run-<svc>` | Chạy một service (`run-auth`, `run-file`, `run-gateway`) |
| `make run-all` | Chạy cả ba, Ctrl+C dừng hết |
| `make migrate-up` | Migrate up cho mọi service có DB |
| `make migrate-down-<svc> STEPS=N` | Rollback N bước |
| `make build` | Build static binary ra `bin/server` |
| `make test` | `go test ./...` |
| `make gen-proto` | Sinh code từ `proto/` (cần `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`) |
| `make dev-up` / `dev-down` | Bật/tắt hạ tầng local |

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
| `common.yaml` | `common.env` | `local` bật gRPC reflection |
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

- Monitor server với `/healthz` và `/metrics` (Prometheus).
- gRPC health check.
- Chuỗi interceptor: recovery → logging → error → interceptor của service.
- `Resources` khởi tạo lười: `DB()`, `Redis(ctx)`, `Kafka()`.

Listener mở đồng bộ nên cấu hình sai sẽ lỗi ngay lúc khởi động. Khi tắt, health chuyển sang NOT_SERVING trước rồi mới drain.

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

1. Viết `proto/<name>/v1/<name>.proto`, chạy `make gen-proto`.
2. Tạo `internal/<name>/` theo layout ở trên: `domain`, `application` (ports, commands, queries), `infrastructure`, `transport/grpc` kèm `Policy`.
3. Tạo `configs/<name>.yaml` và (nếu cần DB) `migrations/<name>/`, thêm tên vào `DB_SERVICES` trong `Makefile`.
4. Tạo `cmd/server/<name>/server.go` gọi `bootstrap.Register`, rồi blank-import nó trong `cmd/server/main.go`.

## Test

```bash
make test
go test -race ./...
```

Use case được test bằng fake trong `application/apptest`, không cần Docker.

## Hạn chế đã biết

- Upload qua presigned URL không có bước xác nhận, nên file ở trạng thái PENDING cho tới khi worker đánh dấu FAILED (sau `cleanup.pending_timeout`). Hướng sửa: RPC `ConfirmUpload` hoặc S3 Event Notification → SQS.
- CORS đang để `*` cùng `credentials: true`; cần đổi sang danh sách origin cụ thể khi có frontend.
- S3 dùng static credentials; trên AWS nên dùng credential chain mặc định (IAM role).
