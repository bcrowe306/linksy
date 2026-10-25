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

def authenticate_page(page: Page, base_url: str):
    """Helper to ensure the page has an active authenticated session"""
    email, password = get_or_create_credentials()
    
    # Check if already authenticated
    page.goto(f"{base_url}/")
    page.wait_for_load_state("networkidle")
    profile_link = page.locator("header a[href='/profile']")
    if profile_link.is_visible():
        return
        
    # Attempt login
    page.goto(f"{base_url}/login")
    page.wait_for_load_state("networkidle")
    page.fill("input[type='email']", email)
    page.fill("input[type='password']", password)
    page.click("button[type='submit']")
    page.wait_for_timeout(2000)
    
    # If login fails (user not registered yet), sign up
    if "/login" in page.url:
        page.goto(f"{base_url}/signup")
        page.wait_for_load_state("networkidle")
        page.fill("input[placeholder='Jane']", "QA_User")
        page.fill("input[placeholder='Doe']", "Tester")
        page.fill("input[placeholder='jane@company.com']", email)
        page.fill("input[placeholder='At least 8 characters']", password)
        page.fill("input[placeholder='Re-enter password']", password)
        page.click("button:has-text('Create Free Account')")
        page.wait_for_url(lambda u: "/signup" not in u, timeout=12000)

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
    
    records = cf.get_dns_records(zone["id"], "test.lynqr.app")
    assert len(records) > 0, "DNS record for test.lynqr.app not found in Cloudflare"
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
    assert deploy is not None, "No deployment records found"
    assert deploy.get("status") == "live", f"Latest Render deploy status is {deploy.get('status')}"

# ==============================================================================
# 3. PLAYWRIGHT UI TESTS: THEME SYSTEM
# ==============================================================================
def test_theme_system_toggle_and_persistence(page: Page, base_url: str):
    """Verify dark/light theme switching and persistence across page reloads"""
    page.goto(base_url)
    page.wait_for_load_state("networkidle")
    
    # Locate theme switcher button in header
    theme_btn = page.locator("header button[title*='Theme']")
    expect(theme_btn).to_be_visible()
    
    # Check initial preference
    initial_pref = page.evaluate("() => localStorage.getItem('linksy_theme_pref') || 'system'")
    assert initial_pref in ["system", "dark", "light"]
    
    # Click to toggle theme
    theme_btn.click()
    new_pref = page.evaluate("() => localStorage.getItem('linksy_theme_pref')")
    assert new_pref != initial_pref, f"Theme preference did not change: {new_pref}"
    
    # Verify reload preserves theme preference
    page.reload()
    page.wait_for_load_state("networkidle")
    preserved_pref = page.evaluate("() => localStorage.getItem('linksy_theme_pref')")
    assert preserved_pref == new_pref, f"Theme not preserved after reload: {preserved_pref}"

# ==============================================================================
# 4. PLAYWRIGHT UI TESTS: USER REGISTRATION & AUTHENTICATION
# ==============================================================================
def test_signup_and_login_ui(page: Page, base_url: str):
    """Test user registration and subsequent sign-in via the web interface"""
    email, password = get_or_create_credentials()
    
    # 1. Registration Flow
    page.goto(f"{base_url}/signup")
    page.wait_for_load_state("networkidle")
    
    page.fill("input[placeholder='Jane']", "QA_User")
    page.fill("input[placeholder='Doe']", "Tester")
    page.fill("input[placeholder='jane@company.com']", email)
    page.fill("input[placeholder='At least 8 characters']", password)
    page.fill("input[placeholder='Re-enter password']", password)
    
    # Submit registration
    page.click("button:has-text('Create Free Account')")
    
    # Expect redirect away from /signup
    page.wait_for_url(lambda url: "/signup" not in url, timeout=15000)
    expect(page.locator("header")).to_be_visible()
    
    # Extract session cookie
    cookies = page.context.cookies()
    session_cookie = next((c for c in cookies if c["name"] == "linksy_session"), None)
    if session_cookie:
        cookie_val = session_cookie["value"]
        if "user_" in cookie_val:
            parts = cookie_val.split("_")
            if len(parts) >= 3 and parts[1] == "user":
                SHARED_STATE["user_id"] = "user_" + parts[2]
            
    # 2. Log Out Flow
    signout_btn = page.locator("header button[title='Sign Out']")
    if signout_btn.is_visible():
        signout_btn.click()
        page.wait_for_timeout(1000)
        
    # 3. Log In Flow
    page.goto(f"{base_url}/login")
    page.wait_for_load_state("networkidle")
    
    page.fill("input[type='email']", email)
    page.fill("input[type='password']", password)
    page.click("button[type='submit']")
    
    # Expect successful navigation back to dashboard
    page.wait_for_url(lambda url: "/login" not in url, timeout=15000)
    expect(page.locator("header a[href='/profile']")).to_be_visible(timeout=8000)

