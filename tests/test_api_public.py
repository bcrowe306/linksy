import os
import sys
import time
import random
import string
import pytest
import requests

# Add parent directory to sys.path to import integrations
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from scripts.integrations import WorkOSClient

BASE_URL = os.getenv("TEST_BASE_URL", "https://test.lynqr.app").rstrip("/")

USER_AGENT = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Linksy-TestDrive/2.0"

# Shared session state across test methods
STATE = {
    "user_id": None,
    "email": None,
    "password": "LinksySecurePass!2026",
    "new_password": "LinksyUpdatedPass#2026",
    "token": None,
    "api_key": None,
    "key_id": None,
    "link_id": None,
    "slug": None,
    "qr_id": None,
    "doc_id": None,
    "doc_url": None,
}

@pytest.fixture(scope="session")
def session():
    s = requests.Session()
    s.headers.update({"User-Agent": USER_AGENT})
    return s

def auth_headers():
    return {"Authorization": f"Bearer {STATE['token']}"} if STATE.get("token") else {}

# ==============================================================================
# 1. PUBLIC SYSTEM & OPENAPI DISCOVERY ENDPOINTS
# ==============================================================================
def test_system_health(session):
    """Verify GET /health and GET /api/v1/health return healthy status"""
    r1 = session.get(f"{BASE_URL}/health", timeout=10)
    assert r1.status_code == 200, f"GET /health failed: {r1.status_code}"
    data1 = r1.json()
    assert data1.get("status") in ("healthy", "ok"), f"Unexpected status: {data1}"

    r2 = session.get(f"{BASE_URL}/api/v1/health", timeout=10)
    assert r2.status_code == 200, f"GET /api/v1/health failed: {r2.status_code}"
    data2 = r2.json()
    assert data2.get("status") in ("healthy", "ok"), f"Unexpected status: {data2}"

def test_api_documentation_and_openapi_spec(session):
    """Verify GET /api/docs (HTML) and GET /api/v1/openapi.json (JSON spec)"""
    r_docs = session.get(f"{BASE_URL}/api/docs", timeout=10)
    assert r_docs.status_code == 200, f"GET /api/docs failed: {r_docs.status_code}"
    assert "html" in r_docs.headers.get("Content-Type", "").lower()
    assert len(r_docs.text) > 100

    r_spec = session.get(f"{BASE_URL}/api/v1/openapi.json", timeout=10)
    assert r_spec.status_code == 200, f"GET /api/v1/openapi.json failed: {r_spec.status_code}"
    spec = r_spec.json()
    assert "paths" in spec and "openapi" in spec, f"Invalid OpenAPI spec: {spec.keys()}"

# ==============================================================================
# 2. PUBLIC AUTH DISCOVERY & OAUTH PROVIDER REDIRECTS
# ==============================================================================
def test_auth_config_and_oauth_providers(session):
    """Verify GET /api/v1/auth/config and provider redirect endpoints"""
    r_cfg = session.get(f"{BASE_URL}/api/v1/auth/config", timeout=10)
    assert r_cfg.status_code == 200, f"GET /api/v1/auth/config failed: {r_cfg.status_code}"
    cfg = r_cfg.json()
    assert cfg.get("enabled") is True
    assert "providers" in cfg and cfg.get("dev_login") is False, "dev login must never be enabled on a deployed host"

    # Verify Hosted AuthKit redirect
    r_login = session.get(f"{BASE_URL}/api/v1/auth/login", allow_redirects=False, timeout=10)
    assert r_login.status_code in (301, 302, 303, 307), f"Expected redirect on /auth/login, got {r_login.status_code}"
    loc = r_login.headers.get("Location", "")
    assert "api.workos.com" in loc, f"Redirect target unexpected: {loc}"

    # Verify provider-specific OAuth redirects
    for provider in ["google", "microsoft", "github"]:
        r_oauth = session.get(f"{BASE_URL}/api/v1/auth/oauth/{provider}", allow_redirects=False, timeout=10)
        assert r_oauth.status_code in (301, 302, 303, 307), f"Expected redirect on /auth/oauth/{provider}, got {r_oauth.status_code}"
        assert "api.workos.com" in r_oauth.headers.get("Location", "")

