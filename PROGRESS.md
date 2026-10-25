# Lynqr Production Readiness: Progress Tracker

Goal: a production-ready SaaS with no gaps, per `REQUIREMENTS.md` and `CONVENTIONS.md`.
Each area ships on its own `feat/` branch, is verified with `make test` + `npm run build`,
merged after CI passes, and then checked live with the test-drive suite against https://test.lynqr.app.

Status legend: ✅ done · 🚧 in progress · ⏳ queued · ❓ blocked on input

| # | Area | Branch | Status |
|---|------|--------|--------|
| 1 | Auth & tenancy hardening | `feat/auth-tenancy` | ✅ merged (#1), live test-drive 63/63 API + 13/13 UI |
| 2 | Theme system + modern UI redesign | `feat/ui-redesign` | ✅ merged (#2), live UI 15/15 + API pytest 15/15 |
| 3 | QR studio: real preview, all types, styling, exports | `feat/qr-studio` | 🚧 PR open |
| 4 | Links: password, UTM, expiry & pause | `feat/link-controls` | ⏳ |
| 5 | Team & user management UI | `feat/team-management-ui` | ⏳ |
| 6 | Dashboard & analytics (charts, ranges, geo, export, live feed, CF geo headers) | `feat/analytics` | ⏳ |
| 7 | Tooltips, Quick Docs, shortcuts, user guides | `feat/help-docs` | ⏳ |
| 8 | Test suites (Go, Vitest, Playwright, live test-drive) | `feat/test-suites` | ⏳ (tests also added per area) |
| 9 | Stripe billing + email notifications | `feat/billing-email` | ⏸ deferred by user (2026-10-25); revisit last |

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
- [x] Merged, `APP_SECRET` set on Render, live test-drive green (API 63/63, Playwright UI 13/13, pytest API 14/15 → stale `client_id` assertion fixed)

## 2. Theme system & UI redesign (feat/ui-redesign)
- [x] Token-based theming: neutral scale + semantic tokens (`canvas/surface/elevated/subtle/line/fg*`) as CSS variables; light/dark/system, dark by default (PRD §1.12), applied before mount, persisted, follows OS changes
- [x] Design system in `style.css` (buttons, inputs, cards, badges, tables, kbd, skeletons) + UI kit: modal, toasts, confirm dialog, tooltip, dropdown menu, switch, file drop, empty state, page header, copy button, logo
- [x] New shell: grouped, collapsible sidebar (mobile drawer), sticky blurred header with global search (`/`), theme menu, create menu, user menu, workspace switcher
- [x] Rebuilt: auth pages (split-screen brand layout, OAuth buttons, password strength), invite & reset pages, create modal, Links (filters/sort/pause/edit/delete), QR gallery (downloads, type filter), Documents (versions, replace, bulk delete), API reference, help drawer
- [x] Remaining views (dashboard, analytics, studio, settings, profile) theme-correct now; fully rebuilt in items 3, 5 and 6
- [x] Fixed: embedded API reference blocked by X-Frame-Options DENY
- [x] CI + merge + live test-drive

## 3. QR studio (feat/qr-studio)
- [x] New vector-first engine (`internal/qr`): one Scene drawn by SVG, raster (PNG/JPEG via gg) and vector PDF renderers, so exports match the preview
- [x] Styles: 5 module shapes (neighbor-aware fluid/classy), 5 eye frames × 4 eye centers, eye colors, solid/linear(angle)/radial gradients, transparent bg, quiet zone, rounded corners, logo with module excavation + badge, 6 frames (border, banner, top banner, card, badge) with auto-fit, auto-legible labels
- [x] Security: every style value validated/normalized; SVG output contains only numbers, validated colors, escaped text and re-encoded PNG (fixes SVG script injection); remote logo URLs dropped (SSRF)
- [x] Fixes: quiet-zone border offset made eye styling wrong; labels were never drawn; eye/frame/angle options were ignored
- [x] Scannability check: every preview is decoded with ZXing + WCAG contrast, density and quiet-zone warnings
- [x] Types: website, document, vCard (3.0/4.0/MeCard, escaped), Wi-Fi (escaped, WPA3), app stores (device routing + chooser page), text, email, SMS, phone (new)
- [x] Dynamic codes get their own short scan code `/q/{code}`: scans tracked separately from link clicks; paused/expired destinations show a branded page; legacy `/qr/r/{id}` still works
- [x] Document codes resolve the current file (survive replacement); stable `/d/{id}` document links; dynamic vCards hosted as .vcf
- [x] Exports: PNG/JPEG up to 4096px, JPEG on custom background, SVG, PDF A4 sheet and PDF card/sticker
- [x] Studio UI: type picker, per-type forms with tooltips, templates, pattern/colors/logo/frame/advanced panels, debounced live server preview with scan badge, duplicate, unsaved-changes guard, downloads
- [x] Auth limiter split (credentials 30/min, OAuth/invite flows 120/min); logs include CF client IP to verify per-client limiting
- [x] Tests: every shape/eye/frame combination must decode; SVG injection; payload escaping per type; e2e for scans, app routing, vCard hosting, document replacement, static types, preview, export options
- [ ] CI + merge + live test-drive

## Open questions / blocked items
- Stripe: need `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` and the plan/pricing structure.
- Email: need a provider + API key (Resend, Postmark or SES) and a verified sending domain on lynqr.app.
- WorkOS: confirm whether the environment emails password-reset links itself (otherwise item 8 sends them).

## Notes
- Production data was reset by migration `0001_init` (approved: test data only).
- Local dev without WorkOS: `DEV_LOGIN=true APP_URL=http://localhost:8080 make dev-backend` shows a dev sign-in on /login.
