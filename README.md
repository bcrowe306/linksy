# Linksy ⚡️

> A dynamic QR code and short link management platform built with **Go (Fiber v3)**, **Vue 3**, **TailwindCSS**, and **PostgreSQL**. Inspired by Bitly, designed with a sleek dark-theme UI, real-time analytics, dynamic link routing, reusable document-sharing QR codes, multi-format exports, and a public developer REST API.

---

## 🚀 Key Features & Requirements Matrix

| Requirement | Implementation Details |
|---|---|
| **1. QR Code CRUD & Customization** | Foreground/background color palettes, quiet zones, border styling, label frames ("SCAN ME"), and center logo embedding. |
| **2. Short Link Generation** | Collision-resistant random slugs and custom vanity slugs (e.g. `/s/my-brand`). |
| **3. Analytics & Tracking** | Tracks every scan and click: IP, User-Agent, Browser, Operating System, Device (Mobile, Desktop, Tablet), Country, and Referrer. |
| **4. On-Demand Tracking Reports** | Dashboard visualizations, asset filtering, device distribution, geographic breakdown, and CSV report export. |
| **5. Multi-Format QR Export** | High-resolution exports available via API & UI in **PNG (HD)**, **JPEG**, **Printable PDF**, and **SVG Vector**. |
| **6. Dynamic Destination Editing** | Change what a QR code or short link points to at any time without reprinting physical merchandise or changing URLs. |
| **7. 8 Specialized QR Types** | Website / URL, Document, Contact Card (vCard 3.0), Wi-Fi Network, SMS, Email, Plain Text, and App Store links. |
| **8. Reusable Document Sharing** | Upload PDF menus, catalogs, or reports and generate a sharing QR code. Re-upload a newer version anytime while keeping the QR code identical. |
| **9. Multi-User Team Auth** | Powered by WorkOS (Enterprise SSO, SAML, Magic Links) with Role-Based Access Control (`Owner`, `Admin`, `Member`, `Viewer`). |
| **10. Public Developer REST API** | Fully documented OpenAPI 3.1 schema and interactive **Scalar** API reference embedded directly at `/api/docs`. |
| **11. Team & Admin Management** | Workspace member invitations, role management, and audit visibility in Team Settings. |
| **12. End-User Self Service** | Instant API key generation, profile settings, and workspace switcher. |
| **13. In-App User Guidance** | Interactive "Quick Docs & Guidance" drawer, tooltip explanations for error correction levels, and cheat sheets. |

---

## 🛠 Tech Stack

- **Backend**: Go (Go 1.25+)
- **HTTP Framework**: [Go Fiber v3](https://github.com/gofiber/fiber/v3)
- **QR Engine**: [skip2/go-qrcode](https://github.com/skip2/go-qrcode) + Go `image` / `gofpdf`
- **Database**: Dual-mode engine supporting **PostgreSQL 16+** (Production/Render) and embedded **SQLite** (Local zero-dependency development & testing)
- **Frontend**: Vue 3 (Composition API) + Vite + Pinia + TailwindCSS + Lucide Icons (Dark theme by default)
- **Authentication**: WorkOS (`workos.com`)
- **Hosting**: Render (`render.yaml` Blueprint with persistent disks)
- **CI / CD**: GitHub Actions (`.github/workflows/ci.yml`)

---

## 📦 Local Development Quickstart

### Prerequisites
- Go 1.24+ installed
- Node.js 20+ and npm installed

### 1. Clone the repository
```bash
git clone https://github.com/bcrowe306/linksy.git
cd linksy
```

### 2. Run Tests & Verification
Linksy comes with automated test harnesses covering unit and E2E API integration flows:
```bash
make test
```

### 3. Build & Run Application
You can build both the Vue 3 frontend and Go backend binary with:
```bash
make build
./bin/linksy
```
Then visit **`http://localhost:8080`** in your browser!

### 4. Running Backend & Frontend in Dev Mode (Hot Reload)
Terminal 1 (Backend API server):
```bash
make dev-backend
```

Terminal 2 (Vite Frontend with HMR):
```bash
make dev-frontend
```
Open **`http://localhost:3000`** (Vite proxies API calls to port 8080).

---

## 📖 API Documentation & OpenAPI Specification

- **Interactive Scalar API Portal**: `http://localhost:8080/api/docs`
- **Raw OpenAPI 3.1 Specification**: `http://localhost:8080/api/v1/openapi.json`
- **Health Check**: `http://localhost:8080/health`

### Example: Create a Dynamic QR Code via cURL
```bash
curl -X POST http://localhost:8080/api/v1/qr \
  -H "Content-Type: application/json" \
  -d '{
    "type": "website",
    "title": "Summer Campaign",
    "target_url": "https://example.com/promo",
    "is_dynamic": true,
    "style": {
      "fg_color": "#4F46E5",
      "bg_color": "#FFFFFF",
      "border_width": 2,
      "border_color": "#6366F1",
      "label_text": "SCAN ME"
    }
  }'
```

### Example: Export QR Code to Print-Ready PDF
```bash
curl -O "http://localhost:8080/api/v1/qr/{id}/export?format=pdf&download=true"
```

---

## ☁️ Deployment to Render

1. Log in to [dashboard.render.com](https://dashboard.render.com/).
2. Click **New +** > **Blueprint**.
3. Connect your GitHub repository: `bcrowe306/linksy`.
4. Render will read [`render.yaml`](file:///Users/brandoncrowe/Documents/Git/linksy/render.yaml) and automatically provision:
   - **`linksy-postgres`**: Managed PostgreSQL database.
   - **`linksy-app`**: Go web service running the compiled Linksy server.
   - **`linksy-storage`**: 10GB persistent disk for document storage.
5. In your GitHub repository **Settings > Secrets and variables > Actions**, add:
   - `RENDER_DEPLOY_HOOK_URL`: Your Render Webhook Deploy URL to enable automatic deployment on every push to `main`.
