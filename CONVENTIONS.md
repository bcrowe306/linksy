# Linksy - Engineering Conventions & Architecture Guidelines

This document outlines the engineering principles, directory layout, coding standards, testing methodologies, security practices, and CI/CD workflow for the Linksy project.

---

## 1. Project Structure

Standard Go Clean-Architecture layout coupled with a Vite/Vue 3 modern frontend:

```text
linksy/
├── .github/
│   └── workflows/
│       └── ci.yml               # Automated testing, building, and deployment pipeline
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/                     # Fiber HTTP handlers, routers, and middlewares
│   │   ├── handler/             # Route handlers (auth, links, qr, analytics, teams, docs)
│   │   ├── middleware/          # Auth, CORS, RateLimiter, Logger, Recovery
│   │   └── router.go            # Central route registration & versioning (/api/v1)
│   ├── config/                  # Environment & config loader (Render / local)
│   ├── models/                  # Domain entity definitions
│   ├── repository/              # PostgreSQL data access layer & queries
│   ├── service/                 # Business logic interfaces and implementations
│   ├── qr/                      # QR code generation, styling, logo embedding, export engine
│   ├── storage/                 # Document storage abstraction (local disk / S3 / cloud)
│   └── auth/                    # WorkOS client, session, and RBAC enforcement
├── migrations/                  # SQL schema migrations (up/down)
├── test/
│   ├── e2e/                     # End-to-end integration test harness
│   └── mocks/                   # Test mocks and fixtures
├── web/                         # Vue 3 Frontend SPA
│   ├── src/
│   │   ├── assets/              # Static styling, themes
│   │   ├── components/          # Reusable UI widgets (QR preview, analytics charts, etc.)
│   │   ├── views/               # Pages (Dashboard, Links, QR Codes, Analytics, Settings, Docs)
│   │   ├── stores/              # Pinia state stores (auth, links, qrcodes, theme)
│   │   ├── router/              # Vue Router configuration
│   │   └── api/                 # Axios / Fetch client wrappers
│   ├── index.html
│   ├── package.json
│   ├── tailwind.config.js
│   └── vite.config.js
├── docs/                        # API specs (OpenAPI), user guides, architecture diagrams
├── render.yaml                  # Render Infrastructure-as-Code Blueprint
├── CONVENTIONS.md
├── REQUIREMENTS.md
└── Makefile                     # Developer productivity shortcuts (make dev, make test, etc.)
```

---

## 2. Development & Testing Conventions

### 2.1 Test-Driven & Verified Code
1. **Unit Testing**:
   - Every service, QR generator module, slug generator, and utility must have co-located `*_test.go` unit tests.
   - Maintain high coverage on critical paths: dynamic redirect routing, access control / permissions, slug collision handling, QR rendering.
2. **Automated Test Harness**:
   - Develop integration test harnesses (`test/e2e/`) that spin up a test Fiber app instance to verify HTTP endpoints, header parsing, payload validation, and analytics ingestion.
   - Frontend component tests using Vitest or Playwright for critical UI interactions (QR customization preview, short link creation modal).
3. **Verification Before Push**:
   - Run verification checks locally via `make test` and `make check` before committing.

---

## 3. Security Conventions

1. **Open Redirect Prevention & Destination Sanitization**:
   - All user-supplied destination URLs must be strictly parsed and validated against allowed protocols (`http://`, `https://`).
   - Disallow loopback, private IP ranges (`10.0.0.0/8`, `192.168.0.0/16`, `127.0.0.1`), and malformed schemes to prevent SSRF and phishing redirects.
2. **Authentication & Authorization**:
   - WorkOS token validation on all protected endpoints.
   - Granular RBAC checks: Verify team membership and user permissions before reading/mutating links, QR codes, or team settings.
3. **Rate Limiting & Abuse Prevention**:
   - Go Fiber rate-limiter middleware enabled on public redirect routes, login callbacks, and public API endpoints.
4. **Document Upload Protection**:
   - Enforce MIME-type sniffing and file signature validation (not just file extension).
   - Enforce strict size limits (e.g. 15MB for documents/menus).
   - Store uploads outside webroot or stream via signed URLs with randomized cryptographic storage keys.
5. **Secure Headers & Secrets**:
   - Zero hardcoded secrets: all credentials (WorkOS API keys, DB credentials) must be supplied via environment variables.
   - Apply security headers: Content-Security-Policy, X-Frame-Options, X-Content-Type-Options.

---

## 4. Version Control & Git Strategy

1. **Repository**: GitHub repo `bcrowe306/linksy`.
2. **Branching Strategy**:
   - `main`: Production-ready code, continuously deployed to Render.
   - Feature branches: `feat/<feature-name>`, `fix/<bug-name>`, `docs/<topic>`.
3. **Commit Messages**:
   - Use Conventional Commits standard:
     - `feat: add PDF and JPEG export support for custom QR codes`
     - `fix: prevent potential open redirect on unvalidated scheme`
     - `test: add e2e test harness for click tracking analytics`
     - `docs: update OpenAPI schema with document upload endpoint`
4. **Git CLI (`gh`)**:
   - Manage pull requests, releases, and repository configuration via `gh`.

---

## 5. CI / CD & Deployment Pipeline

1. **GitHub Actions (`.github/workflows/ci.yml`)**:
   - **Lint & Vet**: `golangci-lint` for Go, `eslint` for Vue.
   - **Test**: Run Go unit & integration tests with coverage reporting.
   - **Frontend Build**: `npm run build` to verify Vue 3 bundle integrity.
   - **Deploy Hook**: Trigger Render deployment webhook upon successful build on `main`.
2. **Render Hosting**:
   - Render Web Service for Go Fiber API server and static Vue SPA delivery.
   - Render Managed PostgreSQL database instance.
