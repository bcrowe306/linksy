# Linksy - Product Requirements Document (PRD)

Linksy is a high-performance, developer-friendly, and modern QR code and shortened link management platform inspired by Bitly, featuring real-time analytics, dynamic link routing, rich QR code customization, document-sharing QR codes, multi-tenant team collaboration, and a public REST API.

---

## 1. Core Requirements

### 1.1 QR Code Management & Customization (CRUD)
- Full lifecycle management (Create, Read, Update, Delete) for QR codes.
- **Customization engine:**
  - Foreground and background color palettes with theme support.
  - Center logo / icon embedding with safe error correction leveling (H - 30% recovery).
  - Custom border styling, padding / quiet zones, and caption / frame labels (e.g., "SCAN ME").
  - Dot patterns and corner / eye shape styling.
- Live preview editor in the web application.

### 1.2 Short Link Generation & Dynamic Redirection
- Generate unique, slug-based short URLs (e.g., `https://lnksy.io/s/{slug}`).
- Support for custom slugs and random collision-resistant nano-IDs.
- **Dynamic Link & QR Destination Editing:**
  - Update the destination URL or payload without re-printing or re-generating the physical QR code.
  - Temporary pausing / disabling of links and expiration dates.
  - Password protection and UTM parameter auto-tagging.

### 1.3 Tracking & Deep Analytics
- Track every scan and short link click event:
  - Timestamp (UTC + localized).
  - Referrer domain / channel (direct, social, email, web).
  - User-Agent parsing: Browser, Operating System, Device category (Mobile, Desktop, Tablet).
  - Geo-location resolution: Country, region, city (via IP geolocation / CDN headers).
  - Scan type differentiation (Direct QR scan vs. Web link click).
- Real-time event streaming / websocket updates for live dashboard activity.

### 1.4 Interactive Tracking Reports & Visualizations
- User-requested report generation on web dashboard:
  - Scan & click counts over custom time intervals (Today, 7D, 30D, custom range).
  - Geographic heat maps and top countries/cities table.
  - Device and browser breakdown charts.
  - Top performing links and QR codes.
- Exportable report summaries (CSV, JSON, PDF).

### 1.5 Export Engine in Multiple Formats
- High-resolution client-side & server-side export options:
  - **PNG** (standard web / print up to 4K resolution).
  - **JPEG** (custom background).
  - **PDF** (vector / ready-to-print single sheet or card format).
  - **SVG** (infinite vector scaling for designers and merchandise).

### 1.6 QR Code Types Supported
Linksy supports specialized payloads encoded natively into QR codes or routed via dynamic redirectors:
1. **Website / URL**: Dynamic HTTP/HTTPS redirection.
2. **Document / PDF / File**: Hosted file upload with instant QR distribution and re-upload capabilities.
3. **Contact Card (vCard 3.0 / 4.0 & MeCard)**: Name, phone, email, company, title, address, website.
4. **Wi-Fi Network**: SSID, encryption type (WPA/WPA2/WPA3/WEP/None), password, hidden status.
5. **Application Store Link**: Auto-detects iOS App Store or Android Google Play store based on device.
6. **Plain Text**: Raw textual content up to standard QR character limits.
7. **Email / Mailto**: Target email, subject, and prefilled body.
8. **SMS / Phone Call**: Phone number and prefilled text message.

### 1.7 Document Hosting & Reusable QR Codes
- Secure document upload (PDF, DOCX, presentation, image menus, etc.).
- Pluggable storage backend: Local disk storage for local development; S3 / Cloudflare R2 / Render persistent disk for production.
- **Reusability**: Replace the underlying document at any time while keeping the exact same physical QR code and short link intact (e.g. daily restaurant menus, quarterly reports, event agendas).

### 1.8 Team Collaboration & Multi-User Authentication
- Powered by WorkOS (Enterprise SSO, OAuth, Magic Links).
- Organization / Team workspaces:
  - Multiple users per team.
  - Role-Based Access Control (RBAC):
    - **Owner**: Full billing, team management, and workspace deletion rights.
    - **Admin**: User invitations, global workspace link/QR management, API key issuance.
    - **Member**: Create, view, edit, and export links and QR codes within their assigned workspace.
    - **Viewer**: Read-only analytics and QR download access.

### 1.9 Admin & User Management
- **End-User Self Service**: Profile management, personal API token generation, team switching, notification preferences.
- **Team / Admin Management**: Member invite links, role assignment, member suspension / removal, audit logs.

### 1.10 Publicly Accessible REST API
- Complete RESTful API for developers and integrations.
- API Key / Bearer token authentication with rate limiting tiers.
- Endpoints for QR generation, short link creation, analytics retrieval, and document upload.
- OpenAPI 3.1 / Swagger specification and interactive documentation (Scalar / Swagger UI) embedded directly in the application.
- Code examples in cURL, Go, Python, and JavaScript.

### 1.11 User-Facing Help & Guidance
- In-app tooltips for complex settings (e.g., error correction levels, UTM parameters, vCard fields).
- Embedded "Quick Docs" drawer / modal and keyboard shortcuts.
- Contextual help links to dedicated guides for each QR code type.

### 1.12 Look & Feel
- Inspired by Bitly: Clean, modern, responsive, high data density, and effortless navigation.
- Built-in multi-theme engine with **Dark Theme enabled by default** and Light theme toggle.
- TailwindCSS design system with custom accent colors and polished UI states.

---

## 2. Tech Stack

| Layer | Technology | Details |
|---|---|---|
| **Backend Language** | Go (1.24+) | High-throughput, low-latency, compiled binary |
| **HTTP Framework** | Go Fiber v3 (`github.com/gofiber/fiber/v3`) | Express-inspired ergonomics, fasthttp performance |
| **Realtime / Sockets** | Go Fiber Websocket v3 (`contrib/v3/websocket`) | Live analytics stream to web clients |
| **QR Generation** | `github.com/skip2/go-qrcode` + Go `image` / `fpdf` | Custom raster, vector SVG, logo injection, PDF export |
| **Database** | PostgreSQL 16+ (Render / Prod) & SQLite (Local / Test) | Dual-mode SQL database support for zero-dependency local dev |
| **Document Storage** | Storage Abstraction (Local Disk / S3 / R2) | Clean interface with local disk fallback and cloud support |
| **Frontend Framework** | Vue 3 + Vite | Composition API, `<script setup>`, Pinia, Vue Router |
| **Styling** | TailwindCSS + Heroicons / Lucide | Dark theme default, responsive mobile & desktop |
| **Authentication** | WorkOS (`workos.com`) | SSO, Directory Sync, Multi-factor auth, Session management |
| **Hosting & Cloud** | Render (`dashboard.render.com`) | Web service, managed PostgreSQL, background workers |
| **Repository** | GitHub (`bcrowe306/linksy`) | Private repository |
| **CI / CD** | GitHub Actions | Automated build, test, lint, and Render deploy hook trigger |
