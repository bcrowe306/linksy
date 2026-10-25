#!/usr/bin/env python3
"""
Linksy Full Spectrum QA & Test-Drive Automation Suite
Integrates:
  - Cloudflare API (Zone, DNS routing, R2 bucket storage)
  - Render API (Compute service status, live deployment verification)
  - WorkOS API (User management verification, direct password reset/change, cleanup)
  - Linksy REST API (Sign-up, session issuance, API key auth, link & QR generation)
  - Playwright Browser Automation (UI sign-up/login, theme cycling, profile & API keys,
    QR Studio unsaved changes navigation guard, link creation & 302 redirection)
"""

import os
import sys
import time
import json
import random
import string
import argparse
import subprocess
import urllib.request
import urllib.error
import http.cookiejar

from integrations import CloudflareClient, RenderClient, WorkOSClient, get_secret

class LinksyTestHarness:
    def __init__(self, base_url="https://test.lynqr.app", headed=False, skip_ui=False, ui_only=False):
        self.base_url = base_url.rstrip("/")
        self.headed = headed
        self.skip_ui = skip_ui
        self.ui_only = ui_only
        self.cj = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.cj))
        self.token = ""
        self.user = None
        self.api_key = ""
        self.key_id = ""

        # Generate unique dummy credentials
        suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=6))
        ts = int(time.time())
        self.dummy_first_name = "Alex"
        self.dummy_last_name = f"TestDrive_{suffix}"
        self.dummy_email = f"testdrive_{ts}_{suffix}@lynqr.app"
        self.dummy_password = f"P@ssw0rd_{suffix}_99!"
        self.dummy_new_password = f"NewP@ssw0rd_{suffix}_99!"

        self.passed = 0
        self.failed = 0
        self.cf = CloudflareClient()
        self.render = RenderClient()
        self.workos = WorkOSClient()

    def log(self, step, msg, status="INFO"):
        colors = {
            "INFO": "\033[94m",
            "PASS": "\033[92m",
            "FAIL": "\033[91m",
            "WARN": "\033[93m"
        }
        reset = "\033[0m"
        prefix = f"{colors.get(status, '')}[{status}]{reset}"
        print(f"{prefix} {step}: {msg}")

    def request(self, method, path, data=None, headers=None, auth=True):
        url = f"{self.base_url}{path}"
        body = json.dumps(data).encode("utf-8") if data else None
        req = urllib.request.Request(url, data=body, method=method)
        req.add_header("Content-Type", "application/json")
        req.add_header("Accept", "application/json")
        req.add_header("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Linksy-TestDrive/2.0")
        if auth and self.token and (not headers or "Authorization" not in headers):
            req.add_header("Authorization", f"Bearer {self.token}")
        if headers:
            for k, v in headers.items():
                req.add_header(k, v)

        try:
            with self.opener.open(req) as resp:
                raw_bytes = resp.read()
                try:
                    resp_body = raw_bytes.decode("utf-8")
                    return resp.status, json.loads(resp_body)
                except Exception:
                    return resp.status, raw_bytes
        except urllib.error.HTTPError as e:
            err_body = e.read().decode("utf-8")
            try:
                return e.code, json.loads(err_body)
            except Exception:
                return e.code, err_body
        except Exception as e:
            return 500, {"error": str(e)}

    # --------------------------------------------------------------------------
    # 1. Cloudflare Infrastructure Verification
    # --------------------------------------------------------------------------
    def test_cloudflare(self):
        print("\n--- [1] Cloudflare Infrastructure Verification ---")
        if not self.cf.is_configured():
            self.log("Cloudflare", "Skipped: CLOUDFARE_API_TOKEN not configured", "WARN")
            return

        # Check Zone
        zone = self.cf.get_zone()
        if isinstance(zone, dict) and zone.get("id"):
            self.passed += 1
            self.log("Cloudflare Zone", f"Zone {zone.get('name')} active (ID: {zone.get('id')})", "PASS")
            
            # Check DNS Record
            records = self.cf.get_dns_records(zone["id"], "test.lynqr.app")
            if records and "onrender.com" in records[0].get("content", ""):
                self.passed += 1
                self.log("Cloudflare DNS", f"test.lynqr.app -> {records[0].get('content')} (Proxied: {records[0].get('proxied')})", "PASS")
            else:
                self.failed += 1
                self.log("Cloudflare DNS", "DNS record for test.lynqr.app not found or misconfigured", "FAIL")
        else:
            self.failed += 1
            self.log("Cloudflare Zone", f"Zone query failed: {zone}", "FAIL")

        # Check R2 Storage
        buckets = self.cf.get_r2_buckets()
        bucket_names = [b.get("name") for b in buckets]
        if "lynqr-storage" in bucket_names:
            self.passed += 1
            self.log("Cloudflare R2", f"Verified bucket lynqr-storage exists among {len(bucket_names)} buckets", "PASS")
        else:
            self.failed += 1
            self.log("Cloudflare R2", f"lynqr-storage missing from {bucket_names}", "FAIL")

    # --------------------------------------------------------------------------
    # 2. Render Compute & Deployment Verification
    # --------------------------------------------------------------------------
    def test_render(self):
        print("\n--- [2] Render Compute & Deployment Verification ---")
        if not self.render.is_configured():
            self.log("Render", "Skipped: RENDER_API_KEY not configured", "WARN")
            return

        services = self.render.list_services()
        lynqr_svc = None
        for s in services:
            svc = s.get("service", {})
            if "lynqr" in svc.get("name", "").lower():
                lynqr_svc = svc
                break

        if lynqr_svc:
            self.passed += 1
            status = lynqr_svc.get("suspended")
            self.log("Render Service", f"Service '{lynqr_svc.get('name')}' is {status}", "PASS")

            deploy = self.render.get_latest_deploy(lynqr_svc["id"])
            if deploy and deploy.get("status") in ["live", "build_in_progress", "update_in_progress"]:
                self.passed += 1
                commit_msg = deploy.get("commit", {}).get("message", "N/A")[:40]
                self.log("Render Deployment", f"Live deploy: {deploy.get('id')} ({commit_msg}...)", "PASS")
            else:
                self.failed += 1
                self.log("Render Deployment", f"Deploy not live: {deploy}", "FAIL")
        else:
            self.failed += 1
            self.log("Render Service", "Could not find lynqr service in Render account", "FAIL")

    # --------------------------------------------------------------------------
    # 3. Core REST API & WorkOS Test Flow
    # --------------------------------------------------------------------------
    def test_api_suite(self):
        print("\n--- [3] Core REST API & Authentication Test Flow ---")
        # 1. Health
        status, data = self.request("GET", "/health")
        if status == 200 and isinstance(data, dict) and data.get("status") in ["ok", "healthy"]:
            self.passed += 1
            self.log("Health Check", f"Status: 200, Service: {data.get('service')}", "PASS")
        else:
            self.failed += 1
            self.log("Health Check", f"Failed ({status}): {data}", "FAIL")

        # 1.5 Unauthenticated Route Blocking
        unauth_status, _ = self.request("GET", "/api/v1/links")
        if unauth_status == 401:
            self.passed += 1
            self.log("Unauthenticated Route Block", "Unauthenticated request to /api/v1/links blocked (401 Unauthorized)", "PASS")
        else:
            self.failed += 1
            self.log("Unauthenticated Route Block", f"Expected 401, got {unauth_status}", "FAIL")

        # 2. Sign-Up
        signup_payload = {
            "first_name": self.dummy_first_name,
            "last_name": self.dummy_last_name,
            "email": self.dummy_email,
            "password": self.dummy_password
        }
        status, data = self.request("POST", "/api/v1/auth/signup", signup_payload)
        if status in (200, 201) and isinstance(data, dict) and data.get("user"):
            self.passed += 1
            self.token = data.get("token")
            self.user = data.get("user")
            self.log("User Sign-Up", f"Account created for {self.dummy_email} (ID: {self.user.get('id')})", "PASS")
        else:
            self.failed += 1
            self.log("User Sign-Up", f"Failed ({status}): {data}", "FAIL")

        # 3. Password Login
        login_payload = {
            "email": self.dummy_email,
            "password": self.dummy_password
        }
        status, data = self.request("POST", "/api/v1/auth/login", login_payload)
        if status == 200 and isinstance(data, dict) and data.get("token"):
            self.passed += 1
            self.token = data.get("token")
            self.log("Password Login", "Authenticated successfully with primary credentials", "PASS")
        else:
            self.failed += 1
            self.log("Password Login", f"Failed ({status}): {data}", "FAIL")

        # 4. Profile Management
        status, data = self.request("GET", "/api/v1/user/profile")
        if status == 200 and isinstance(data, dict) and data.get("user"):
            self.passed += 1
            u = data["user"]
            self.log("Get Profile", f"Retrieved profile for {u.get('name')} ({u.get('email')})", "PASS")
        else:
            self.failed += 1
            self.log("Get Profile", f"Failed ({status}): {data}", "FAIL")

        updated_first = self.dummy_first_name + "Updated"
        update_payload = {
            "first_name": updated_first,
            "last_name": self.dummy_last_name,
            "avatar_url": "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?auto=format&fit=crop&w=120&h=120&q=80"
        }
        status, data = self.request("PUT", "/api/v1/user/profile", update_payload)
        if status == 200 and isinstance(data, dict) and data.get("user", {}).get("first_name") == updated_first:
            self.passed += 1
            self.log("Update Profile", f"Updated first name to {updated_first}", "PASS")
        else:
            self.failed += 1
            self.log("Update Profile", f"Failed ({status}): {data}", "FAIL")

        # 5. Developer API Keys
        key_payload = {"name": f"Harness-Key-{int(time.time())}", "expires_in_days": 30}
        status, data = self.request("POST", "/api/v1/api-keys", key_payload)
        if status in (200, 201) and isinstance(data, dict) and data.get("api_key", {}).get("key"):
            key_obj = data["api_key"]
            self.api_key = key_obj["key"]
            self.key_id = key_obj["id"]
            self.passed += 1
            self.log("Create API Key", f"Generated API Key: {key_obj['key_prefix']} (ID: {self.key_id})", "PASS")
        else:
            self.failed += 1
            self.log("Create API Key", f"Failed ({status}): {data}", "FAIL")

        # API Key Authenticated Calls
        if self.api_key:
            auth_header = {"Authorization": f"Bearer {self.api_key}"}
            status, data = self.request("GET", "/api/v1/links", headers=auth_header)
            if status == 200:
                self.passed += 1
                self.log("API Key Auth (List Links)", f"Successfully authenticated via Bearer {self.api_key[:12]}...", "PASS")
            else:
                self.failed += 1
                self.log("API Key Auth (List Links)", f"Failed with API key ({status}): {data}", "FAIL")

            qr_payload = {
                "type": "website",
                "title": "Harness QR Code",
                "target_url": "https://lynqr.app",
                "payload": "https://lynqr.app",
                "is_dynamic": True,
                "style": {"fg_color": "#10b981", "bg_color": "#030712"}
            }
            status, data = self.request("POST", "/api/v1/qr", data=qr_payload, headers=auth_header)
            if status in (200, 201):
                self.passed += 1
                qr_id = data.get("id") if isinstance(data, dict) else None
                self.log("API Key Auth (Create QR)", f"Created QR code via API key successfully (ID: {qr_id})", "PASS")
                if qr_id:
                    # Test export without auth header (<img> tag compatibility)
                    exp_status, exp_body = self.request("GET", f"/api/v1/qr/{qr_id}/export?format=png&size=240", auth=False)
                    if exp_status == 200 and isinstance(exp_body, (bytes, bytearray)) and len(exp_body) > 0:
                        self.passed += 1
                        self.log("Public QR Export", f"Fetched QR image preview without auth ({len(exp_body)} bytes)", "PASS")
                    else:
                        self.failed += 1
                        self.log("Public QR Export", f"Failed public QR export ({exp_status}): {exp_body[:100] if isinstance(exp_body, (bytes, str)) else exp_body}", "FAIL")
            else:
                self.failed += 1
                self.log("API Key Auth (Create QR)", f"Failed creating QR via API key ({status}): {data}", "FAIL")

        # Revoke Key
        if self.key_id:
            status, data = self.request("DELETE", f"/api/v1/api-keys/{self.key_id}")
            if status == 200:
                self.passed += 1
                self.log("Revoke API Key", f"Revoked API key {self.key_id}", "PASS")
            else:
                self.failed += 1
                self.log("Revoke API Key", f"Failed ({status}): {data}", "FAIL")

        # WorkOS Password Reset
        status, data = self.request("POST", "/api/v1/auth/password-reset", {"email": self.dummy_email})
        if status == 200 and isinstance(data, dict):
            self.passed += 1
            reset_url = data.get("password_reset_url", "Email Dispatched")
            self.log("Password Reset", f"Initiated WorkOS reset for {self.dummy_email} -> {reset_url[:35]}...", "PASS")
        else:
            self.failed += 1
            self.log("Password Reset", f"Failed ({status}): {data}", "FAIL")

        # WorkOS Direct Password Change
        status, data = self.request("POST", "/api/v1/user/change-password", {"new_password": self.dummy_new_password})
        if status == 200:
            self.passed += 1
            self.log("Change Password", "Updated password directly via WorkOS API", "PASS")
        else:
            self.failed += 1
            self.log("Change Password", f"Failed ({status}): {data}", "FAIL")

        # Re-login with New Password
        status, data = self.request("POST", "/api/v1/auth/login", {"email": self.dummy_email, "password": self.dummy_new_password})
        if status == 200 and isinstance(data, dict) and data.get("token"):
            self.passed += 1
            self.token = data.get("token")
            self.log("Re-login with New Password", "Successfully logged in with new password", "PASS")
        else:
            self.failed += 1
            self.log("Re-login with New Password", f"Failed ({status}): {data}", "FAIL")

    # --------------------------------------------------------------------------
    # 4. Playwright End-to-End Browser Workflows
    # --------------------------------------------------------------------------
    def test_playwright_e2e(self):
        print("\n--- [4] Playwright End-to-End Browser UI Workflows ---")
        env = os.environ.copy()
        env["TEST_BASE_URL"] = self.base_url

        pytest_cmd = [sys.executable, "-m", "pytest", "-v", "tests/test_ui_playwright.py"]
        if self.headed:
            pytest_cmd.append("--headed")

        try:
            res = subprocess.run(pytest_cmd, env=env, capture_output=True, text=True)
            output_lines = res.stdout.strip().split("\n")
            for line in output_lines:
                if "PASSED" in line or "FAILED" in line:
                    parts = line.split()
                    test_name = parts[0].split("::")[-1].split("[")[0]
                    status = "PASS" if "PASSED" in line else "FAIL"
                    if status == "PASS":
                        self.passed += 1
                        self.log(f"UI E2E ({test_name})", "Verified in browser", "PASS")
                    else:
                        self.failed += 1
                        self.log(f"UI E2E ({test_name})", "Browser workflow failed", "FAIL")

            if res.returncode != 0 and self.failed == 0:
                self.failed += 1
                self.log("Playwright Suite", f"Pytest exited with code {res.returncode}\n{res.stderr}", "FAIL")
        except Exception as e:
            self.failed += 1
            self.log("Playwright Suite", f"Execution error: {e}", "FAIL")

    # --------------------------------------------------------------------------
    # 5. WorkOS Verification & Cleanup
    # --------------------------------------------------------------------------
    def cleanup_workos(self):
        print("\n--- [5] WorkOS User Verification & Automated Cleanup ---")
        if not self.workos.is_configured():
            return

        user_id = self.user.get("id") if self.user else None
        if not user_id:
            # Query by email
            res = self.workos.list_users(limit=20)
            for u in res.get("data", []):
                if u.get("email") == self.dummy_email:
                    user_id = u.get("id")
                    break

        if user_id:
            user_data = self.workos.get_user(user_id)
            if isinstance(user_data, dict) and user_data.get("id") == user_id:
                self.passed += 1
                self.log("WorkOS Query", f"Verified user {user_id} active in WorkOS User Management", "PASS")

            # Clean up dummy user
            deleted = self.workos.delete_user(user_id)
            if deleted:
                self.passed += 1
                self.log("WorkOS Cleanup", f"Deleted test dummy user {user_id} from WorkOS", "PASS")
            else:
                self.log("WorkOS Cleanup", f"Notice: Could not delete user {user_id}", "WARN")

    def run_all(self):
        print("=" * 64)
        print(f"🚀 Linksy Full-Spectrum Test-Drive Suite: {self.base_url}")
        print(f"👤 Dummy Target User: {self.dummy_email}")
        print(f"🖥️  Playwright Mode: {'Headed' if self.headed else 'Headless'}")
        print("=" * 64)

        start_time = time.time()

        if self.ui_only:
            self.test_playwright_e2e()
        else:
            self.test_cloudflare()
            self.test_render()
            self.test_api_suite()
            if not self.skip_ui:
                self.test_playwright_e2e()
            self.cleanup_workos()

        duration = time.time() - start_time
        print("\n" + "=" * 64)
        print(f"📊 Test-Drive Summary: {self.passed} Passed, {self.failed} Failed ({duration:.2f}s)")
        print("=" * 64)
        return self.failed == 0

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Linksy Test-Drive Suite")
    parser.add_argument("url", nargs="?", default="https://test.lynqr.app", help="Base URL to test")
    parser.add_argument("--headed", action="store_true", help="Run Playwright in headed browser mode")
    parser.add_argument("--skip-ui", action="store_true", help="Skip Playwright browser UI tests")
    parser.add_argument("--ui-only", action="store_true", help="Run only Playwright browser UI tests")
    args = parser.parse_args()

    harness = LinksyTestHarness(
        base_url=args.url,
        headed=args.headed,
        skip_ui=args.skip_ui,
        ui_only=args.ui_only
    )
    success = harness.run_all()
    sys.exit(0 if success else 1)