# ==============================================================================
# 3. ROUTE AUTHENTICATION ACCESS CONTROLS (UNAUTHENTICATED)
# ==============================================================================
def test_unauthenticated_access_controls(session):
    """Verify unauthenticated state and that protected endpoints reject with 401"""
    # 1. /api/v1/auth/me should indicate unauthenticated
    r_me = session.get(f"{BASE_URL}/api/v1/auth/me", timeout=10)
    assert r_me.status_code == 200
    assert r_me.json().get("authenticated") is False

    # 2. Protected endpoints should return 401 Unauthorized
    endpoints = [
        "/api/v1/links",
        "/api/v1/qr",
        "/api/v1/user/profile",
        "/api/v1/api-keys",
        "/api/v1/documents",
        "/api/v1/analytics/summary",
    ]
    for ep in endpoints:
        r = session.get(f"{BASE_URL}{ep}", timeout=10)
        assert r.status_code == 401, f"Expected 401 on unauthenticated {ep}, got {r.status_code}"

# ==============================================================================
# 4. USER SIGN-UP, AUTHENTICATION, AND PROFILE MANAGEMENT
# ==============================================================================
def test_user_signup_and_login(session):
    """Verify POST /api/v1/auth/signup, /login, and /auth/me authenticated state"""
    rand_suffix = "".join(random.choices(string.ascii_lowercase + string.digits, k=6))
    STATE["email"] = f"api_test_{int(time.time())}_{rand_suffix}@lynqr.app"

    # 1. Sign-up endpoint contract (WorkOS may require an emailed verification code)
    probe = requests.Session()
    r_signup = probe.post(f"{BASE_URL}/api/v1/auth/signup", json={
        "first_name": "API", "last_name": f"Probe_{rand_suffix}",
        "email": STATE["email"].replace("api_test_", "api_probe_"), "password": STATE["password"],
    }, timeout=15)
    assert r_signup.status_code in (200, 202), f"Signup failed ({r_signup.status_code}): {r_signup.text}"
    assert "token" not in r_signup.json(), "Signup must not return a raw session token"
    probe_user = WorkOSClient().find_user_by_email(STATE["email"].replace("api_test_", "api_probe_"))
    if probe_user:
        WorkOSClient().delete_user(probe_user["id"])

    # 2. Pre-verified fixture user + password login (session arrives as an HttpOnly cookie)
    created = WorkOSClient().create_user(STATE["email"], STATE["password"], "API", f"Tester_{rand_suffix}")
    STATE["user_id"] = created.get("id")
    assert STATE["user_id"], f"WorkOS fixture creation failed: {created}"
    r_login = session.post(f"{BASE_URL}/api/v1/auth/login", json={"email": STATE["email"], "password": STATE["password"]}, timeout=15)
    assert r_login.status_code == 200, f"Login failed ({r_login.status_code}): {r_login.text}"
    STATE["token"] = session.cookies.get("lynqr_session")
    assert STATE["token"] and STATE["token"].startswith("lqs_"), "Missing HttpOnly session cookie"

    # 3. Authenticated /auth/me check
    r_me = session.get(f"{BASE_URL}/api/v1/auth/me", headers=auth_headers(), timeout=10)
    assert r_me.status_code == 200
    assert r_me.json().get("authenticated") is True

def test_user_profile_and_password_lifecycle(session):
    """Verify GET/PUT /api/v1/user/profile, password reset, and change password"""
    # 1. Get Profile
    r_prof = session.get(f"{BASE_URL}/api/v1/user/profile", headers=auth_headers(), timeout=10)
    assert r_prof.status_code == 200, f"Get profile failed: {r_prof.text}"

    # 2. Update Profile
    update_payload = {
        "first_name": "API_Updated",
        "last_name": "Tester_Updated",
        "avatar_url": "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?auto=format&fit=crop&w=120&h=120&q=80",
    }
    r_up = session.put(f"{BASE_URL}/api/v1/user/profile", json=update_payload, headers=auth_headers(), timeout=10)
    assert r_up.status_code == 200, f"Update profile failed: {r_up.text}"
    assert r_up.json().get("user", {}).get("first_name") == "API_Updated"

    # 3. Request Password Reset
    r_reset = session.post(f"{BASE_URL}/api/v1/auth/password-reset", json={"email": STATE["email"]}, timeout=10)
    assert r_reset.status_code == 200, f"Password reset request failed: {r_reset.text}"
    assert "password_reset_url" not in r_reset.json(), "Reset URL must never be returned to the caller"

    # 4. Change Password via User endpoint
    r_change = session.post(
        f"{BASE_URL}/api/v1/user/change-password",
        json={"current_password": STATE["password"], "new_password": STATE["new_password"]},
        headers=auth_headers(),
        timeout=10,
    )
    assert r_change.status_code == 200, f"Change password failed: {r_change.text}"

    # 5. Re-authenticate with New Password
    r_relogin = session.post(
        f"{BASE_URL}/api/v1/auth/login",
        json={"email": STATE["email"], "password": STATE["new_password"]},
        timeout=15,
    )
    assert r_relogin.status_code == 200, f"Re-login failed with new password: {r_relogin.text}"
    STATE["token"] = session.cookies.get("lynqr_session")

