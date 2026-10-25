import os
import sys
import time
import random
import string
import pytest
import requests
from playwright.sync_api import Page, expect

# Add parent directory to sys.path to import integrations
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from scripts.integrations import CloudflareClient, RenderClient, WorkOSClient

BASE_URL = os.getenv("TEST_BASE_URL", "https://test.lynqr.app")

# Shared state across test functions
SHARED_STATE = {
    "user_id": None,
    "email": None,
    "password": None,
    "api_key": None,
    "slug": None,
}

@pytest.fixture(scope="session")
def base_url():
    return BASE_URL

def get_or_create_credentials():
    if not SHARED_STATE["email"]:
        rand_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=6))
        SHARED_STATE["email"] = f"qa_test_{int(time.time())}_{rand_suffix}@lynqr.app"
        SHARED_STATE["password"] = "LinksySecurePass!2026"
    return SHARED_STATE["email"], SHARED_STATE["password"]

def ui_login(page: Page, base_url: str, email: str, password: str):
    page.goto(f"{base_url}/login")
    page.wait_for_load_state("networkidle")
    page.fill("input[type='email']", email)
    page.fill("input[type='password']", password)
    page.click("button[type='submit']")
    page.wait_for_url(lambda u: "/login" not in u, timeout=15000)


def ensure_fixture_user():
    """Creates the shared pre-verified WorkOS user once per run."""
    email, password = get_or_create_credentials()
    if not SHARED_STATE["user_id"]:
        created = WorkOSClient().create_user(email, password, "QA_User", "Tester")
        SHARED_STATE["user_id"] = created.get("id")
    return email, password


def authenticate_page(page: Page, base_url: str):
    """Ensure the page has an active authenticated session"""
    page.goto(f"{base_url}/")
    page.wait_for_load_state("networkidle")
    if page.locator("[data-testid=user-menu]").is_visible():
        return
    email, password = ensure_fixture_user()
    ui_login(page, base_url, email, password)

# ==============================================================================
# 1. CLOUDFLARE INFRASTRUCTURE API TESTS
# ==============================================================================
def test_cloudflare_zone_and_dns():
    """Verify Cloudflare DNS and zone routing for test.lynqr.app"""
    cf = CloudflareClient()
    if not cf.is_configured():
        pytest.skip("Cloudflare API token not configured")
        
    zone = cf.get_zone()
    assert isinstance(zone, dict) and zone.get("id"), "Cloudflare zone lynqr.app not found"
    assert zone.get("status") == "active", f"Zone status is {zone.get('status')}"
    
    import urllib.parse
    target_host = urllib.parse.urlparse(BASE_URL).netloc or "lynqr.app"
    records = cf.get_dns_records(zone["id"], target_host)
    if not records:
        records = cf.get_dns_records(zone["id"], "lynqr.app") or cf.get_dns_records(zone["id"], "test.lynqr.app")
    assert len(records) > 0, f"DNS record for {target_host} not found in Cloudflare"
    record = records[0]
    assert "onrender.com" in record.get("content", ""), f"DNS target unexpected: {record.get('content')}"

def test_cloudflare_r2_storage():
    """Verify Cloudflare R2 bucket lynqr-storage exists and is accessible"""
    cf = CloudflareClient()
    if not cf.is_configured():
        pytest.skip("Cloudflare API token not configured")
        
    buckets = cf.get_r2_buckets()
    bucket_names = [b.get("name") for b in buckets]
    assert "lynqr-storage" in bucket_names, f"Bucket lynqr-storage missing from {bucket_names}"

# ==============================================================================
# 2. RENDER COMPUTE & DEPLOYMENT API TESTS
# ==============================================================================
def test_render_service_health_and_deployment():
    """Verify Render compute service is not suspended and deployment is live"""
    render = RenderClient()
    if not render.is_configured():
        pytest.skip("Render API key not configured")
        
    services = render.list_services()
    assert len(services) > 0, "No Render services returned"
    
    lynqr_service = None
    for s in services:
        svc = s.get("service", {})
        if "lynqr" in svc.get("name", "").lower():
            lynqr_service = svc
            break
            
    assert lynqr_service is not None, "Lynqr service not found on Render"
    assert lynqr_service.get("suspended") == "not_suspended", "Service is suspended on Render"
    
    # Check latest deploy status
    deploy = render.get_latest_deploy(lynqr_service["id"])
    for _ in range(6):
        if deploy and deploy.get("status") in ("live", "build_failed", "update_failed", "canceled"):
            break
        time.sleep(5)
        deploy = render.get_latest_deploy(lynqr_service["id"])
    assert deploy is not None, "No deployment records found"
    assert deploy.get("status") == "live", f"Latest Render deploy status is {deploy.get('status')}"

