---
name: linksy-test-drive
description: Run the automated Linksy test-drive suite against local or live environment to test user sign-up, login, profile management, WorkOS password reset, API key generation, authenticated API calls, and key revocation.
---

# Linksy Test-Drive Automation Skill

This skill executes a complete end-to-end verification of Linksy application capabilities:
- User registration with dummy credentials (`Alex`, `TestDrive_<hash>`, `testdrive_<timestamp>@lynqr.app`, secure password)
- Password-based authentication & session issuance
- User profile retrieval & updating (`GET /api/v1/user/profile`, `PUT /api/v1/user/profile`)
- WorkOS self-service password reset token generation & email dispatch (`POST /api/v1/auth/password-reset`)
- Authenticated password updates (`POST /api/v1/user/change-password`)
- API key generation with secure token generation (`POST /api/v1/api-keys`)
- API key verification for protected endpoints (`GET /api/v1/links`, `POST /api/v1/qr`)
- API key revocation (`DELETE /api/v1/api-keys/:id`)

## When to Run

Run this skill:
1. After making changes to authentication, routing, user profiles, or API key management.
2. Before tagging and pushing a new release to production or staging.
3. To verify live production health and integration after deployment.

## Execution

### Run against the live production environment (`https://test.lynqr.app`):
```bash
python3 scripts/test_drive.py https://test.lynqr.app
```

### Run against a local development server (`http://localhost:8080` or `http://localhost:10000`):
```bash
python3 scripts/test_drive.py http://localhost:8080
```

## Expected Output

```
============================================================
🚀 Starting Linksy Test-Drive Suite on https://test.lynqr.app
👤 Target Dummy User: testdrive_1792890..._abc@lynqr.app
============================================================
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
============================================================
📊 Test-Drive Summary: 12 Passed, 0 Failed
============================================================
```
