---
name: linksy-test-drive
description: Run the automated Linksy test-drive suite against local or live environment to test Cloudflare DNS & R2, Render deployment, WorkOS authentication & cleanup, REST APIs, and Playwright browser UI workflows.
---

# Linksy Test-Drive Automation Skill

This skill executes a complete, full-spectrum verification and QA suite across all architectural tiers of Linksy:

1. **Cloudflare Infrastructure API**:
   - Zone status (`lynqr.app` active)
   - DNS routing validation (`test.lynqr.app` -> `lynqr-test.onrender.com` CNAME, proxied)
   - Cloudflare R2 bucket accessibility (`lynqr-storage`)
2. **Render Compute & Deployment API**:
   - Web service health (`lynqr-test` active / not suspended)
   - Live deployment verification and Git commit matching
3. **Core REST API & WorkOS Integration**:
   - Health check endpoint verification
   - User registration and session token issuance
   - Password authentication grant
   - Profile management (`GET /api/v1/user/profile`, `PUT /api/v1/user/profile`)
   - Developer API key generation (`lk_live_...`), listing, prefix masking, and revocation
   - API key authorization across protected endpoints (`GET /api/v1/links`, `POST /api/v1/qr`)
   - WorkOS self-service password reset token generation & dispatch
   - Authenticated password updates via WorkOS User Management API
   - Re-authentication with updated credentials
4. **Playwright End-to-End Browser UI Workflows**:
   - **Theme System**: Cycles through OS Specification (System), Light, and Dark themes; verifies `<html>` class toggle and `localStorage` persistence across page reloads.
   - **Sign-Up & Login UI**: Submits registration forms, verifies dashboard transition and session cookies, exercises sign-out and sign-in.
   - **Account Profile & API Keys**: Clicks user avatar in header to navigate to `/profile`, updates profile name, opens "Developer API Keys" tab, generates a new key, verifies full key display modal, checks masked prefix in active table, and revokes the key.
   - **QR Studio Navigation Guard**: Modifies QR code parameters in `/qr-studio`, verifies "Unsaved Changes" badge, attempts navigation to `/links`, intercepts `window.confirm` dialog, confirms warning message, and accepts navigation.
   - **Links Management & Redirection**: Creates short link via UI modal with custom slug, verifies entry in links table, and executes HTTP request asserting 302/301 dynamic redirection.
5. **WorkOS Verification & Automated Cleanup**:
   - Queries WorkOS User Management API to verify user profile existence.
   - Automatically deletes created test dummy users to prevent test artifact accumulation.

---

## When to Run

Run this skill:
1. After making changes to authentication, routing, user profiles, themes, or API key management.
2. Before tagging and pushing a new release to production or staging.
3. Post-deployment on Render to verify live health across Cloudflare, Render, WorkOS, and browser clients.

---

## Execution Modes

### 1. Full Spectrum Verification (Default, Headless Browser)
Runs Cloudflare, Render, WorkOS, REST APIs, and Playwright UI workflows:
```bash
python3 scripts/test_drive.py https://test.lynqr.app
```

### 2. Headed Browser Mode
Runs Playwright with a visible Chromium browser window:
```bash
python3 scripts/test_drive.py https://test.lynqr.app --headed
```

### 3. API-Only Verification (Fast)
Runs infrastructure and REST API checks without launching browser instances:
```bash
python3 scripts/test_drive.py https://test.lynqr.app --skip-ui
```

### 4. UI-Only Verification
Runs Playwright end-to-end browser workflows:
```bash
python3 scripts/test_drive.py https://test.lynqr.app --ui-only
```

### 5. Direct Pytest Invocation
Run or debug individual Playwright tests directly with pytest:
```bash
pytest -v tests/test_ui_playwright.py
pytest -v -k "test_qr_studio_unsaved_changes_warning" tests/test_ui_playwright.py
```

---

## Expected Output

```
================================================================
🚀 Linksy Full-Spectrum Test-Drive Suite: https://test.lynqr.app
👤 Dummy Target User: testdrive_1792892866_u4fa9c@lynqr.app
🖥️  Playwright Mode: Headless
================================================================

--- [1] Cloudflare Infrastructure Verification ---
[PASS] Cloudflare Zone: Zone lynqr.app active (ID: 6b9167c34fbed0de61d7b80120e5cb83)
[PASS] Cloudflare DNS: test.lynqr.app -> lynqr-test.onrender.com (Proxied: True)
[PASS] Cloudflare R2: Verified bucket lynqr-storage exists among 2 buckets

--- [2] Render Compute & Deployment Verification ---
[PASS] Render Service: Service 'lynqr-test' is not_suspended
[PASS] Render Deployment: Live deploy: dep-dbeljnla647c73edbbqg (...)

--- [3] Core REST API & Authentication Test Flow ---
[PASS] Health Check: Status: 200, Service: linksy
[PASS] User Sign-Up: Account created for testdrive_... (ID: user_...)
[PASS] Password Login: Authenticated successfully with primary credentials
[PASS] Get Profile: Retrieved profile for Alex TestDrive_...
[PASS] Update Profile: Updated first name to AlexUpdated
[PASS] Create API Key: Generated API Key: lk_live_... (ID: key_...)
[PASS] API Key Auth (List Links): Successfully authenticated via Bearer lk_live_...
[PASS] API Key Auth (Create QR): Created QR code via API key successfully
[PASS] Revoke API Key: Revoked API key key_...
[PASS] Password Reset: Initiated WorkOS reset for testdrive_... -> https://...
[PASS] Change Password: Updated password directly via WorkOS API
[PASS] Re-login with New Password: Successfully logged in with new password

--- [4] Playwright End-to-End Browser UI Workflows ---
[PASS] UI E2E (test_cloudflare_zone_and_dns): Verified in browser
[PASS] UI E2E (test_cloudflare_r2_storage): Verified in browser
[PASS] UI E2E (test_render_service_health_and_deployment): Verified in browser
[PASS] UI E2E (test_theme_system_toggle_and_persistence): Verified in browser
[PASS] UI E2E (test_signup_and_login_ui): Verified in browser
[PASS] UI E2E (test_profile_and_developer_api_keys): Verified in browser
[PASS] UI E2E (test_qr_studio_unsaved_changes_warning): Verified in browser
[PASS] UI E2E (test_short_link_creation_and_redirection): Verified in browser
[PASS] UI E2E (test_workos_user_verification_and_cleanup): Verified in browser

--- [5] WorkOS User Verification & Automated Cleanup ---
[PASS] WorkOS Query: Verified user user_... active in WorkOS User Management
[PASS] WorkOS Cleanup: Deleted test dummy user user_... from WorkOS

================================================================
📊 Test-Drive Summary: 28 Passed, 0 Failed (30.15s)
================================================================
```