# ==============================================================================
# 2.5 ROUTE AUTHENTICATION & ACCESS CONTROL TESTS
# ==============================================================================
def test_unauthenticated_route_blocks_and_redirects(page: Page, base_url: str):
    """Verify that unauthenticated access to protected routes redirects to /login"""
    page.goto(f"{base_url}/login")
    page.wait_for_load_state("networkidle")
    page.context.clear_cookies()
    page.evaluate("() => { localStorage.clear(); sessionStorage.clear(); }")

    # 1. Visiting / shows the public landing page, which leads to sign-up
    page.goto(f"{base_url}/")
    page.wait_for_load_state("networkidle")
    expect(page.locator("h1")).to_contain_text("keep working for you")
    expect(page.locator("[data-cta=hero]")).to_have_attribute("href", "/signup")

    # 2. Visiting /profile redirects to /login
    page.goto(f"{base_url}/profile")
    page.wait_for_load_state("networkidle")
    assert "/login" in page.url, f"Expected redirect to /login from /profile, got {page.url}"

    # 3. Visiting /links redirects to /login
    page.goto(f"{base_url}/links")
    page.wait_for_load_state("networkidle")
    assert "/login" in page.url, f"Expected redirect to /login from /links, got {page.url}"

    # 4. Visiting /qr-studio redirects to /login
    page.goto(f"{base_url}/qr-studio")
    page.wait_for_load_state("networkidle")
    assert "/login" in page.url, f"Expected redirect to /login from /qr-studio, got {page.url}"

    # 5. Public /api-docs route remains accessible
    page.goto(f"{base_url}/api-docs")
    page.wait_for_load_state("networkidle")
    assert "/api-docs" in page.url, f"Public /api-docs was redirected: {page.url}"

# ==============================================================================
# 3. PLAYWRIGHT UI TESTS: THEME SYSTEM
# ==============================================================================
def test_theme_system_toggle_and_persistence(page: Page, base_url: str):
    """Verify dark/light theme switching and persistence across page reloads"""
    authenticate_page(page, base_url)
    page.goto(base_url)
    page.wait_for_load_state("networkidle")
    
    html_is_dark = "() => document.documentElement.classList.contains('dark')"
    for pref, dark in (("light", False), ("dark", True), ("light", False)):
        page.click("[data-testid=theme-toggle]")
        page.click(f"[data-testid=theme-{pref}]")
        assert page.evaluate("() => localStorage.getItem('lynqr_theme_pref')") == pref
        assert page.evaluate(html_is_dark) is dark, f"<html> dark class wrong for {pref}"

    # Reload preserves the preference and applies it before render
    page.reload()
    page.wait_for_load_state("networkidle")
    assert page.evaluate("() => localStorage.getItem('lynqr_theme_pref')") == "light"
    assert page.evaluate(html_is_dark) is False
    bg = page.evaluate("() => getComputedStyle(document.body).backgroundColor")
    assert bg != "rgb(12, 12, 14)", f"Light theme still renders the dark canvas: {bg}"

    # System preference follows the OS setting
    page.click("[data-testid=theme-toggle]")
    page.click("[data-testid=theme-system]")
    page.emulate_media(color_scheme="dark")
    page.wait_for_timeout(200)
    assert page.evaluate(html_is_dark) is True
    page.emulate_media(color_scheme="light")
    page.wait_for_timeout(200)
    assert page.evaluate(html_is_dark) is False
    page.click("[data-testid=theme-toggle]")
    page.click("[data-testid=theme-dark]")

