# Lynqr (lynqr.app) — SEO, Organic Visibility & Growth Playbook

This document serves as the master strategy and implementation reference for ranking **lynqr.app** across traditional search engines (Google, Bing, DuckDuckGo) and AI answer engines (ChatGPT Search, Perplexity, Google AI Overviews, Claude).

---

## 1. Market Positioning & The "Anti-Bait-and-Switch" Wedge

### Why Broad Head Terms Take Time
Keywords like `"QR code generator"` and `"URL shortener"` are dominated by 10+ year-old domains (`qr-code-generator.com`, Canva, Adobe, Bitly, Uniqode) with Domain Ratings (DR) of 70–90+ and tens of thousands of backlinks. A newly launched domain cannot outrank them on day one for two-word generic queries alone.

### Our Unfair Organic Wedge
Market research across Reddit (`r/DigitalMarketing`, `r/AskMarketing`, `r/graphic_design`, `r/smallbusiness`) and review platforms reveals the **#1 user pain point in the QR code market**:
1. **Deceptive "Free" Trials (Bait-and-Switch):** Competitors advertise "Free QR Code Generator," trick users into creating a *dynamic* trial code without clear warnings, wait until the user prints 1,000 menus or flyers, and **deactivate the QR code after 14 days** behind a \$100–\$240/year annual-only paywall.
2. **Per-User & Scan-Limit Pricing Traps:** Competitors charge \$35–\$99/month, cap monthly scans, or charge per team member.

**Lynqr's SEO & Conversion Hook:**
- **Truly Free Static QR Codes:** Created at `/qr-code-generator` with **no sign-up, no watermark, and zero expiration—ever**.
- **Transparent Dynamic QR & Short Links:** A flat **\$5/month (billed annually) or \$8.99/month per workspace** with unlimited team members, links, and QR codes, plus a 7-day cardless free trial.

---

## 2. Keyword & Content Architecture

### Tier 1: High-Intent "Anti-Expiration" & Comparison Queries (Fastest Ranking & Highest Conversion)
| Target Keyword Cluster | Target URL | Search Intent |
| :--- | :--- | :--- |
| `free qr code generator that doesn't expire`, `qr code generator no expiration`, `permanent qr code generator` | `/guides/free-qr-code-generator-no-expiration` & `/qr-code-generator` | Frustrated buyers looking for an honest tool after being burned by 14-day trial traps |
| `bitly qr code alternative`, `uniqode alternative`, `qr code generator pro alternative` | `/guides/bitly-qr-code-alternative` | Bottom-of-funnel buyers comparing paid dynamic QR & link platforms |
| `static vs dynamic qr code`, `do qr codes expire` | `/guides/dynamic-vs-static` | Educational + commercial investigation |

### Tier 2: Programmatic / Niche Use-Case Generators (Mid-Funnel Micro-Tools)
| Target Keyword Cluster | Target URL | Product Tie-In |
| :--- | :--- | :--- |
| `free qr code generator`, `url qr code generator` | `/qr-code-generator` | Core free static generator + upsell to dynamic links |
| `wifi qr code generator`, `share wifi password qr code` | `/qr-code-generator/wifi` | Cafés, Airbnbs, offices, hotels |
| `vcard qr code generator`, `business card qr code` | `/qr-code-generator/vcard` | Sales teams, events, badges |
| `google review qr code generator` | `/qr-code-generator/google-review` | Restaurants, local businesses, clinics |
| `pdf qr code generator`, `restaurant menu qr code` | `/qr-code-generator/pdf` | Direct bridge to Lynqr Pro **Hosted Documents** (`/d/:id`) |
| `whatsapp qr code generator`, `click to chat qr code` | `/qr-code-generator/whatsapp` | Customer support, international commerce |
| `instagram qr code generator` | `/qr-code-generator/instagram` | Creators, retail packaging, events |
| `email qr code generator` | `/qr-code-generator/email` | Support & feedback flows |
| `sms qr code generator` | `/qr-code-generator/sms` | Opt-ins & text campaigns |
| `phone call qr code generator` | `/qr-code-generator/phone` | Service businesses & signage |
| `text qr code generator` | `/qr-code-generator/text` | Offline serials, notes & instructions |

---

## 3. Iterative Codebase Implementation Plan

Each iteration is self-contained, verified locally (`make test`), committed and pushed to GitHub (`main`), deployed to Render production (`https://lynqr.app`), and verified live.

### Iteration 1: Technical SEO Foundation, `/llms.txt`, `<lastmod>` Sitemap & Homepage Internal Linking
- [x] Create `SEO-ADVERTIZING.md` reference playbook.
- [x] **Upgrade `/sitemap.xml` (`internal/api/handler/public.go`):** Include `<lastmod>` timestamps for core pages, generator pages, and Markdown guides (using front-matter `updated` dates).
- [x] **Upgrade `/robots.txt` (`internal/api/handler/public.go`):** Explicitly allow `/qr-code-generator`, `/guides`, `/llms.txt`, `/terms`, `/privacy`, `/acceptable-use` while blocking private/redirect routes (`/api/`, `/admin`, `/s/`, `/q/`, `/d/`, `/qr/`, `/settings`).
- [x] **Add `/llms.txt` (`internal/api/handler/public.go` & `internal/api/router.go`):** Machine-readable Markdown overview of Lynqr's free generator tools, dynamic QR Studio, hosted PDF documents, pricing, guides, and OpenAPI spec for AI search engines (ChatGPT Search, Perplexity, Claude, Gemini).
- [x] **Fix Homepage Internal Linking (`web/home.html`):**
  - Add `Free QR generator` (`/qr-code-generator`) to the desktop and mobile top navigation bars so `/` passes direct link equity to `/qr-code-generator`.
  - Add an indexable "Free QR Code Generators for Every Use Case" section on `home.html` with direct links to all generator subpages.
  - Enrich homepage JSON-LD (`SoftwareApplication` `featureList`).

