import json
import urllib.request
import urllib.error
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