# ==============================================================================
# 5. DEVELOPER API KEYS FULL CRUD & AUTHENTICATION
# ==============================================================================
def test_developer_api_keys(session):
    """Verify POST, GET, DELETE /api/v1/api-keys and Bearer API Key authentication"""
    # 1. Create API Key
    key_name = f"Pytest-Key-{int(time.time())}"
    r_create = session.post(
        f"{BASE_URL}/api/v1/api-keys",
        json={"name": key_name, "expires_in_days": 30},
        headers=auth_headers(),
        timeout=10,
    )
    assert r_create.status_code in (200, 201), f"Create API key failed: {r_create.text}"
    key_data = r_create.json().get("api_key", {})
    STATE["api_key"] = key_data.get("key")
    STATE["key_id"] = key_data.get("id")
    assert STATE["api_key"] and STATE["api_key"].startswith("lk_live_")

    # 2. List API Keys
    r_list = session.get(f"{BASE_URL}/api/v1/api-keys", headers=auth_headers(), timeout=10)
    assert r_list.status_code == 200
    keys = r_list.json().get("api_keys", [])
    assert any(k.get("id") == STATE["key_id"] for k in keys), "Created API key not found in list"

    # 3. Authenticate with Bearer API Key
    r_api_auth = session.get(f"{BASE_URL}/api/v1/links", headers={"Authorization": f"Bearer {STATE['api_key']}"}, timeout=10)
    assert r_api_auth.status_code == 200, f"Bearer API key authentication failed: {r_api_auth.status_code}"

    # 4. Revoke API Key
    r_del = session.delete(f"{BASE_URL}/api/v1/api-keys/{STATE['key_id']}", headers=auth_headers(), timeout=10)
    assert r_del.status_code == 200, f"Revoke API key failed: {r_del.text}"

    # 5. Verify Revoked Key is Rejected
    r_revoked = session.get(f"{BASE_URL}/api/v1/links", headers={"Authorization": f"Bearer {STATE['api_key']}"}, timeout=10)
    assert r_revoked.status_code == 401, f"Expected 401 for revoked key, got {r_revoked.status_code}"

