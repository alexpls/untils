---
title: Webhook notifications
sidebar_title: Webhooks
url: /notifications/webhooks
section: Notifications
description: Send monitor notifications to an HTTP endpoint as JSON.
last_updated: 17 May 2026
---

Webhook notifications send monitor changes to an HTTP endpoint as JSON. They can
be used to connect untils to chat tools, automation services, logging systems, or
custom applications.

## How webhook notifications work

Webhook targets are configured from **[Settings → Webhooks](/app/settings/webhook)**.
Add the HTTPS or HTTP URL that should receive notifications, then use **Test** to
send a test message.

When a monitor has webhook notifications enabled and new results are detected,
untils sends a webhook for each configured webhook target. Each delivery
is sent as an HTTP `POST` request containing a JSON body.

A delivery is treated as successful when the endpoint returns any HTTP status code
below `400`. Status codes `400` and above are treated as failures. Failed delivery
jobs are retried automatically, up to 10 attempts.

Webhook requests have a 15 second HTTP timeout. The receiving endpoint should respond
quickly so as to stay within the timeout.

## Payload reference

Webhook request payloads, attributes, examples, and delivery outcomes are documented
in the [Webhooks API reference](/docs/api/reference#webhooks).