# ==============================================================================
# 5. PLAYWRIGHT UI TESTS: PROFILE & DEVELOPER API KEYS
# ==============================================================================
def test_profile_and_developer_api_keys(page: Page, base_url: str):
    """Test account profile editing, API key generation modal, and revocation"""
    authenticate_page(page, base_url)
    
    # Navigate to Profile via avatar link
    avatar_link = page.locator("header a[href='/profile']")
    expect(avatar_link).to_be_visible()
    avatar_link.click()
    page.wait_for_url("**/profile", timeout=10000)
    
    # 1. Update Profile Form
    first_name_input = page.locator("input[placeholder='Jane']")
    first_name_input.fill("QA_Updated")
    page.click("button:has-text('Save Changes')")
    expect(page.locator("text=Profile updated successfully")).to_be_visible(timeout=8000)
    
    # 2. Switch to Developer API Keys Tab
    page.click("button:has-text('API Keys')")
    page.wait_for_timeout(500)
    
    # Click Generate New API Key
    page.click("button:has-text('Generate New API Key')")
    expect(page.locator("h3:has-text('Generate New API Key')")).to_be_visible()
    
    key_name = f"Playwright-Key-{int(time.time())}"
    page.fill("input[placeholder*='Production Backend Bot']", key_name)
    page.click("button:has-text('Generate Key')")
    
    # Verify Modal with full key appears
    expect(page.locator("text=API Key Created Successfully")).to_be_visible(timeout=8000)
    key_locator = page.locator("div.font-mono.text-emerald-400").first
    full_key = key_locator.inner_text().strip()
    assert full_key.startswith("lk_live_"), f"Generated key does not start with lk_live_: {full_key}"
    SHARED_STATE["api_key"] = full_key
    
    # Dismiss copy modal
    page.click("button:has-text('I Have Saved This Key')")
    page.wait_for_timeout(1000)
    
    # Verify key prefix appears in table
    prefix = full_key[:12]
    expect(page.get_by_text(prefix)).to_be_visible()
    
    # 3. Revoke API Key
    revoke_btn = page.locator("button[title='Revoke key']").first
    if revoke_btn.is_visible():
        revoke_btn.click()
        page.wait_for_timeout(1000)

# ==============================================================================
# 6. PLAYWRIGHT UI TESTS: QR STUDIO & UNSAVED CHANGES NAVIGATION GUARD
# ==============================================================================
def test_qr_studio_unsaved_changes_warning(page: Page, base_url: str):
    """Verify that editing QR code prompts user when attempting to navigate away"""
    page.goto(f"{base_url}/qr-studio")
    page.wait_for_load_state("networkidle")
    
    # Fill the QR Studio Title input
    title_input = page.locator("input[placeholder*='Summer Campaign']")
    expect(title_input).to_be_visible()
    title_input.fill("Modified QR Code Without Saving")
    
    # Verify the "Unsaved Changes" badge appears in header
    expect(page.locator("text=Unsaved Changes")).to_be_visible(timeout=5000)
    
    # Listen for the window.confirm dialog
    dialog_handled = {"triggered": False, "message": ""}
    def handle_dialog(dialog):
        dialog_handled["triggered"] = True
        dialog_handled["message"] = dialog.message
        dialog.accept() # Accept to proceed with navigation
        
    page.once("dialog", handle_dialog)
    
    # Attempt to navigate away by clicking "Short Links" in the sidebar
    links_nav = page.locator("aside a[href='/links']").first
    links_nav.click()
    
    # Verify dialog was triggered with unsaved changes prompt
    page.wait_for_url("**/links", timeout=8000)
    assert dialog_handled["triggered"], "Navigation prompt dialog was not triggered when leaving with unsaved changes"
    assert "unsaved changes" in dialog_handled["message"].lower(), f"Unexpected dialog message: {dialog_handled['message']}"

# ==============================================================================
# 7. PLAYWRIGHT UI TESTS: LINKS CREATION & REDIRECTION
# ==============================================================================
def test_short_link_creation_and_redirection(page: Page, base_url: str):
    """Test creating a new short link via modal and verifying 302 redirection"""
    page.goto(f"{base_url}/links")
    page.wait_for_load_state("networkidle")
    
    # Click New Link button
    new_link_btn = page.locator("button:has-text('New Link')").first
    new_link_btn.click()
    
    # Modal should appear
    expect(page.locator("h2:has-text('Create New Asset')")).to_be_visible()
    
    slug = f"qa-link-{int(time.time())}"
    SHARED_STATE["slug"] = slug
    
    page.fill("input[type='url']", "https://lynqr.app")
    page.fill("input[placeholder*='Marketing Promo']", "QA Automation Link")
    page.fill("input[placeholder='launch2026']", slug)
    
    page.click("button:has-text('Create Asset')")
    
    # Verify link appears in table
    page.wait_for_timeout(2000)
    expect(page.get_by_text(f"/s/{slug}")).to_be_visible(timeout=8000)
    
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