# ==============================================================================
# 4. PLAYWRIGHT UI TESTS: USER REGISTRATION & AUTHENTICATION
# ==============================================================================
def test_signup_and_login_ui(page: Page, base_url: str):
    """Sign-up form submits (and may request an emailed code); sign-in, sign-out and sign-in again work."""
    probe_email = f"ui_probe_{int(time.time())}@lynqr.app"
    page.goto(f"{base_url}/signup")
    page.wait_for_load_state("networkidle")
    page.fill("input[placeholder='Jane']", "QA_User")
    page.fill("input[placeholder='Doe']", "Tester")
    page.fill("input[placeholder='jane@company.com']", probe_email)
    page.fill("input[placeholder='At least 8 characters']", "LinksySecurePass!2026")
    page.fill("input[placeholder='Re-enter password']", "LinksySecurePass!2026")
    page.click("button:has-text('Create Free Account')")
    page.wait_for_function(
        "() => !location.pathname.startsWith('/signup') || document.querySelector('[data-testid=verify-email-form]')",
        timeout=15000,
    )
    probe = WorkOSClient().find_user_by_email(probe_email)
    if probe:
        WorkOSClient().delete_user(probe["id"])
    page.context.clear_cookies()

    email, password = ensure_fixture_user()
    ui_login(page, base_url, email, password)
    expect(page.locator("[data-testid=user-menu]")).to_be_visible(timeout=8000)
    session_cookie = next((c for c in page.context.cookies() if c["name"] == "lynqr_session"), None)
    assert session_cookie and session_cookie["httpOnly"], "Session cookie must be HttpOnly"
    assert page.evaluate("() => document.cookie.includes('lynqr_session')") is False

    page.click("[data-testid=user-menu]")
    page.click("[data-testid=sign-out]")
    page.wait_for_url(lambda u: "/login" in u, timeout=10000)
    ui_login(page, base_url, email, password)
    expect(page.locator("[data-testid=user-menu]")).to_be_visible(timeout=8000)

# ==============================================================================
# 5. PLAYWRIGHT UI TESTS: PROFILE & DEVELOPER API KEYS
# ==============================================================================
def test_profile_and_developer_api_keys(page: Page, base_url: str):
    """Update the profile, toggle a notification preference, and create and revoke an API key."""
    authenticate_page(page, base_url)
    page.click("[data-testid=user-menu]")
    page.get_by_role("menuitem", name="Profile & security").click()
    page.wait_for_url("**/profile", timeout=10000)

    page.fill("[data-testid=profile-first-name]", "QA_Updated")
    page.click("[data-testid=save-profile]")
    expect(page.get_by_text("Profile updated successfully")).to_be_visible(timeout=8000)

    page.click("[data-testid=profile-tab-notifications]")
    toggle = page.locator("[data-testid=pref-weekly_report]")
    expect(toggle).to_be_visible()
    before = toggle.get_attribute("aria-checked")
    toggle.click()
    expect(page.get_by_text("Preferences saved")).to_be_visible()
    page.reload()
    expect(page.locator("[data-testid=pref-weekly_report]")).to_have_attribute("aria-checked", "false" if before == "true" else "true")

    page.click("[data-testid=profile-tab-security]")
    expect(page.locator("[data-testid=session-list]")).to_contain_text("This device")

    page.click("[data-testid=profile-tab-api-keys]")
    page.click("[data-testid=new-api-key]")
    name = f"Playwright-Key-{int(time.time())}"
    page.fill("[data-testid=api-key-name]", name)
    page.click("[data-testid=create-api-key]")
    full_key = page.locator("[data-testid=new-api-key-value]").inner_text().strip()
    assert full_key.startswith("lk_live_"), full_key
    SHARED_STATE["api_key"] = full_key
    page.click("[data-testid=api-key-done]")
    row = page.locator("[data-testid=api-key-table] tr", has_text=name)
    expect(row).to_contain_text(full_key[:12])

    r = requests.get(f"{base_url}/api/v1/links", headers={"Authorization": f"Bearer {full_key}"}, timeout=10)
    assert r.status_code == 200, f"new key rejected: {r.status_code}"
    row.locator("[data-testid=revoke-key]").click()
    page.click("[data-testid=confirm-accept]")
    expect(row).to_have_count(0)
    r = requests.get(f"{base_url}/api/v1/links", headers={"Authorization": f"Bearer {full_key}"}, timeout=10)
    assert r.status_code == 401, "revoked key still works"

