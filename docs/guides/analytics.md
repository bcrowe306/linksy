---
title: Understanding analytics
description: What Lynqr counts as a click or scan, how locations, channels and visitors are measured, and how to export reports.
order: 7
---

# Understanding analytics

## Clicks and scans
- A **click** is a visit to a short link (`/s/…`).
- A **scan** is a visit through a QR code's scan address (`/q/…`).

Scans of a QR code that points at a short link count as scans, not clicks, so the numbers never double up.

## What's excluded
Search-engine crawlers, link-preview fetchers (Slack, Facebook, Twitter, WhatsApp…), uptime monitors and scripted requests are filtered out. Visits to paused or expired links are not counted.

## Unique visitors
Estimated from anonymized network addresses. Lynqr never stores raw IP addresses.

## Locations
Country, region and city come from Cloudflare's network. Visitors behind VPNs appear at the VPN's location.

## Channels
| Channel | Meaning |
|---|---|
| Direct | No referring site (typed, bookmarked, apps) |
| Social | Facebook, Instagram, LinkedIn, X, TikTok, Reddit… |
| Email | Webmail such as Gmail or Outlook |
| Search | Google, Bing, DuckDuckGo… |
| Other websites | Any other referring site |
| QR scan | Arrived by scanning a QR code |

## Time zones
Reports use your browser's time zone (shown next to the filters), so "today" and the weekday×hour heatmap match your local clock.

## Exports
From **Analytics → Export**:
- **CSV**: every event in the range, for spreadsheets.
- **JSON**: the full report, for your own tools.
- **PDF**: a printable summary to share.
