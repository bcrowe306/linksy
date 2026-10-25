import json
import urllib.request
import urllib.error
from .config import get_secret

class RenderClient:
    def __init__(self):
        self.api_key = get_secret("RENDER_API_KEY")
        self.base_url = "https://api.render.com/v1"
        self.headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Accept": "application/json",
            "Content-Type": "application/json"
        }

    def is_configured(self):
        return bool(self.api_key)

    def list_services(self, limit=20):
        url = f"{self.base_url}/services?limit={limit}"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode())
        except Exception as e:
            return []

    def get_service(self, service_id):
        url = f"{self.base_url}/services/{service_id}"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode())
        except Exception as e:
            return {"error": str(e)}

    def get_latest_deploy(self, service_id):
        url = f"{self.base_url}/services/{service_id}/deploys?limit=1"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read().decode())
                return data[0]["deploy"] if data else None
        except Exception as e:
            return {"error": str(e)}

    def get_env_vars(self, service_id):
        url = f"{self.base_url}/services/{service_id}/env-vars"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode())
        except Exception as e:
            return []