# ==============================================================================
# 6. SHORT LINKS FULL CRUD & REDIRECTION
# ==============================================================================
def test_short_links_crud_and_redirect(session):
    """Verify POST, GET, PUT, DELETE /api/v1/links and GET /s/:slug redirection"""
    slug = f"qa-link-{int(time.time())}"
    STATE["slug"] = slug

    # 1. Create Link
    create_payload = {
        "slug": slug,
        "custom_slug": slug,
        "target_url": "https://lynqr.app",
        "title": "QA Test Link",
        "description": "Created during public API automated test drive",
    }
    r_create = session.post(f"{BASE_URL}/api/v1/links", json=create_payload, headers=auth_headers(), timeout=10)
    assert r_create.status_code in (200, 201), f"Create link failed: {r_create.text}"
    link = r_create.json()
    STATE["link_id"] = link.get("id")
    actual_slug = link.get("slug", slug)
    STATE["slug"] = actual_slug
    assert STATE["link_id"], "Missing link ID"

    # 2. List Links
    r_list = session.get(f"{BASE_URL}/api/v1/links", headers=auth_headers(), timeout=10)
    assert r_list.status_code == 200
    links = r_list.json().get("data", [])
    assert any(l.get("id") == STATE["link_id"] for l in links), "Created link not found in list"

    # 3. Update Link
    update_payload = {
        "target_url": "https://lynqr.app/updated",
        "title": "QA Test Link Updated",
    }
    r_up = session.put(f"{BASE_URL}/api/v1/links/{STATE['link_id']}", json=update_payload, headers=auth_headers(), timeout=10)
    assert r_up.status_code == 200, f"Update link failed: {r_up.text}"
    assert r_up.json().get("target_url") == "https://lynqr.app/updated"

    # 4. Public Short Link Redirection
    r_redir = session.get(f"{BASE_URL}/s/{actual_slug}", allow_redirects=False, timeout=10)
    assert r_redir.status_code in (301, 302, 307, 308), f"Expected redirect, got {r_redir.status_code}"
    loc = r_redir.headers.get("Location", "")
    assert "lynqr.app/updated" in loc, f"Redirect location unexpected: {loc}"

    # 5. Delete Link
    r_del = session.delete(f"{BASE_URL}/api/v1/links/{STATE['link_id']}", headers=auth_headers(), timeout=10)
    assert r_del.status_code in (200, 204), f"Delete link failed: {r_del.status_code}"

# ==============================================================================
# 7. QR CODES FULL CRUD, SCAN REDIRECT, & ALL 4 EXPORT FORMATS
# ==============================================================================
def test_qr_codes_crud_and_exports(session):
    """Verify POST, GET, PUT, DELETE /api/v1/qr, GET /qr/r/:id, and PNG/JPEG/SVG/PDF exports"""
    # 1. Create QR Code
    qr_payload = {
        "type": "website",
        "title": "Pytest Dynamic QR",
        "target_url": "https://lynqr.app/destination",
        "payload": "https://lynqr.app/destination",
        "is_dynamic": True,
        "style": {"fg_color": "#2563eb", "bg_color": "#ffffff"},
    }
    r_create = session.post(f"{BASE_URL}/api/v1/qr", json=qr_payload, headers=auth_headers(), timeout=10)
    assert r_create.status_code in (200, 201), f"Create QR failed: {r_create.text}"
    qr_data = r_create.json()
    STATE["qr_id"] = qr_data.get("id")
    assert STATE["qr_id"], "Missing QR ID"

    # 2. List QR Codes
    r_list = session.get(f"{BASE_URL}/api/v1/qr", headers=auth_headers(), timeout=10)
    assert r_list.status_code == 200
    qrs = r_list.json().get("data", [])
    assert any(q.get("id") == STATE["qr_id"] for q in qrs), "Created QR not found in list"

    # 3. Get Specific QR Code
    r_get = session.get(f"{BASE_URL}/api/v1/qr/{STATE['qr_id']}", headers=auth_headers(), timeout=10)
    assert r_get.status_code == 200
    assert r_get.json().get("id") == STATE["qr_id"]

    # 4. Update QR Code
    up_qr_payload = {
        "title": "Pytest QR Updated",
        "target_url": "https://lynqr.app/destination-v2",
        "style": {"fg_color": "#10b981", "bg_color": "#030712"},
    }
    r_up = session.put(f"{BASE_URL}/api/v1/qr/{STATE['qr_id']}", json=up_qr_payload, headers=auth_headers(), timeout=10)
    assert r_up.status_code == 200, f"Update QR failed: {r_up.text}"

    # 5. Public QR Scan Redirector
    r_scan = session.get(f"{BASE_URL}/qr/r/{STATE['qr_id']}", allow_redirects=False, timeout=10)
    assert r_scan.status_code in (301, 302, 307, 308), f"Expected scan redirect, got {r_scan.status_code}"

    # 6. Public QR Image & Document Exports (Unauthenticated <img> compatibility)
    formats = [
        ("png", "image/png"),
        ("jpeg", "image/jpeg"),
        ("svg", "image/svg+xml"),
        ("pdf", "application/pdf"),
    ]
    for fmt, expected_mime in formats:
        url = f"{BASE_URL}/api/v1/qr/{STATE['qr_id']}/export?format={fmt}&size=256"
        # Test without Authorization header
        r_exp = session.get(url, timeout=10)
        assert r_exp.status_code == 200, f"Export format {fmt} failed with status {r_exp.status_code}"
        assert expected_mime in r_exp.headers.get("Content-Type", ""), f"Unexpected MIME type for {fmt}: {r_exp.headers.get('Content-Type')}"
        assert len(r_exp.content) > 0, f"Empty export body for {fmt}"

    # 7. Delete QR Code
    r_del = session.delete(f"{BASE_URL}/api/v1/qr/{STATE['qr_id']}", headers=auth_headers(), timeout=10)
    assert r_del.status_code in (200, 204), f"Delete QR failed: {r_del.status_code}"

