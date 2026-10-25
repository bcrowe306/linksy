---
title: Designing QR codes that scan
description: Colors, logos, frames, error correction and print sizes. How to brand a QR code without breaking it.
order: 4
---

# Designing QR codes that scan

QR Studio test-scans every design as you edit it. If the badge says **May not scan**, follow the suggestions it shows.

## Contrast
Scanners need a dark code on a light background. Aim for a contrast ratio of **4:1 or higher** (shown in the preview). Light-on-dark ("inverted") codes work on modern phones but fail on some older scanners.

## Logos
Logos hide part of the code. Lynqr removes the modules under the logo cleanly and locks error correction to **High**, which can recover about 30% of the code. Keep logos at or below ~25% of the code width, and use a badge (circle or rounded square) so the logo stands apart.

## Error correction
| Level | Recovers | Use when |
|---|---|---|
| L | ~7% | Clean screens, smallest code |
| M | ~15% | Default for most uses |
| Q | ~25% | Outdoor or rough surfaces |
| H | ~30% | Logos, stickers that may scuff |

Higher levels make the pattern denser, so print slightly larger.

## Quiet zone
Leave blank space around the code: at least 2 modules (the default), and 4 for print. Frames add their own padding.

## Print size
A good rule is **scan distance ÷ 10**. A code scanned from 30 cm should be at least 3 cm wide. Dense codes (long URLs, big vCards) need more space. Dynamic codes stay small because they only encode a short address.

## Formats
- **SVG**: infinitely scalable, for designers and large prints.
- **PDF sheet / card**: print-ready, vector.
- **PNG**: up to 4096 px, transparent background supported.
- **JPEG**: solid background, for tools that don't accept PNG.
