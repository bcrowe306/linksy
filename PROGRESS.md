# Lynqr Production Readiness: Progress Tracker

Goal: a production-ready SaaS with no gaps, per `REQUIREMENTS.md` and `CONVENTIONS.md`.
Each area ships on its own `feat/` branch, is verified with `make test` + `npm run build`,
merged after CI passes, and then checked live with the test-drive suite against https://test.lynqr.app.

Status legend: ✅ done · 🚧 in progress · ⏳ queued · ❓ blocked on input

| # | Area | Branch | Status |
|---|------|--------|--------|
| 1 | Auth & tenancy hardening | `feat/auth-tenancy` | 🚧 PR open |
| 2 | Theme system + modern UI redesign | `feat/ui-redesign` | ⏳ |
| 3 | QR editor: real preview, 8 types, styling, exports | `feat/qr-studio` | ⏳ |
| 4 | Links: password, UTM, expiry & pause | `feat/link-controls` | ⏳ |
| 5 | Team & user management UI | `feat/team-management-ui` | ⏳ |
| 6 | Dashboard & analytics (charts, ranges, geo, export, live feed, CF geo headers) | `feat/analytics` | ⏳ |
| 7 | Tooltips, Quick Docs, shortcuts, user guides | `feat/help-docs` | ⏳ |
| 8 | Stripe billing + email notifications | `feat/billing-email` | ❓ needs Stripe + email provider keys |
| 9 | Test suites (Go, Vitest, Playwright, live test-drive) | `feat/test-suites` | ⏳ (tests also added per area) |

## 1. Auth & tenancy (feat/auth-tenancy)
- [x] Opaque, server-side sessions (`lqs_` tokens, SHA-256 at rest, HttpOnly + Secure + SameSite=Lax cookie, 30-day sliding expiry, revocable)
- [x] Removed forgeable auth: raw `user_` ids, unsigned `sess_<id>` tokens, unverified JWTs, `usr_` fallbacks, demo user
- [x] `X-Team-ID` honoured only for active memberships; API keys are bound to one team and use the owner's live role
- [x] Personal workspace per signup; multi-workspace membership, create & switch
- [x] Token invitations (`lqi_`, 7-day, single use, email-bound), accept/decline, preview page
- [x] RBAC matrix (`internal/auth/rbac.go`) enforced at the route and service layers; owner > admin > member > viewer hierarchy
- [x] Member suspension and removal (membership only; never deletes the WorkOS account); ownership transfer; leave; delete workspace
- [x] Audit log table and events for member, link, QR, document and API-key changes
- [x] Removed hardcoded owner emails/IDs, `testdrive`/`qa_` admin bypass, and the destructive boot-time user cleanup
- [x] Analytics, exports and documents scoped to the team; QR export requires auth
- [x] Password reset never returns the reset URL; change-password re-verifies the current password and signs out other sessions
- [x] WorkOS email verification step (`/auth/verify-email`); OAuth `state` CSRF cookie; open-redirect-safe `redirect` param
- [x] Security headers (CSP, XFO, nosniff, HSTS, Referrer-Policy), same-origin check for cookie-authenticated writes, auth/API rate limits
- [x] SSRF-hardened destination validation (all private/reserved ranges, encoded IPs, credentials in URLs)
- [x] XSS fix: branded error page escaped; uploaded HTML/SVG served as attachments
- [x] Versioned embedded migrations (`migrations/{postgres,sqlite}`), single-dialect queries via `Rebind`
- [x] Tests: RBAC matrix, sessions, tenancy isolation, hierarchy, invitations, forged tokens, header spoofing, CSRF, API key scope (SQLite + Postgres in CI)
- [x] Test-drive + pytest suites updated for cookie sessions and pre-verified WorkOS fixtures
- [ ] Merge, set `APP_SECRET` on Render, run the live test-drive

## Open questions / blocked items
- Stripe: need `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` and the plan/pricing structure.
- Email: need a provider + API key (Resend, Postmark or SES) and a verified sending domain on lynqr.app.
- WorkOS: confirm whether the environment emails password-reset links itself (otherwise item 8 sends them).

## Notes
- Production data was reset by migration `0001_init` (approved: test data only).
- Local dev without WorkOS: `DEV_LOGIN=true APP_URL=http://localhost:8080 make dev-backend` shows a dev sign-in on /login.
