import json
import urllib.request
import urllib.error
from .config import get_secret

class CloudflareClient:
    def __init__(self):
        self.api_token = get_secret("CLOUDFARE_API_TOKEN")
        self.account_id = get_secret("CLOUDFARE_ACCOUNT_ID")
        self.domain_name = get_secret("DOMAIN_NAME", "lynqr.app")
        self.base_url = "https://api.cloudflare.com/client/v4"
        self.headers = {
            "Authorization": f"Bearer {self.api_token}",
            "Content-Type": "application/json"
        }

    def is_configured(self):
        return bool(self.api_token and self.account_id)

    def get_zone(self, domain=None):
        domain = domain or self.domain_name
        url = f"{self.base_url}/zones?name={domain}"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read().decode())
                zones = data.get("result", [])
                return zones[0] if zones else None
        except Exception as e:
            return {"error": str(e)}

    def get_dns_records(self, zone_id, record_name=None):
        url = f"{self.base_url}/zones/{zone_id}/dns_records"
        if record_name:
            url += f"?name={record_name}"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read().decode())
                return data.get("result", [])
        except Exception as e:
            return []

    def get_r2_buckets(self):
        url = f"{self.base_url}/accounts/{self.account_id}/r2/buckets"
        req = urllib.request.Request(url, headers=self.headers)
        try:
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read().decode())
                return data.get("result", {}).get("buckets", [])
        except Exception as e:
            return []

    def verify_r2_s3_storage(self, bucket_name="lynqr-storage"):
        access_key = get_secret("CLOUDFARE_ACCESS_KEY_ID")
        secret_key = get_secret("CLOUDFARE_SECRET_KEY")
        endpoint = get_secret("CLOUDFARE_S3_ENDPOINT")
        
        if not (access_key and secret_key and endpoint):
            return {"status": "skipped", "message": "R2 S3 credentials not configured"}
            
        try:
            import boto3
            from botocore.config import Config
            
            s3 = boto3.client(
                's3',
                endpoint_url=endpoint,
                aws_access_key_id=access_key,
                aws_secret_access_key=secret_key,
                config=Config(signature_version='s3v4')
            )
            # Test listing or checking bucket
            test_key = "healthcheck-test.txt"
            s3.put_object(Bucket=bucket_name, Key=test_key, Body=b"lynqr healthcheck ok")
            s3.delete_object(Bucket=bucket_name, Key=test_key)
            return {"status": "ok", "bucket": bucket_name, "verified": True}
        except Exception as e:
            return {"status": "error", "error": str(e)}
