---
title: QR code types
description: Every QR code type Lynqr supports (website, document, contact card, Wi-Fi, app stores, text, email, SMS and phone) and how each one behaves.
order: 3
---

# QR code types

## Website {#website}
Opens a web page. Turn **Dynamic** on to change the destination later and track scans. Destinations must be public `http`/`https` addresses; private network addresses are blocked for security.

## Document {#document}
Hosts a PDF, image menu, presentation or spreadsheet (up to 15 MB) and links the code to it. Upload a new version on the **Documents** page and every printed code serves it immediately. Ideal for daily menus, price lists and reports.

## Contact card (vCard) {#vcard}
Saves a contact to the phone: name, company, title, phones, email, website and address.
- **vCard 3.0** works on virtually every phone (recommended).
- **vCard 4.0** is the newer standard.
- **MeCard** is the most compact format, producing a smaller code, but supports fewer fields.

Turn **Dynamic** on to host the card on Lynqr. The code gets smaller, you can edit the contact later, and scans are tracked.

## Wi-Fi network {#wifi}
Lets guests join your network without typing the password. Choose the security type your router uses (WPA/WPA2 for most networks). Wi-Fi codes are static: the credentials live in the code, so it works offline. Special characters in names and passwords are handled automatically.

## App stores {#app}
One code for both stores. iPhones go to the **App Store**, Android phones to **Google Play**, and everyone else to your fallback page, or to a page with both store buttons.

## Plain text {#text}
Shows a short message on the phone, up to 1,200 characters. Shorter text gives a less dense, easier-to-scan code.

## Email {#email}
Opens a new email with the recipient, subject and body filled in.

## SMS {#sms}
Starts a text message to a number, optionally with a prefilled message.

## Phone call {#phone}
Dials a number. Include the country code (for example `+1`) so it works abroad.
