#!/usr/bin/env python3
"""
Linksy Full Spectrum QA & Test-Drive Automation Suite
Integrates:
  - Cloudflare API (Zone, DNS routing, R2 bucket storage)
  - Render API (Compute service status, live deployment verification)
  - WorkOS API (User management verification, direct password reset/change, cleanup)
  - Linksy Public & Authenticated REST APIs (Docs, Auth, Links, QR, Docs, Analytics, Logout)
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
import requests

from integrations import CloudflareClient, RenderClient, WorkOSClient, get_secret

class LinksyTestHarness:
    def __init__(self, base_url="https://test.lynqr.app", headed=False, skip_ui=False, ui_only=False):
        self.base_url = base_url.rstrip("/")
        self.headed = headed
        self.skip_ui = skip_ui
        self.ui_only = ui_only
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Linksy-TestDrive/2.0"
        })
        self.token = ""
        self.user = None
        self.api_key = ""
        self.key_id = ""
        self.last_headers = {}

        # Generate unique dummy credentials
        suffix = "".join(random.choices(string.ascii_lowercase + string.digits, k=6))
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
        prefix = f"{colors.get(status, "")}[{status}]{reset}"
        print(f"{prefix} {step}: {msg}")

    def request(self, method, path, data=None, files=None, headers=None, auth=True, allow_redirects=True):
        url = f"{self.base_url}{path}" if path.startswith("/") else path
        req_headers = {}
        if auth and self.token and (not headers or "Authorization" not in headers):
            req_headers["Authorization"] = f"Bearer {self.token}"
        if headers:
            req_headers.update(headers)

        try:
            if files:
                resp = self.session.request(
                    method, url, data=data, files=files, headers=req_headers, allow_redirects=allow_redirects, timeout=15
                )
            elif data is not None:
                resp = self.session.request(
                    method, url, json=data, headers=req_headers, allow_redirects=allow_redirects, timeout=15
                )
            else:
                resp = self.session.request(
                    method, url, headers=req_headers, allow_redirects=allow_redirects, timeout=15
                )

            self.last_headers = resp.headers
            ct = resp.headers.get("Content-Type", "")
            if "application/json" in ct:
                try:
                    body = resp.json()
                except Exception:
                    body = resp.text
            elif "image" in ct or "pdf" in ct or "octet-stream" in ct:
                body = resp.content
            else:
                try:
                    body = resp.json()
                except Exception:
                    body = resp.text

            return resp.status_code, body
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
                self.log("Cloudflare DNS", f"test.lynqr.app -> {records[0].get('content')} (Proxied: {records[0].get('proxied')})	", "PASS")
            else:
                self.failed += 1
                self.log("Cloudflare DNS", "DNS record for test.lynqr.app not found or misconfigured", "FAIL")
        else:
            self.failed += 1
            self.log("Cloudflare Zone", "Failed to retrieve Cloudflare zone lynqr.app", "FAIL")

        # Check R2 Storage Bucket
        buckets = self.cf.get_r2_buckets()
        bucket_names = [b.get("name") for b in buckets] if isinstance(buckets, list) else []
        if "lynqr-storage" in bucket_names:
            self.passed += 1
            self.log("Cloudflare R2", f"Verified bucket lynqr-storage exists among {len(bucket_names)} buckets", "PASS")
        else:
            self.failed += 1
            self.log("Cloudflare R2", f"Bucket lynqr-storage missing from {bucket_names}", "FAIL")

    # --------------------------------------------------------------------------
    # 2. Render Compute & Deployment Health
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
    # 3. Exhaustive Public & Authenticated REST API Suite
    # --------------------------------------------------------------------------
    def test_api_suite(self):
        print("\n--- [3] Public-Facing & Authenticated REST API Suite ---")
        # 1. Health & Discovery Endpoints
        status, data = self.request("GET", "/health")
        if status == 200 and isinstance(data, dict) and data.get("status") in ["ok", "healthy"]:
            self.passed += 1
            self.log("Health Check (/health)", f"Status: 200, Service: {data.get('service')}", "PASS")
        else:
            self.failed += 1
            self.log("Health Check (/health)", f"Failed ({status}): {data}", "FAIL")

        status, data = self.request("GET", "/api/v1/health")
        if status == 200 and isinstance(data, dict) and data.get("status") in ["ok", "healthy"]:
            self.passed += 1
            self.log("API v1 Health (/api/v1/health)", "Status: 200 Healthy", "PASS")
        else:
            self.failed += 1
            self.log("API v1 Health (/api/v1/health)", f"Failed ({status}): {data}", "FAIL")

        # 2. Interactive Docs & OpenAPI 3.0 Spec
        status, data = self.request("GET", "/api/docs")
        if status == 200 and "<html" in str(data).lower():
            self.passed += 1
            self.log("Interactive API Docs (/api/docs)", "Loaded HTML OpenAPI Documentation", "PASS")
        else:
            self.failed += 1
            self.log("Interactive API Docs (/api/docs)", f"Failed ({status}): {str(data)[:60]}", "FAIL")

        status, data = self.request("GET", "/api/v1/openapi.json")
        if status == 200 and isinstance(data, dict) and "paths" in data:
            self.passed += 1
            self.log("OpenAPI 3.0 Spec (/api/v1/openapi.json)", f"Valid specification with {len(data['paths'])} paths", "PASS")
        else:
            self.failed += 1
            self.log("OpenAPI 3.0 Spec (/api/v1/openapi.json)", f"Failed ({status}): {data}", "FAIL")

        # 3. Auth Configuration & OAuth Providers
        status, data = self.request("GET", "/api/v1/auth/config")
        if status == 200 and isinstance(data, dict) and data.get("enabled"):
            self.passed += 1
            providers = data.get("providers", [])
            self.log("Auth Configuration (/api/v1/auth/config)", f"Enabled: True, Providers: {', '.join(providers)}", "PASS")
        else:
            self.failed += 1
            self.log("Auth Configuration (/api/v1/auth/config)", f"Failed ({status}): {data}", "FAIL")

        status, _ = self.request("GET", "/api/v1/auth/login", allow_redirects=False)
        loc = self.last_headers.get("Location", "")
        if status in (301, 302, 303, 307) and "api.workos.com" in loc:
            self.passed += 1
            self.log("AuthKit Login Redirect (/api/v1/auth/login)", f"Redirects to WorkOS AuthKit (HTTP {status})", "PASS")
        else:
            self.failed += 1
            self.log("AuthKit Login Redirect (/api/v1/auth/login)", f"Failed ({status}): {loc}", "FAIL")

        for provider in ["google", "microsoft", "github"]:
            status, _ = self.request("GET", f"/api/v1/auth/oauth/{provider}", allow_redirects=False)
            loc = self.last_headers.get("Location", "")
            if status in (301, 302, 303, 307) and "api.workos.com" in loc:
                self.passed += 1
                self.log(f"OAuth Redirect ({provider})", f"Redirects to WorkOS {provider} provider", "PASS")
            else:
                self.failed += 1
                self.log(f"OAuth Redirect ({provider})", f"Failed ({status}): {loc}", "FAIL")

        # 4. Access Control & Unauthenticated State
        status, data = self.request("GET", "/api/v1/auth/me", auth=False)
        if status == 200 and isinstance(data, dict) and data.get("authenticated") is False:
            self.passed += 1
            self.log("Unauthenticated Session Check", "GET /api/v1/auth/me correctly returns authenticated: false", "PASS")
        else:
            self.failed += 1
            self.log("Unauthenticated Session Check", f"Failed ({status}): {data}", "FAIL")

        unauth_status, _ = self.request("GET", "/api/v1/links", auth=False)
        if unauth_status == 401:
            self.passed += 1
            self.log("Unauthenticated Route Block", "GET /api/v1/links blocked (401 Unauthorized)", "PASS")
        else:
            self.failed += 1
            self.log("Unauthenticated Route Block", f"Expected 401, got {unauth_status}", "FAIL")

        # 5. User Sign-Up
        signup_payload = {
            "first_name": self.dummy_first_name,
            "last_name": self.dummy_last_name,
            "email": self.dummy_email,
            "password": self.dummy_password
        }
        status, data = self.request("POST", "/api/v1/auth/signup", signup_payload, auth=False)
        if status in (200, 201) and isinstance(data, dict) and data.get("user"):
            self.passed += 1
            self.token = data.get("token")
            self.user = data.get("user")
            self.log("User Sign-Up (/api/v1/auth/signup)", f"Account created for {self.dummy_email} (ID: {self.user.get('id')})", "PASS")
        else:
            self.failed += 1
            self.log("User Sign-Up (/api/v1/auth/signup)", f"Failed ({status}): {data}", "FAIL")

        # 6. Password Login
        login_payload = {
            "email": self.dummy_email,
            "password": self.dummy_password
        }
        status, data = self.request("POST", "/api/v1/auth/login", login_payload, auth=False)
        if status == 200 and isinstance(data, dict) and data.get("token"):
            self.passed += 1
            self.token = data.get("token")
            self.log("Password Login (/api/v1/auth/login)", "Authenticated successfully and received session token", "PASS")
        else:
            self.failed += 1
            self.log("Password Login (/api/v1/auth/login)", f"Failed ({status}): {data}", "FAIL")

        # 7. Authenticated Session Check (/api/v1/auth/me)
        status, data = self.request("GET", "/api/v1/auth/me")
        if status == 200 and isinstance(data, dict) and data.get("authenticated") is True:
            self.passed += 1
            self.log("Authenticated Session Check", "GET /api/v1/auth/me returns authenticated: true", "PASS")
        else:
            self.failed += 1
            self.log("Authenticated Session Check", f"Failed ({status}): {data}", "FAIL")

        # 8. User Profile Retrieval & Update
        status, data = self.request("GET", "/api/v1/user/profile")
        if status == 200 and isinstance(data, dict) and data.get("user"):
            self.passed += 1
            u = data["user"]
            self.log("Get Profile (/api/v1/user/profile)", f"Retrieved profile for {u.get('name')} ({u.get('email')})", "PASS")
        else:
            self.failed += 1
            self.log("Get Profile (/api/v1/user/profile)", f"Failed ({status}): {data}", "FAIL")

        updated_first = self.dummy_first_name + "Updated"
        update_payload = {
            "first_name": updated_first,
            "last_name": self.dummy_last_name,
            "avatar_url": "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?auto=format&fit=crop&w=120&h=120&q=80"
        }
        status, data = self.request("PUT", "/api/v1/user/profile", update_payload)
        if status == 200 and isinstance(data, dict) and data.get("user", {}).get("first_name") == updated_first:
            self.passed += 1
            self.log("Update Profile (/api/v1/user/profile)", f"Updated profile first name to {updated_first}", "PASS")
        else:
            self.failed += 1
            self.log("Update Profile (/api/v1/user/profile)", f"Failed ({status}): {data}", "FAIL")

        # 9. Password Reset & Direct Password Change
        status, data = self.request("POST", "/api/v1/auth/password-reset", {"email": self.dummy_email}, auth=False)
        if status == 200 and isinstance(data, dict):
            self.passed += 1
            reset_url = data.get("password_reset_url", "Email Dispatched")
            self.log("Password Reset (/api/v1/auth/password-reset)", f"Generated WorkOS reset link: {reset_url[:35]}...", "PASS")
        else:
            self.failed += 1
            self.log("Password Reset (/api/v1/auth/password-reset)", f"Failed ({status}): {data}", "FAIL")

        status, data = self.request("POST", "/api/v1/user/change-password", {"new_password": self.dummy_new_password})
        if status == 200:
            self.passed += 1
            self.log("Change Password (/api/v1/user/change-password)", "Updated account password via WorkOS", "PASS")
        else:
            self.failed += 1
            self.log("Change Password (/api/v1/user/change-password)", f"Failed ({status}): {data}", "FAIL")

        status, data = self.request("POST", "/api/v1/auth/login", {"email": self.dummy_email, "password": self.dummy_new_password}, auth=False)
        if status == 200 and isinstance(data, dict) and data.get("token"):
            self.passed += 1
            self.token = data.get("token")
            self.log("Re-login with New Password", "Successfully re-authenticated with updated credentials", "PASS")
        else:
            self.failed += 1
            self.log("Re-login with New Password", f"Failed ({status}): {data}", "FAIL")

        # 10. Developer API Keys CRUD & Verification
        key_payload = {"name": f"Harness-Key-{int(time.time())}", "expires_in_days": 30}
        status, data = self.request("POST", "/api/v1/api-keys", key_payload)
        if status in (200, 201) and isinstance(data, dict) and data.get("api_key", {}).get("key"):
            key_obj = data["api_key"]
            self.api_key = key_obj["key"]
            self.key_id = key_obj["id"]
            self.passed += 1
            self.log("Create API Key (/api/v1/api-keys)", f"Generated API Key: {key_obj['key_prefix']} (ID: {self.key_id})", "PASS")
        else:
            self.failed += 1
            self.log("Create API Key (/api/v1/api-keys)", f"Failed ({status}): {data}", "FAIL")

        status, data = self.request("GET", "/api/v1/api-keys")
        if status == 200 and isinstance(data, dict) and any(k.get("id") == self.key_id for k in data.get("api_keys", [])):
            self.passed += 1
            self.log("List API Keys (/api/v1/api-keys)", f"Verified API key exists in team key list ({len(data['api_keys'])} keys)", "PASS")
        else:
            self.failed += 1
            self.log("List API Keys (/api/v1/api-keys)", f"Failed ({status}): {data}", "FAIL")

        # API Key Bearer Authentication
        if self.api_key:
            auth_header = {"Authorization": f"Bearer {self.api_key}"}
            status, data = self.request("GET", "/api/v1/links", headers=auth_header, auth=False)
            if status == 200:
                self.passed += 1
                self.log("API Key Authentication", f"Authenticated /api/v1/links via Bearer {self.api_key[:12]}...", "PASS")
            else:
                self.failed += 1
                self.log("API Key Authentication", f"Failed with API key ({status}): {data}", "FAIL")

            # Revoke Key
            status, _ = self.request("DELETE", f"/api/v1/api-keys/{self.key_id}")
            if status == 200:
                self.passed += 1
                self.log("Revoke API Key (/api/v1/api-keys/:id)", f"Revoked API key {self.key_id}", "PASS")
            else:
                self.failed += 1
                self.log("Revoke API Key (/api/v1/api-keys/:id)", f"Failed ({status})", "FAIL")

            # Verify Revoked Key Rejection
            status, _ = self.request("GET", "/api/v1/links", headers=auth_header, auth=False)
            if status == 401:
                self.passed += 1
                self.log("Revoked Key Rejection", "Revoked API key rejected with 401 Unauthorized", "PASS")
            else:
                self.failed += 1
                self.log("Revoked Key Rejection", f"Expected 401 for revoked key, got {status}", "FAIL")

        # 11. Short Links Feature (Full CRUD + Redirection)
        test_slug = f"qa-link-{int(time.time())}"
        link_payload = {
            "slug": test_slug,
            "custom_slug": test_slug,
            "target_url": "https://lynqr.app",
            "title": "Harness Short Link",
            "description": "Generated during API test drive"
        }
        status, data = self.request("POST", "/api/v1/links", link_payload)
        link_id = None
        actual_slug = test_slug
        if status in (200, 201) and isinstance(data, dict):
            link_id = data.get("id")
            actual_slug = data.get("slug", test_slug)
            self.passed += 1
            self.log("Create Short Link (/api/v1/links)", f"Created short link /s/{actual_slug} (ID: {link_id})", "PASS")
        else:
            self.failed += 1
            self.log("Create Short Link (/api/v1/links)", f"Failed ({status}): {data}", "FAIL")

        status, data = self.request("GET", "/api/v1/links")
        if status == 200 and isinstance(data, dict) and any(l.get("id") == link_id for l in data.get("data", [])):
            self.passed += 1
            self.log("List Short Links (/api/v1/links)", f"Verified link present in team list ({len(data['data'])} total)", "PASS")
        else:
            self.failed += 1
            self.log("List Short Links (/api/v1/links)", f"Failed ({status}): {data}", "FAIL")

        if link_id:
            up_link_payload = {
                "target_url": "https://lynqr.app/updated-destination",
                "title": "Harness Short Link Updated"
            }
            status, data = self.request("PUT", f"/api/v1/links/{link_id}", up_link_payload)
            if status == 200 and isinstance(data, dict) and data.get("target_url") == "https://lynqr.app/updated-destination":
                self.passed += 1
                self.log("Update Short Link (/api/v1/links/:id)", "Updated short link destination and title", "PASS")
            else:
                self.failed += 1
                self.log("Update Short Link (/api/v1/links/:id)", f"Failed ({status}): {data}", "FAIL")

            # Public Redirection
            status, _ = self.request("GET", f"/s/{actual_slug}", auth=False, allow_redirects=False)
            loc = self.last_headers.get("Location", "")
            if status in (301, 302, 307, 308) and "lynqr.app/updated-destination" in loc:
                self.passed += 1
                self.log("Short Link Redirection (/s/:slug)", f"Verified public 302 redirect to {loc}", "PASS")
            else:
                self.failed += 1
                self.log("Short Link Redirection (/s/:slug)", f"Failed ({status}): {loc}", "FAIL")

            # Delete Link
            status, _ = self.request("DELETE", f"/api/v1/links/{link_id}")
            if status in (200, 204):
                self.passed += 1
                self.log("Delete Short Link (/api/v1/links/:id)", f"Deleted short link {link_id}", "PASS")
            else:
                self.failed += 1
                self.log("Delete Short Link (/api/v1/links/:id)", f"Failed ({status})", "FAIL")

        # 12. QR Codes Feature (Full CRUD, Scan Redirection, & All 4 Export Formats)
        qr_payload = {
            "type": "website",
            "title": "Harness QR Code",
            "target_url": "https://lynqr.app/qr-target",
            "payload": "https://lynqr.app/qr-target",
            "is_dynamic": True,
            "style": {"fg_color": "#10b981", "bg_color": "#030712"}
        }
        status, data = self.request("POST", "/api/v1/qr", data=qr_payload)
        qr_id = None
        if status in (200, 201) and isinstance(data, dict):
            qr_id = data.get("id")
            self.passed += 1
            self.log("Create QR Code (/api/v1/qr)", f"Created dynamic QR code (ID: {qr_id})", "PASS")
        else:
            self.failed += 1
            self.log("Create QR Code (/api/v1/qr)", f"Failed ({status}): {data}", "FAIL")

        status, data = self.request("GET", "/api/v1/qr")
        if status == 200 and isinstance(data, dict) and any(q.get("id") == qr_id for q in data.get("data", [])):
            self.passed += 1
            self.log("List QR Codes (/api/v1/qr)", f"Verified QR code present in team list ({len(data['data'])} total)", "PASS")
        else:
            self.failed += 1
            self.log("List QR Codes (/api/v1/qr)", f"Failed ({status}): {data}", "FAIL")

        if qr_id:
            status, data = self.request("GET", f"/api/v1/qr/{qr_id}")
            if status == 200 and isinstance(data, dict) and data.get("id") == qr_id:
                self.passed += 1
                self.log("Get QR Code (/api/v1/qr/:id)", f"Retrieved QR details for {data.get('title')}", "PASS")
            else:
                self.failed += 1
                self.log("Get QR Code (/api/v1/qr/:id)", f"Failed ({status}): {data}", "FAIL")

            up_qr_payload = {
                "title": "Harness QR Code Updated",
                "target_url": "https://lynqr.app/qr-target-v2",
                "style": {"fg_color": "#3b82f6", "bg_color": "#ffffff"}
            }
            status, data = self.request("PUT", f"/api/v1/qr/{qr_id}", data=up_qr_payload)
            if status == 200 and isinstance(data, dict) and data.get("title") == "Harness QR Code Updated":
                self.passed += 1
                self.log("Update QR Code (/api/v1/qr/:id)", "Updated QR code styling and destination", "PASS")
            else:
                self.failed += 1
                self.log("Update QR Code (/api/v1/qr/:id)", f"Failed ({status}): {data}", "FAIL")

            # Public Scan Redirection
            status, _ = self.request("GET", f"/qr/r/{qr_id}", auth=False, allow_redirects=False)
            if status in (301, 302, 307, 308):
                self.passed += 1
                self.log("QR Scan Redirection (/qr/r/:id)", f"Scan redirector returned HTTP {status}", "PASS")
            else:
                self.failed += 1
                self.log("QR Scan Redirection (/qr/r/:id)", f"Failed ({status})", "FAIL")

            # Public Export Formats (Unauthenticated for <img> and direct downloads)
            formats = [("png", "image/png"), ("jpeg", "image/jpeg"), ("svg", "image/svg+xml"), ("pdf", "application/pdf")]
            for fmt, expected_mime in formats:
                status, body = self.request("GET", f"/api/v1/qr/{qr_id}/export?format={fmt}&size=240", auth=False)
                mime = self.last_headers.get("Content-Type", "")
                if status == 200 and expected_mime in mime and len(body) > 0:
                    self.passed += 1
                    self.log(f"Public QR Export ({fmt.upper()})", f"Exported {fmt.upper()} preview without auth ({len(body)} bytes, {mime})", "PASS")
                else:
                    self.failed += 1
                    self.log(f"Public QR Export ({fmt.upper()})", f"Failed ({status}): {mime}", "FAIL")

            # Delete QR Code
            status, _ = self.request("DELETE", f"/api/v1/qr/{qr_id}")
            if status in (200, 204):
                self.passed += 1
                self.log("Delete QR Code (/api/v1/qr/:id)", f"Deleted QR code {qr_id}", "PASS")
            else:
                self.failed += 1
                self.log("Delete QR Code (/api/v1/qr/:id)", f"Failed ({status})", "FAIL")

        # 13. Documents Management & Storage (Upload, List, Update, Download)
        doc_bytes = b"%PDF-1.4 Mock Linksy Document Content for Test Drive"
        files = {"file": ("test_doc.pdf", doc_bytes, "application/pdf")}
        status, data = self.request("POST", "/api/v1/documents/upload", files=files)
        doc_id = None
        doc_url = None
        if status in (200, 201) and isinstance(data, dict):
            doc = data.get("document", {})
            doc_id = doc.get("id")
            doc_url = data.get("download_url")
            self.passed += 1
            self.log("Upload Document (/api/v1/documents/upload)", f"Uploaded document {doc.get('filename')} (ID: {doc_id})", "PASS")
        else:
            self.failed += 1
            self.log("Upload Document (/api/v1/documents/upload)", f"Failed ({status}): {data}", "FAIL")

        status, data = self.request("GET", "/api/v1/documents")
        if status == 200 and isinstance(data, dict) and any(d.get("id") == doc_id for d in data.get("data", [])):
            self.passed += 1
            self.log("List Documents (/api/v1/documents)", f"Verified document in list ({len(data['data'])} documents)", "PASS")
        else:
            self.failed += 1
            self.log("List Documents (/api/v1/documents)", f"Failed ({status}): {data}", "FAIL")

        if doc_id and doc_url:
            # Public Download
            status, body = self.request("GET", doc_url, auth=False)
            if status == 200 and (body == doc_bytes or doc_bytes in (body if isinstance(body, bytes) else body.encode())):
                self.passed += 1
                self.log("Download Document File", f"Verified public download of {len(body)} bytes", "PASS")
            else:
                self.failed += 1
                self.log("Download Document File", f"Download failed ({status})", "FAIL")

            # Update Document File
            doc_bytes_v2 = b"%PDF-1.4 Mock Linksy Document Content Version 2"
            files_v2 = {"file": ("test_doc_v2.pdf", doc_bytes_v2, "application/pdf")}
            status, data = self.request("PUT", f"/api/v1/documents/{doc_id}", files=files_v2)
            if status == 200 and isinstance(data, dict) and data.get("document", {}).get("filename") == "test_doc_v2.pdf":
                self.passed += 1
                self.log("Update Document File (/api/v1/documents/:id)", "Updated document file to version 2", "PASS")
            else:
                self.failed += 1
                self.log("Update Document File (/api/v1/documents/:id)", f"Failed ({status}): {data}", "FAIL")

        # 14. Analytics Summary
        status, data = self.request("GET", "/api/v1/analytics/summary")
        if status == 200 and isinstance(data, dict) and "total_clicks" in data and "total_scans" in data:
            self.passed += 1
            self.log("Analytics Summary (/api/v1/analytics/summary)", f"Retrieved metrics: {data.get('total_clicks')} clicks, {data.get('total_scans')} scans", "PASS")
        else:
            self.failed += 1
            self.log("Analytics Summary (/api/v1/analytics/summary)", f"Failed ({status}): {data}", "FAIL")

        # 15. User Logout & Session Cleared
        status, data = self.request("POST", "/api/v1/auth/logout")
        if status == 200 and isinstance(data, dict) and data.get("success") is True:
            self.passed += 1
            self.log("User Logout (/api/v1/auth/logout)", "Logged out successfully and expired session cookie", "PASS")
        else:
            self.failed += 1
            self.log("User Logout (/api/v1/auth/logout)", f"Failed ({status}): {data}", "FAIL")

        # Post-Logout Verification
        self.token = None
        status, data = self.request("GET", "/api/v1/auth/me", auth=False)
        if status == 200 and isinstance(data, dict) and data.get("authenticated") is False:
            self.passed += 1
            self.log("Post-Logout Session Check", "Session confirmed unauthenticated (authenticated: false)", "PASS")
        else:
            self.failed += 1
            self.log("Post-Logout Session Check", f"Failed ({status}): {data}", "FAIL")

        unauth_status, _ = self.request("GET", "/api/v1/links", auth=False)
        if unauth_status == 401:
            self.passed += 1
            self.log("Post-Logout Route Block", "Protected route rejected with 401 Unauthorized after logout", "PASS")
        else:
            self.failed += 1
            self.log("Post-Logout Route Block", f"Expected 401, got {unauth_status}", "FAIL")

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