# ==============================================================================
# 8. DOCUMENTS STORAGE, UPLOAD, UPDATE, AND DOWNLOAD
# ==============================================================================
def test_documents_management_and_download(session):
    """Verify POST, GET, PUT /api/v1/documents and GET /api/v1/documents/files/:filename"""
    # 1. Upload Document
    mock_pdf = b"%PDF-1.4 Mock document content for Linksy automated testing"
    files = {"file": ("test_report.pdf", mock_pdf, "application/pdf")}
    r_up = session.post(f"{BASE_URL}/api/v1/documents/upload", files=files, headers=auth_headers(), timeout=15)
    assert r_up.status_code in (200, 201), f"Upload document failed ({r_up.status_code}): {r_up.text}"
    doc_res = r_up.json()
    doc = doc_res.get("document", {})
    STATE["doc_id"] = doc.get("id")
    STATE["doc_url"] = doc_res.get("download_url")
    assert STATE["doc_id"], "Missing document ID"
    assert STATE["doc_url"], "Missing document download URL"

    # 2. List Documents
    r_list = session.get(f"{BASE_URL}/api/v1/documents", headers=auth_headers(), timeout=10)
    assert r_list.status_code == 200
    docs = r_list.json().get("data", [])
    assert any(d.get("id") == STATE["doc_id"] for d in docs), "Uploaded document not in list"

    # 3. Download Document File (Public)
    r_dl = session.get(STATE["doc_url"], timeout=10)
    assert r_dl.status_code == 200, f"Download document failed ({r_dl.status_code})"
    assert r_dl.content == mock_pdf, "Downloaded document content does not match original"

    # 4. Update Document File
    mock_pdf_v2 = b"%PDF-1.4 Updated document version 2 for Linksy automated testing"
    files_v2 = {"file": ("test_report_v2.pdf", mock_pdf_v2, "application/pdf")}
    r_update = session.put(f"{BASE_URL}/api/v1/documents/{STATE['doc_id']}", files=files_v2, headers=auth_headers(), timeout=15)
    assert r_update.status_code == 200, f"Update document failed: {r_update.text}"
    v2_url = r_update.json().get("download_url")

    # 5. Verify Updated Download
    r_dl2 = session.get(v2_url, timeout=10)
    assert r_dl2.status_code == 200
    assert r_dl2.content == mock_pdf_v2

    # 6. Delete Single Document & Confirm DB and File Removal
    r_del = session.delete(f"{BASE_URL}/api/v1/documents/{STATE['doc_id']}?delete_qr=true", headers=auth_headers(), timeout=10)
    assert r_del.status_code == 200, f"Delete document failed ({r_del.status_code}): {r_del.text}"
    del_data = r_del.json()
    assert "deleted_qr_count" in del_data

    # Confirm database entry is gone
    r_check = session.get(f"{BASE_URL}/api/v1/documents", headers=auth_headers(), timeout=10)
    assert r_check.status_code == 200
    assert not any(d.get("id") == STATE["doc_id"] for d in r_check.json().get("data", [])), "Deleted document still in database"

    # Confirm physical file is gone
    r_file_check = session.get(v2_url, headers={"Cache-Control": "no-cache"}, params={"nocache": str(time.time())}, timeout=10)
    assert r_file_check.status_code == 404, f"Expected 404 for deleted file, got {r_file_check.status_code}"

    # 7. Bulk Delete Documents Test
    bulk_f1 = {"file": ("bulk_doc_1.pdf", b"%PDF-1.4 Bulk Doc 1 Content", "application/pdf")}
    bulk_f2 = {"file": ("bulk_doc_2.pdf", b"%PDF-1.4 Bulk Doc 2 Content", "application/pdf")}
    r_b1 = session.post(f"{BASE_URL}/api/v1/documents/upload", files=bulk_f1, headers=auth_headers(), timeout=15)
    r_b2 = session.post(f"{BASE_URL}/api/v1/documents/upload", files=bulk_f2, headers=auth_headers(), timeout=15)
    assert r_b1.status_code in (200, 201) and r_b2.status_code in (200, 201)
    b1_id = r_b1.json().get("document", {}).get("id")
    b2_id = r_b2.json().get("document", {}).get("id")
    b1_url = r_b1.json().get("download_url")

    # Execute Bulk Delete
    bulk_payload = {"document_ids": [b1_id, b2_id], "delete_qr": True}
    r_bulk_del = session.post(f"{BASE_URL}/api/v1/documents/bulk-delete", json=bulk_payload, headers=auth_headers(), timeout=10)
    assert r_bulk_del.status_code == 200, f"Bulk delete failed ({r_bulk_del.status_code}): {r_bulk_del.text}"
    bulk_res = r_bulk_del.json()
    assert bulk_res.get("deleted_docs_count") == 2, f"Expected 2 deleted docs, got {bulk_res}"

    # Confirm both gone from DB
    r_check_bulk = session.get(f"{BASE_URL}/api/v1/documents", headers=auth_headers(), timeout=10)
    docs_after = [d.get("id") for d in r_check_bulk.json().get("data", [])]
    assert b1_id not in docs_after and b2_id not in docs_after, "Bulk deleted documents still in database"
    assert session.get(b1_url, timeout=10).status_code == 404, "Bulk deleted file still served" 

