import json
import urllib.request
import urllib.error
import urllib.parse
from .config import get_secret

class WorkOSClient:
    def __init__(self):
        self.api_key = get_secret("WORKOS_API_KEY_PROD") or get_secret("WORKOS_API_KEY") or get_secret("WORKOS_API_KEY_TEST")
        self.client_id = get_secret("WORKOS_CLIENT_ID_PROD") or get_secret("WORKOS_CLIENT_ID") or get_secret("WORKOS_CLIENT_ID_STAGING")
        self.base_url = "https://api.workos.com/user_management"
        self.headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        }

    def is_configured(self):
        return bool(self.api_key)

    def get_user(self, user_id):
        url = f"{self.base_url}/users/{user_id}"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode())
        except Exception as e:
            return {"error": str(e)}

    def create_user(self, email, password, first_name="", last_name=""):
        """Creates a pre-verified user via the management API (test fixtures only)."""
        body = json.dumps({
            "email": email, "password": password, "first_name": first_name,
            "last_name": last_name, "email_verified": True,
        }).encode()
        req = urllib.request.Request(f"{self.base_url}/users", data=body, headers=self.headers, method="POST")
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            return {"error": e.read().decode()}

    def find_user_by_email(self, email):
        url = f"{self.base_url}/users?email={urllib.parse.quote(email)}"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read().decode()).get("data", [])
                return data[0] if data else None
        except Exception:
            return None

    def delete_user(self, user_id):
        url = f"{self.base_url}/users/{user_id}"
        req = urllib.request.Request(url, headers=self.headers, method="DELETE")
        try:
            with urllib.request.urlopen(req) as resp:
                return resp.status in (200, 204)
        except Exception:
            return False

    def list_users(self, limit=10):
        url = f"{self.base_url}/users?limit={limit}"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode())
        except Exception as e:
            return {"error": str(e)}
