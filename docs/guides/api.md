---
title: Using the Lynqr API
description: Authenticate with API keys and automate short links, QR codes, documents and analytics.
order: 9
---

# Using the Lynqr API

## API keys
Create a key in **Profile → API keys**. Keys act as you, with your current role, in the workspace that was active when you created them. Send the key as a bearer token:

```bash
curl https://lynqr.app/api/v1/links \
  -H "Authorization: Bearer lk_live_…"
```

Keys are shown once. Revoke them any time, and give each system its own key.

## Create a short link

```bash
curl -X POST https://lynqr.app/api/v1/links \
  -H "Authorization: Bearer $LYNQR_KEY" \
  -H "Content-Type: application/json" \
  -d '{"target_url":"https://example.com/launch","custom_slug":"launch","utm_source":"newsletter"}'
```

## Create a dynamic QR code and download it

```bash
curl -X POST https://lynqr.app/api/v1/qr \
  -H "Authorization: Bearer $LYNQR_KEY" \
  -H "Content-Type: application/json" \
  -d '{"type":"website","title":"Poster","target_url":"https://example.com","is_dynamic":true,
       "style":{"dot_style":"extra-rounded","frame_style":"banner-bottom","label_text":"SCAN ME"}}'

curl -o poster.svg "https://lynqr.app/api/v1/qr/{id}/export?format=svg" \
  -H "Authorization: Bearer $LYNQR_KEY"
```

## Rate limits
600 requests per minute per key. Requests over the limit return `429 Too Many Requests` with a `Retry-After` header.

## Errors
Errors return JSON: `{"error": "human-readable message"}` with a meaningful status (`400` invalid input, `401` bad credentials, `403` your role can't do this, `404` not found in this workspace, `409` conflict such as a taken slug).

The full, interactive reference, with examples in cURL, Go, Python and JavaScript, is in **API Reference**.