### Iteration 2: High-Intent Use-Case Generator Pages & Rich Schema (`HowTo` + `BreadcrumbList`)
- [ ] **Add Niche Use-Case Generators (`internal/api/handler/generator_pages.go` & `internal/web/templates/generator.html`):**
  - Add `/qr-code-generator/google-review` (Google Review QR Code Generator)
  - Add `/qr-code-generator/pdf` (PDF & Restaurant Menu QR Code Generator)
  - Add `/qr-code-generator/whatsapp` (WhatsApp QR Code Generator)
  - Add `/qr-code-generator/instagram` (Instagram QR Code Generator)
  - Support custom input label and placeholder for URL-based niche generators.
- [ ] **Optimize Titles & Meta Descriptions:** Emphasize *"Free ... QR Code Generator (No Sign-Up, Never Expires)"* to boost click-through rates (CTR) in SERPs.
- [ ] **Add `HowTo` and `BreadcrumbList` JSON-LD Schema:** Emit `HowTo`, `BreadcrumbList`, `WebApplication`, and `FAQPage` in the `@graph` of every generator page.

### Iteration 3: Bottom-of-Funnel Comparison & Anti-Expiration Guides + Guide Schema
- [ ] **Create `/guides/free-qr-code-generator-no-expiration` (`docs/guides/free-qr-code-generator-no-expiration.md`):** Comprehensive guide explaining why "free" QR codes from other providers deactivate after 14 days, when to use free static codes vs. dynamic codes, and print-size best practices.
- [ ] **Create `/guides/bitly-qr-code-alternative` (`docs/guides/bitly-qr-code-alternative.md`):** Side-by-side comparison of Lynqr vs. Bitly, QR Code Generator Pro, and Uniqode (workspace flat pricing vs. per-seat pricing, hosted PDF replacement, custom short links, and live analytics).
- [ ] **Enrich Guide JSON-LD & Template (`internal/api/handler/public.go` & `internal/web/templates/doc.html`):** Add `BreadcrumbList` and `dateModified` to `TechArticle` schema, and link directly to the free generator and signup from guide footers.

### Iteration 4: Built-in Viral Referral Loop ("Powered by Lynqr") & Full Production Audit
- [ ] **Viral Loop on Public Interstitials (`internal/api/handler/handlers.go`):** Upgrade the footer on password-protected link pages (`renderPasswordPage`), app store choosers (`renderAppChooser`), and branded status pages (`renderBrandedError`) to include a clean link to `https://lynqr.app/qr-code-generator` ("Powered by Lynqr · Free QR Code Generator").
- [ ] **End-to-End Verification:** Run full test suite + verify live production (`https://lynqr.app`) across all new routes, schemas, and sitemaps.

---

## 4. Off-Site Visibility & Backlink Playbook (Zero-Ad-Spend Channels)

Because Google relies on backlinks (Domain Rating) to trust new domains, pair the on-site improvements above with these free off-site actions:

1. **Google Search Console & Bing Webmaster Tools:**
   - Submit `https://lynqr.app/sitemap.xml` in Google Search Console.
   - Use **URL Inspection → Request Indexing** on `/`, `/qr-code-generator`, `/qr-code-generator/wifi`, `/qr-code-generator/google-review`, `/qr-code-generator/pdf`, `/guides/free-qr-code-generator-no-expiration`, and `/guides/bitly-qr-code-alternative`.
   - Import the GSC property into [Bing Webmaster Tools](https://www.bing.com/webmasters) (powers Bing, DuckDuckGo, ChatGPT Search, and Copilot).
2. **High-Authority Software Directories (First 15 Backlinks):**
   - **AlternativeTo:** List Lynqr as an alternative to *Bitly*, *QR Code Generator*, *QR Code Monkey*, *Uniqode*, and *Dub.co*.
   - **Product Hunt & BetaList:** Launch *"Lynqr — Dynamic QR codes, hosted PDFs & short links for \$5/mo (plus a genuinely free static QR generator)"*.
   - **SaaSHub, G2, Capterra, Indie Hackers Products, There's An AI/Tool For That.**
3. **High-Signal Community Posts (Reddit & Hacker News):**
   - Post a **Show HN** on Hacker News highlighting the Go + Fiber + SVG/PNG QR verification engine (`internal/qr/verify.go` contrast & scannability checker) and OpenAPI spec.
   - Reply helpfully in `r/smallbusiness`, `r/restaurateur`, and `r/DigitalMarketing` threads where users ask for QR generators that don't expire or lock menus behind \$100+ paywalls.
