import os

def load_secrets():
    secrets = {}
    paths_to_check = [
        os.path.join(os.path.dirname(__file__), '..', '..', 'secrets.env'),
        'secrets.env',
        '/Users/brandoncrowe/Documents/Git/linksy/secrets.env'
    ]
    for p in paths_to_check:
        if os.path.exists(p):
            with open(p, 'r') as f:
                for line in f:
                    line = line.strip()
                    if line and not line.startswith('#') and '=' in line:
                        k, v = line.split('=', 1)
                        secrets[k.strip()] = v.strip()
            break
            
    # Overlay with system environment variables if present
    for k, v in os.environ.items():
        secrets[k] = v
        
    return secrets

SECRETS = load_secrets()

def get_secret(key, default=""):
    return SECRETS.get(key, os.getenv(key, default))
