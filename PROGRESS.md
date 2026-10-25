# Lynqr Production Readiness: Progress Tracker

Goal: a production-ready SaaS with no gaps, per `REQUIREMENTS.md` and `CONVENTIONS.md`.
Each area ships on its own `feat/` branch, is verified with `make test` + `npm run build`,
merged after CI passes, and then checked live with the test-drive suite against https://test.lynqr.app.

Status legend: ✅ done · 🚧 in progress · ⏳ queued · ❓ blocked on input

| # | Area | Branch | Status |
|---|------|--------|--------|
| 1 | Auth & tenancy hardening | `feat/auth-tenancy` | ✅ merged (#1), live test-drive 63/63 API + 13/13 UI |
| 2 | Theme system + modern UI redesign | `feat/ui-redesign` | ✅ merged (#2), live UI 15/15 + API pytest 15/15 |
| 3 | QR studio: real preview, all types, styling, exports | `feat/qr-studio` | ✅ merged (#3), live API 64/64 + UI 14/14 |
| 4 | Links: password, UTM, expiry & pause | `feat/link-controls` | ✅ merged (#4), live API 64/64 + UI 15/15 |
| 5 | Team & user management UI | `feat/team-management-ui` | ✅ merged (#5, #6), live API 64/64 + UI 15/15 |
| 6 | Dashboard & analytics (charts, ranges, geo, export, live feed, CF geo headers) | `feat/analytics` | ✅ merged (#7), live API 64/64 + UI 16/16 |
| 7 | Tooltips, Quick Docs, shortcuts, user guides | `feat/help-docs` | 🚧 PR open |
| 8 | Stripe billing (7-day no-card trial · $5/mo annual or $8.99 monthly · per workspace) | `feat/billing` | ✅ PR open 2026-10-25 |
| 9 | SEO & public marketing site | `feat/seo-marketing` | ⏳ requested 2026-10-25 |
| 10 | Test suites (Go, Vitest, Playwright, live test-drive) | `feat/test-suites` | ⏳ (tests also added per area) |
| 11 | Email notifications (Resend) | `feat/email` | ✅ PR open 2026-10-25 |
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
- [x] CI + merge + live test-drive (rate limiter verified per client IP in Render logs)

## 4. Link controls (feat/link-controls)
- [x] Password protection: bcrypt hashes (never returned by the API), branded unlock page that doesn't reveal the destination, signed per-link unlock cookie (12h, revoked when the password changes), unlock attempts rate-limited; protected links' QR scans route through the unlock page without double counting
- [x] UTM auto-tagging (source/medium/campaign/term/content) appended at redirect time without overwriting existing parameters; presets and a live tagged-URL preview in the UI
- [x] Pause and expiry with an optional fallback URL; paused/expired visits return a branded 410 and aren't counted as clicks
- [x] Editable back-half with collision checks; default titles from the destination host/path
- [x] "Create QR" from any link attaches a dynamic QR code to it (`owns_link` keeps the link when that QR is deleted)
- [x] Fixed: the unlock form's redirect was blocked by CSP `form-action`
- [x] Tests: e2e for UTM, passwords (wrong/right/cookie/revocation/removal), pause/expiry/fallback, slug changes, QR attachment; live Playwright unlock flow
- [x] CI + merge + live test-drive

## 5. Team & user management UI (feat/team-management-ui)
- [x] Team & Settings tabs (deep-linkable `?tab=`): Members, General, Audit log (admins+)
- [x] Members: search, role dropdown (hierarchy-aware: only owners grant/manage admins), suspend/reactivate, password-reset email, transfer ownership (typed confirmation), remove; invite modal with role descriptions and a one-time copyable invite link; pending invitations with revoke; roles & permissions matrix
- [x] General: rename, role/members/created summary, leave workspace, delete workspace (type-the-name confirmation)
- [x] Audit log: human-readable actions with actor, affected member, details and IP; category filter; pagination; CSV export
- [x] Profile tabs: profile (avatar upload, client-side center-crop to 256px), security (change password, active sessions, sign out others), API keys (expiry, last used, show-once secret, revoke), workspaces (invitations for you: accept/decline; switch), notifications (preferences + report time zone)
- [x] Backend: notification preferences (`/user/preferences`, migration 0004); audit entries resolve the affected member
- [x] Tests: e2e for preferences and session revocation; two-user local Playwright flow (invite → accept → role → suspend → reactivate → audit → rename → API key → prefs → leave); live suite updated
- [x] CI + merge + live test-drive

## 6. Dashboard & analytics (feat/analytics)
- [x] Ingestion: bot/link-preview filtering, referrer stored as domain only, channel classification (direct/social/email/search/web/qr), richer device/browser/OS parsing, country normalization, live event hub
- [x] Cloudflare "Add visitor location headers" managed transform enabled on lynqr.app (city/region now captured)
- [x] Report API (`/analytics/report`): any range ≤ 2 years in the viewer's IANA time zone, hourly/daily/weekly zero-filled series (DST-correct rollup from UTC hours), previous-period comparison, unique visitors, weekday×hour heatmap, countries/regions/cities, devices/browsers/OS, channels/referrers, top links and QR codes
- [x] Exports: raw events CSV (formula-injection safe, 100k cap), report JSON, printable PDF report with KPI tiles, trend chart and tables
- [x] Live websocket (`/analytics/live`) with same-origin check; dashboard live feed with reconnect/backoff
- [x] Charts built to the dataviz method: validated two-series palette (light+dark), sequential blue ramp, thin marks, crosshair tooltips, legends, partial-period fading, and a table view for every chart
- [x] Dashboard: greeting, onboarding checklist, KPI tiles with deltas + sparklines, trend, live activity, top links/QR/countries, recent assets; Analytics: date-range presets/custom, asset filter, shareable URL, KPIs, trend, world choropleth, locations, tech breakdowns, channels, referrers, heatmap, top assets
- [x] Tests: ingestion enrichment & bot filtering, tz/DST-correct series and heatmap, previous period, report/export e2e (incl. CSV injection, tenant isolation), websocket e2e (incl. cross-origin rejection); live Playwright analytics test
- [x] CI + merge + live test-drive

## 7. Help, docs & usability (feat/help-docs)
- [x] Nine user guides in `docs/guides/*.md` (getting started, dynamic vs static, QR types with per-type anchors, design for scannability, short links, documents, analytics, teams & roles, API), embedded in the binary for reuse by public SEO pages
- [x] Help drawer: searchable guide library, reader with deep-linkable sections, related guides; contextual "Learn more" links on Links, QR Studio (per type, design tips), Documents, Analytics, Team settings, API keys, onboarding
- [x] Keyboard shortcuts with an overlay (`?`): `/` search, `H` help, `T` theme, `N`+`L/Q/D/S` create, `G`+`H/L/Q/D/A/S/P` navigate; disabled while typing or in dialogs
- [x] Complete OpenAPI 3.1 spec (`docs/openapi.yaml`, served as JSON/YAML): every API-key endpoint, schemas for every QR content type and style option, errors, rate limits, and code samples in cURL/Go/Python/JavaScript; drift test fails if a public route is undocumented
- [x] API Reference page: four-language quick start, theme-matched embedded Scalar reference, spec download
- [x] Tooltips across forms (already added per item); chart axis end-label fix
- [ ] CI + merge + live test-drive

## 8. Stripe billing
Done:
- [x] Migration 0005: per-workspace trial (7 days from creation, existing workspaces get 7 days from today) and subscription columns, plus a `stripe_events` idempotency table
- [x] `internal/billing`: Stripe REST client (no SDK); catalog bootstrap by lookup key (`lynqr_pro_monthly` $8.99, `lynqr_pro_annual` $60) and customer portal config at startup; webhook signature verification
- [x] Checkout with promotion codes and live server-side preview; 100%-off codes need no card; the remaining trial days carry over to the subscription
- [x] Webhooks (`/api/webhooks/stripe`) sync status, interval, renewal, cancellation and discount; replays are ignored
- [x] Lapse enforcement: the workspace becomes read-only (402 `subscription_required` on writes, effective permissions in `/auth/me`) and links, QR scans and document links show an inactive page (410)
- [x] Settings → Billing tab (plans, interval toggle, code preview, `?code=` prefill, portal, status); global trial, past-due and read-only banners
- [x] Go tests with a fake Stripe (state machine, signature, idempotency, RBAC, 402/410); Render env vars set
Original plan:
- Per-workspace subscription: 7-day trial without a card, then $5/mo billed annually ($60) or $8.99 monthly; owner-only billing page; Stripe Checkout + Customer Portal; webhook-driven status
- Lapsed or expired trial → read-only workspace, and links/QR codes show a "subscription inactive" page
- **Checkout codes** via Stripe Coupons + Promotion Codes: code field on the billing page with a live server-side preview ("50% off for 3 months"), applied to the Checkout session (plus Stripe's own code field); 100%-off codes create card-free comp accounts (`payment_method_collection: if_required`); `?code=` deep links from pricing; redemption stored on the workspace and in the audit log
- Optional: Lynqr-side trial-extension codes (Stripe codes can't lengthen trials)
- Restricted key needs: Coupons + Promotion Codes (write, for the admin console), Customers, Checkout Sessions, Subscriptions, Billing Portal, Prices

## 8b. Platform admin console
Done (`feat/admin`):
- [x] Migration 0006: platform-admin and suspension flags, admin session elevation, platform audit log, blocklist, abuse reports, trial codes
- [x] `ADMIN_EMAILS` bootstrap; `/admin` re-authentication by password (WorkOS) or a 6-digit email code (Resend), unlocking for 1 hour per session; API keys never reach admin routes
- [x] Overview metrics (users, workspaces, plans, list-price MRR, trial conversion, daily signups, content and engagement)
- [x] Workspaces: search, detail, extend trial, comp, suspend (members read-only, codes show a disabled page); Users: search, suspend (sessions ended, sign-in and API keys blocked), grant or revoke staff
- [x] Codes: Stripe checkout codes (create, deactivate, copy deep link) and Lynqr trial codes (redeemed from the Billing tab)
- [x] Moderation: public `/report` form, report queue, asset search, disable or restore links and QR codes, domain blocklist enforced at create, update and redirect time
- [x] Append-only platform audit log for every admin action; Stripe API version pinned to 2025-08-27.basil
Original plan:
- Platform-admin flag on users (separate from workspace roles); first admins bootstrapped from the `ADMIN_EMAILS` env var (verified accounts only); admins can grant or revoke others in the console
- Same WorkOS login; entering `/admin` requires password re-auth within 15 minutes → short-lived admin session; all `/api/v1/admin/*` routes gated server-side; `noindex`
- **Checkout codes:** create, list and deactivate Stripe coupons + promotion codes (percent/amount, duration, max redemptions, expiry, plan restriction), view redemptions; Lynqr trial-extension codes
- **Workspaces & users:** search; view plan, trial, members and usage; extend trial; comp; suspend or unsuspend
- **Platform metrics:** signups, trials, active subscriptions, MRR, trial→paid conversion, links/QR created, clicks/scans per day
- **Abuse moderation:** search links/QR by destination or domain, disable or restore with reason (shows a "disabled for abuse" page), domain blocklist checked at create/update time, user reports from the public error page
- **Platform audit log:** every admin action with actor, target, before/after and IP; append-only

## 9. SEO & marketing site (plan)
- [x] **Landing page (requested 2026-10-25):** static, crawlable `home.html` (a second Vite entry) served at `/` to visitors without a session; signed-in users get the app. It has a hero with real product screenshots (light and dark), features, QR Studio and analytics spotlights, how it works, pricing with an annual/monthly toggle, FAQ, and a CTA. `?code=` is carried through sign-up to Billing.
- [x] Public guides at `/guides` and `/guides/{slug}`, server-rendered from `docs/guides` (goldmark) with the landing page's CSS/JS, plus TechArticle JSON-LD and a sign-up CTA
- [x] Terms of Service, Privacy Policy and Acceptable Use Policy at `/terms`, `/privacy` and `/acceptable-use` (`docs/legal`), linked from the footer and sign-up form. **Needs a lawyer's review before relying on it.**
- [x] Free QR code generator at `/qr-code-generator` plus per-type pages (Wi-Fi, vCard, email, SMS, phone, text). Static codes only, nothing stored, live preview with a scan check, PNG/SVG downloads, rate-limited public API, blocklist enforced, WebApplication + FAQ JSON-LD, and a CTA to dynamic codes
- [x] Open Graph/Twitter card image, JSON-LD (Organization, SoftwareApplication with offers, FAQPage), canonical URL, `robots.txt` and `sitemap.xml`; test and preview hosts get `noindex` and `Disallow: /`
- Server-rendered (Go `html/template`) public pages, served to visitors without a session so crawlers get real HTML: landing `/`, `/pricing`, `/features`, `/about`, legal (`/terms`, `/privacy`)
- Free public QR generator (no login, static codes, rate-limited) plus one landing page per type (`/qr-code-generator/wifi`, `/vcard`, `/pdf`, `/app-store`, …) targeting high-volume searches, with signup CTAs for dynamic codes and analytics
- Guides/blog section (`/guides/...`) from embedded Markdown, shared with in-app help (item 7)
- Per-page title/description/canonical, Open Graph + Twitter cards with generated OG images, JSON-LD (Organization, SoftwareApplication with offers, FAQPage, BreadcrumbList, HowTo)
- `sitemap.xml`, `robots.txt`; `noindex` on test.lynqr.app and every app/auth route; short links and scan URLs are `noindex` + `X-Robots-Tag`
- Core Web Vitals: no JS required for marketing pages, inline critical CSS, self-hosted fonts with preload, image dimensions, compression and caching headers
- Search Console / Bing verification hooks via env vars; optional privacy-friendly analytics snippet

## Open questions / blocked items
- Stripe: decided (see item 8). The live key is used on test.lynqr.app by choice, so automated tests stop at the Checkout page.
- Email: Resend, from `noreply@lynqr.app` (provided 2026-10-25).
- WorkOS: confirm whether the environment emails password-reset links itself (otherwise item 8 sends them).

## Notes
- Production data was reset by migration `0001_init` (approved: test data only).
- Local dev without WorkOS: `DEV_LOGIN=true APP_URL=http://localhost:8080 make dev-backend` shows a dev sign-in on /login.

## 11. Email notifications (Resend)
Done (`feat/email`):
- [x] `internal/email`: Resend sender, a branded HTML and plain-text layout, and one-click `List-Unsubscribe`
- [x] Sent: welcome; new-browser sign-in and password-change security alerts (always on); invitations (with the accept link) and "member joined" for the inviter; role changes; subscription confirmed, payment failed, subscription ended; trial reminders at 3 days, 1 day and on end; an opt-in weekly analytics report; abuse reports to staff
- [x] Preferences respected (team activity, usage alerts, weekly report); signed `/email/unsubscribe` links; scheduled emails recorded in `email_log` so each goes out once
- [x] Test-suite addresses are never emailed (`EMAIL_SUPPRESS_PATTERN`); password reset and verification emails stay with WorkOS