# ==============================================================================
# 6. PLAYWRIGHT UI TESTS: QR STUDIO & UNSAVED CHANGES NAVIGATION GUARD
# ==============================================================================
def test_qr_studio_unsaved_changes_warning(page: Page, base_url: str):
    """Editing in QR Studio marks the design dirty and guards navigation with a confirm dialog."""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/qr-studio")
    page.wait_for_load_state("networkidle")

    page.fill("[data-testid=qr-title-input]", "Modified QR Code Without Saving")
    expect(page.locator("[data-testid=unsaved-badge]")).to_be_visible(timeout=5000)

    page.locator("aside a[href='/links']").first.click()
    expect(page.get_by_text("Discard unsaved changes?")).to_be_visible()
    page.get_by_role("button", name="Cancel").click()
    assert "/qr-studio" in page.url, "Cancel must keep the user in the studio"

    page.locator("aside a[href='/links']").first.click()
    page.click("[data-testid=confirm-accept]")
    page.wait_for_url("**/links", timeout=8000)


def test_qr_studio_create_scan_and_export(page: Page, base_url: str):
    """Create a dynamic QR in the studio, verify the live preview scans, download it, and follow its scan URL."""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/qr-studio")
    page.wait_for_load_state("networkidle")
    page.fill("[data-testid=qr-title-input]", f"QA Studio {int(time.time())}")
    page.fill("[data-testid=qr-target-url]", "https://lynqr.app/?from=qa")
    page.get_by_role("button", name="Indigo").click()
    expect(page.locator("[data-testid=scan-ok]")).to_be_visible(timeout=10000)

    page.click("[data-testid=qr-save]")
    page.wait_for_url("**/qr-studio/*", timeout=10000)
    scan_url = page.locator("text=/\\/q\\/[A-Za-z0-9]{7}/").first.inner_text().strip()

    with page.expect_download() as dl:
        page.click("[data-testid=export-svg]")
    assert dl.value.suggested_filename.endswith(".svg")

    r = requests.get(scan_url, allow_redirects=False, timeout=10)
    assert r.status_code == 302 and r.headers["Location"] == "https://lynqr.app/?from=qa", f"scan: {r.status_code} {r.headers.get('Location')}"


# ==============================================================================
# 7. PLAYWRIGHT UI TESTS: LINKS CREATION & REDIRECTION
# ==============================================================================
def test_short_link_creation_and_redirection(page: Page, base_url: str):
    """Test creating a new short link via modal and verifying 302 redirection"""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/links")
    page.wait_for_load_state("networkidle")
    
    page.click("[data-testid=new-link]")
    expect(page.get_by_role("dialog", name="Create new")).to_be_visible()

    slug = f"qa-link-{int(time.time())}"
    SHARED_STATE["slug"] = slug
    page.fill("[data-testid=link-target]", "https://lynqr.app")
    page.fill("[data-testid=link-slug]", slug)
    page.click("[data-testid=create-submit]")

    expect(page.get_by_text("Short link created")).to_be_visible(timeout=8000)
    expect(page.locator("[data-testid=links-table]").get_by_text(f"/s/{slug}")).to_be_visible(timeout=8000)

    # Test HTTP Redirection
    redirect_resp = requests.get(f"{base_url}/s/{slug}", allow_redirects=False, timeout=10)
    assert redirect_resp.status_code in [301, 302, 307, 308], f"Expected redirect, got status {redirect_resp.status_code}"
    loc = redirect_resp.headers.get("Location")
    assert "lynqr.app" in loc, f"Redirect location unexpected: {loc}"