# ==============================================================================
# 9. ANALYTICS SUMMARY
# ==============================================================================
def test_analytics_summary(session):
    """Verify GET /api/v1/analytics/summary returns metrics aggregation"""
    r = session.get(f"{BASE_URL}/api/v1/analytics/summary", headers=auth_headers(), timeout=10)
    assert r.status_code == 200, f"Get analytics summary failed: {r.text}"
    data = r.json()
    assert "total_clicks" in data
    assert "total_scans" in data
    assert "by_country" in data
    assert "by_device" in data
    assert "by_browser" in data

# ==============================================================================
# 10. AUTH LOGOUT & POST-LOGOUT ROUTE BLOCKING
# ==============================================================================


# ==============================================================================
# 10. BRANDED HTML ERROR PAGE FOR INVALID / NON-EXISTENT SCANS
# ==============================================================================
def test_branded_error_page_on_invalid_links_and_qr(session):
    """Verify invalid or expired short links & QR scans return a Lynqr-branded HTML error page"""
    # 1. Invalid / Non-existent short link
    bogus_slug = f"non-existent-{int(time.time())}"
    r_link_err = session.get(
        f"{BASE_URL}/s/{bogus_slug}",
        headers={"Accept": "text/html,application/xhtml+xml"},
        allow_redirects=False,
        timeout=10
    )
    assert r_link_err.status_code == 404, f"Expected 404, got {r_link_err.status_code}"
    assert "Lynqr" in r_link_err.text, "Branded error page missing Lynqr branding"
    assert "Link Not Found or Inactive" in r_link_err.text or "not exist" in r_link_err.text
    assert "Return to Lynqr Home" in r_link_err.text

    # 2. Invalid / Non-existent QR code scan
    bogus_qr_id = f"qr-missing-{int(time.time())}"
    r_qr_err = session.get(
        f"{BASE_URL}/qr/r/{bogus_qr_id}",
        headers={"Accept": "text/html,application/xhtml+xml"},
        allow_redirects=False,
        timeout=10
    )
    assert r_qr_err.status_code == 404, f"Expected 404, got {r_qr_err.status_code}"
    assert "Lynqr" in r_qr_err.text, "Branded error page missing Lynqr branding"
    assert "QR Code Not Found" in r_qr_err.text or "not exist" in r_qr_err.text
    assert "Return to Lynqr Home" in r_qr_err.text


