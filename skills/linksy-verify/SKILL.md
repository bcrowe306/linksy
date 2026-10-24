---
name: linksy-verify
description: Comprehensive test and verification skill for Linksy (Go Fiber v3 backend, PostgreSQL migrations, QR code generation, Vue 3 frontend build, and API integration test harnesses).
---

# Linksy Verification & Quality Skill

Use this skill whenever verifying changes, running test suites, checking compilation, running database migrations, or ensuring the integrity of the Linksy application.

## Prerequisites
- Go 1.24+ (checked via `go version`)
- Node.js 20+ and npm (checked via `node -v` and `npm -v`)
- Git (`git status`)

## Verification Steps

### 1. Backend Verification
Run Go formatting, vet, and unit tests:
```bash
go vet ./...
go test -v -race ./...
```

To run a specific package test (e.g., QR generator or analytics):
```bash
go test -v ./internal/qr/...
go test -v ./internal/api/handler/...
```

### 2. Integration / E2E Test Harness
Run the API and redirection test harness:
```bash
go test -v ./test/e2e/...
```

### 3. Frontend Verification
Navigate to `web/` and verify linting and production build:
```bash
cd web
npm run build
```

### 4. Security Audit
Check for known vulnerabilities or misconfigurations:
```bash
go list -m all
```
Ensure no unvalidated redirect logic or insecure file upload paths exist in handlers.