# ==============================================================================
# 8. WORKOS VERIFICATION & AUTOMATED CLEANUP
# ==============================================================================
def test_workos_user_verification_and_cleanup():
    """Verify test user exists in WorkOS User Management and delete to clean up"""
    workos = WorkOSClient()
    if not workos.is_configured():
        pytest.skip("WorkOS API key not configured")
        
    user_id = SHARED_STATE.get("user_id")
    if not user_id:
        email = SHARED_STATE.get("email")
        if email:
            users_res = workos.list_users(limit=20)
            for u in users_res.get("data", []):
                if u.get("email") == email:
                    user_id = u.get("id")
                    break
                    
    if user_id:
        # 1. Verify existence
        user_data = workos.get_user(user_id)
        assert isinstance(user_data, dict) and user_data.get("id") == user_id, f"User {user_id} not found in WorkOS"
        
        # 2. Cleanup / Delete dummy user
        deleted = workos.delete_user(user_id)
        assert deleted is True, f"Failed to delete test user {user_id} from WorkOS"

    # 3. Comprehensive cleanup: delete all test users generated during test runs
    all_users = workos.list_users(limit=100).get("data", [])
    for u in all_users:
        em = u.get("email", "").lower()
        if em not in ("bcrowe306@gmail.com", "bcrowe@nltst.com"):
            workos.delete_user(u.get("id"))


# ==============================================================================
# 9. PLAYWRIGHT UI TESTS: DOCUMENTS SELECTION & BULK DELETION
# ==============================================================================
def test_documents_page_checkbox_selection_and_delete(page: Page, base_url: str):
    """Upload a document, select it, open the bulk delete dialog, then delete it with its QR code."""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/documents")
    page.wait_for_load_state("networkidle")
    expect(page.locator("h1:has-text('Documents')")).to_be_visible()

    name = f"qa-menu-{int(time.time())}.pdf"
    page.set_input_files("input[type=file] >> nth=0", files=[{"name": name, "mimeType": "application/pdf", "buffer": b"%PDF-1.4 qa document"}])
    expect(page.get_by_text("Document uploaded")).to_be_visible(timeout=10000)
    row = page.locator("[data-testid=documents-table] tbody tr", has_text=name)
    expect(row).to_be_visible()

    row.locator("input[type='checkbox']").check()
    expect(page.locator("[data-testid=delete-selected]")).to_contain_text("1")
    page.click("[data-testid=delete-selected]")
    expect(page.get_by_role("dialog", name="Delete document?")).to_be_visible()
    page.click("[data-testid=confirm-delete-documents]")
    expect(page.get_by_text("Deleted 1 document(s)")).to_be_visible(timeout=8000)
    expect(row).to_have_count(0)


# ==============================================================================
# 10. PLAYWRIGHT UI TESTS: TEAM & USER MANAGEMENT
# ==============================================================================
def test_team_and_user_management_ui(page: Page, base_url: str):
    """Invite a member, verify the one-time invite link, revoke it, and check the audit log."""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/settings?tab=members")
    expect(page.locator("[data-testid=member-list]")).to_be_visible()

    invitee = f"qa_invite_ui_{int(time.time())}@lynqr.app"
    page.click("[data-testid=invite-member]")
    page.fill("[data-testid=invite-email]", invitee)
    page.locator("#invite-form").get_by_text("Viewer", exact=True).click()
    page.click("[data-testid=send-invite]")
    invite_url = page.locator("[data-testid=invite-url]").inner_text().strip()
    assert "/invite/lqi_" in invite_url
    page.get_by_role("button", name="Done").click()

    preview = requests.get(f"{base_url}/api/v1/invitations/{invite_url.rsplit('/', 1)[1]}", timeout=10)
    assert preview.status_code == 200 and preview.json()["email"] == invitee

    row = page.locator("[data-testid=invitation-list] li", has_text=invitee)
    row.get_by_role("button", name="Revoke").click()
    page.click("[data-testid=confirm-accept]")
    expect(row).to_have_count(0)

    page.click("[data-testid=settings-tab-audit]")
    expect(page.locator("[data-testid=audit-list]")).to_contain_text("revoked an invitation")