# ==============================================================================
# 11. TEAM & USER MANAGEMENT WITH WORKOS API
# ==============================================================================
def test_team_and_user_management(session):
    """Verify WorkOS user invitations, list members, roles, credential reset, and data isolation"""
    # 1. List Team Members
    r_members = session.get(f"{BASE_URL}/api/v1/team/members", headers=auth_headers(), timeout=10)
    assert r_members.status_code == 200, f"List members failed ({r_members.status_code}): {r_members.text}"
    m_data = r_members.json()
    assert "members" in m_data and "team_id" in m_data
    assert m_data.get("current_user_role") in ("owner", "admin"), f"Expected admin/owner role, got {m_data.get('current_user_role')}"

    # 2. Invite Member via WorkOS
    invite_email = f"qa_invite_{int(time.time())}_{''.join(random.choices(string.ascii_lowercase, k=4))}@lynqr.app"
    r_invite = session.post(
        f"{BASE_URL}/api/v1/team/members/invite",
        json={"email": invite_email, "role": "member"},
        headers=auth_headers(),
        timeout=15
    )
    assert r_invite.status_code in (200, 201), f"Invite member failed ({r_invite.status_code}): {r_invite.text}"
    inv_res = r_invite.json().get("invitation", {})
    inv_id = inv_res.get("id") or inv_res.get("workos_invitation_id")
    assert inv_id, "Missing invitation ID"

    # 3. Revoke Invitation
    if inv_id:
        r_rev = session.delete(f"{BASE_URL}/api/v1/team/invitations/{inv_id}", headers=auth_headers(), timeout=10)
        assert r_rev.status_code == 200, f"Revoke invitation failed: {r_rev.text}"

    # 4. Data Isolation Check (Ensure foreign team cannot access/delete resources)
    test_slug = f"iso-link-{int(time.time())}"
    r_link = session.post(
        f"{BASE_URL}/api/v1/links",
        json={"slug": test_slug, "target_url": "https://lynqr.app", "title": "Isolation Link"},
        headers=auth_headers(),
        timeout=10
    )
    assert r_link.status_code in (200, 201)
    iso_link_id = r_link.json().get("id")

    # Attacker team tries to delete this link using different team header
    r_hacked_del = session.delete(
        f"{BASE_URL}/api/v1/links/{iso_link_id}",
        headers={**auth_headers(), "X-Team-ID": "foreign_team_rogue_999"},
        timeout=10
    )
    assert r_hacked_del.status_code in (401, 403, 404), f"Foreign team should not be able to delete link: {r_hacked_del.status_code}"

    # Clean up with legitimate team credentials
    r_clean = session.delete(f"{BASE_URL}/api/v1/links/{iso_link_id}", headers=auth_headers(), timeout=10)
    assert r_clean.status_code in (200, 204)

def test_auth_logout_and_post_logout_rejection(session):
    """Verify POST /api/v1/auth/logout terminates session and invalidates access"""
    # 1. Logout
    r_logout = session.post(f"{BASE_URL}/api/v1/auth/logout", headers=auth_headers(), timeout=10)
    assert r_logout.status_code == 200, f"Logout failed: {r_logout.text}"
    assert r_logout.json().get("success") is True

    # 2. Clear local token
    STATE["token"] = None

    # 3. GET /auth/me should now report unauthenticated
    r_me = session.get(f"{BASE_URL}/api/v1/auth/me", timeout=10)
    assert r_me.status_code == 200
    assert r_me.json().get("authenticated") is False

    # 4. Protected route should return 401 Unauthorized
    r_prot = session.get(f"{BASE_URL}/api/v1/links", timeout=10)
    assert r_prot.status_code == 401

# ==============================================================================
# 11. WORKOS AUTOMATED CLEANUP
# ==============================================================================
def test_workos_user_cleanup():
    """Verify and clean up the created test user from WorkOS User Management"""
    workos = WorkOSClient()
    if not workos.is_configured():
        pytest.skip("WorkOS API key not configured")

    user_id = STATE.get("user_id")
    if not user_id and STATE.get("email"):
        res = workos.list_users(limit=20)
        for u in res.get("data", []):
            if u.get("email") == STATE["email"]:
                user_id = u.get("id")
                break

    if user_id:
        deleted = workos.delete_user(user_id)
        assert deleted is True, f"Failed to delete test user {user_id} from WorkOS"