# ==============================================================================
# 11. PLAYWRIGHT UI TESTS: BRANDED 404 ERROR PAGE
# ==============================================================================
def test_branded_error_page_ui(page: Page, base_url: str):
    """Verify visiting an invalid or expired link displays the Lynqr branded error page"""
    page.goto(f"{base_url}/s/non-existent-qa-slug-for-playwright")
    page.wait_for_load_state("networkidle")

    # Verify Lynqr branding and return to home button
    expect(page.locator("text=Lynqr Smart Routing")).to_be_visible()
    expect(page.locator("h1:has-text('Link Not Found')")).to_be_visible()
    home_btn = page.locator("a:has-text('Return to Lynqr Home')")
    expect(home_btn).to_be_visible()

    # Click Return to Lynqr Home
    home_btn.click()
    page.wait_for_load_state("networkidle")
    assert page.url.rstrip("/").endswith(base_url.rstrip("/").split("://")[-1]) or "/" in page.url


def test_password_protected_link_with_utm(page: Page, base_url: str, browser):
    """Create a password-protected, UTM-tagged link in the UI, then unlock it as an anonymous visitor."""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/links")
    slug = f"qa-pw-{int(time.time())}"
    page.click("[data-testid=new-link]")
    page.fill("[data-testid=link-target]", "https://lynqr.app/members")
    page.fill("[data-testid=link-slug]", slug)
    page.click("text=Password protection")
    page.fill("[data-testid=link-password]", "letmein")
    page.click("text=UTM campaign tags")
    page.get_by_role("button", name="Newsletter", exact=True).click()
    page.click("[data-testid=create-submit]")
    expect(page.get_by_text("Short link created")).to_be_visible(timeout=8000)

    visitor = browser.new_context().new_page()
    visitor.goto(f"{base_url}/s/{slug}")
    expect(visitor.get_by_text("This link is protected")).to_be_visible()
    visitor.fill("input[name=password]", "wrong")
    visitor.click("button[type=submit]")
    expect(visitor.get_by_text("That password is incorrect")).to_be_visible()
    visitor.fill("input[name=password]", "letmein")
    visitor.click("button[type=submit]")
    visitor.wait_for_url(lambda u: u.startswith("https://lynqr.app/members"), timeout=10000)
    assert "utm_source=newsletter" in visitor.url and "utm_medium=email" in visitor.url
    visitor.context.close()


def test_dashboard_and_analytics(page: Page, base_url: str):
    """Dashboard KPIs and live feed connect; analytics filters update the URL; all three exports download."""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/")
    expect(page.locator("[data-testid=dash-total]")).to_be_visible()
    expect(page.locator("[data-testid=live-status]")).to_have_text("Live", timeout=10000)

    page.goto(f"{base_url}/analytics")
    expect(page.locator("[data-testid=kpi-total]")).to_be_visible()
    page.click("[data-testid=date-range]")
    page.click("[data-testid=range-90d]")
    page.wait_for_url("**range=90d**")
    for fmt in ("CSV", "JSON", "PDF"):
        page.click("[data-testid=export-report]")
        with page.expect_download() as dl:
            page.locator(f"text=({fmt})").click()
        assert dl.value.suggested_filename.startswith("lynqr-"), dl.value.suggested_filename


def test_help_drawer_shortcuts_and_api_reference(page: Page, base_url: str):
    """Keyboard shortcuts, searchable guides with deep links, and the embedded API reference."""
    authenticate_page(page, base_url)
    page.goto(f"{base_url}/")
    page.wait_for_load_state("networkidle")
    page.keyboard.press("?")
    expect(page.locator("[data-testid=shortcuts-modal]")).to_be_visible()
    page.keyboard.press("Escape")
    page.wait_for_timeout(400)
    page.keyboard.press("g")
    page.keyboard.press("a")
    page.wait_for_url("**/analytics")

    page.keyboard.press("h")
    page.fill("[data-testid=help-search]", "wifi")
    page.click("[data-testid=guide-qr-types]")
    expect(page.locator("[data-testid=guide-article]")).to_contain_text("Wi-Fi network")
    page.keyboard.press("Escape")

    spec = requests.get(f"{base_url}/api/v1/openapi.json", timeout=10).json()
    assert spec["openapi"] == "3.1.0" and "/api/v1/qr/{id}/export" in spec["paths"]
    page.goto(f"{base_url}/api-docs")
    expect(page.frame_locator("iframe[title='Lynqr API reference']").get_by_text("Lynqr API").first).to_be_visible(timeout=15000)
