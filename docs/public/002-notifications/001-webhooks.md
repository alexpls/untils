---
title: Webhook notifications
sidebar_title: Webhooks
url: /notifications/webhooks
section: Notifications
description: Send monitor notifications to an HTTP endpoint as JSON.
last_updated: 25 April 2026
---

Webhook notifications send monitor changes to an HTTP endpoint as JSON. They can
be used to connect untils to chat tools, automation services, logging systems, or
custom applications.

## How webhook notifications work

Webhook targets are configured from **Settings → Webhooks**. Add the HTTPS or HTTP
URL that should receive notifications, then use **Test** to send a test message.

When a monitor has webhook notifications enabled and new results are detected,
untils sends a webhook for each configured webhook target. Each delivery
is sent as an HTTP `POST` request containing a JSON body.

A delivery is treated as successful when the endpoint returns any HTTP status code
below `400`. Status codes `400` and above are treated as failures. Failed delivery
jobs are retried automatically, up to 10 attempts.

Webhook requests have a 15 second HTTP timeout. The receiving endpoint should respond
quickly so as to stay within the timeout.

## Test messages

The **Test** action in webhook settings sends this JSON payload:

```json
{
  "type": "webhook_message",
  "message": {
    "type": "test",
    "hello_world": "Glad you're here"
  }
}
```

## Monitor change messages

Monitor change notifications use this top-level shape:

```json
{
  "type": "webhook_message",
  "message": {
    "type": "new_results",
    "monitor": {
      "type": "monitor",
      "id": 42,
      "subject": "Example monitor"
    },
    "new_results": [],
    "old_result": {}
  }
}
```

### Fields

| Field                     | Type   | Description                                                                                            |
| ------------------------- | ------ | ------------------------------------------------------------------------------------------------------ |
| `type`                    | string | Always `webhook_message` for webhook payloads.                                                         |
| `message.type`            | string | The message kind. Monitor change notifications use `new_results`.                                      |
| `message.monitor.type`    | string | Always `monitor`.                                                                                      |
| `message.monitor.id`      | number | The untils monitor ID.                                                                                 |
| `message.monitor.subject` | string | The monitor subject shown in untils.                                                                   |
| `message.new_results`     | array  | One or more newly detected results.                                                                    |
| `message.old_result`      | object | The previous visible result for comparison. If there was no previous result, the headline is `(none)`. |

Each result in `new_results` and `old_result` uses this shape:

| Field      | Type   | Description                          |
| ---------- | ------ | ------------------------------------ |
| `type`     | string | Always `result`.                     |
| `id`       | number | The untils result ID.                |
| `headline` | string | The rendered result headline.        |
| `subtitle` | string | The rendered result subtitle.        |
| `fields`   | array  | The extracted fields for the result. |

Each item in `fields` uses this shape:

| Field   | Type   | Description                                |
| ------- | ------ | ------------------------------------------ |
| `type`  | string | Always `result_field`.                     |
| `name`  | string | The field name configured for the monitor. |
| `value` | string | The extracted value as text.               |

## Example monitor change message

```json
{
  "type": "webhook_message",
  "message": {
    "type": "new_results",
    "monitor": {
      "type": "monitor",
      "id": 42,
      "subject": "Example monitor"
    },
    "new_results": [
      {
        "type": "result",
        "id": 101,
        "headline": "New value",
        "subtitle": "Released at https://example.com/new",
        "fields": [
          {
            "type": "result_field",
            "name": "Title",
            "value": "New value"
          },
          {
            "type": "result_field",
            "name": "Link",
            "value": "https://example.com/new"
          }
        ]
      }
    ],
    "old_result": {
      "type": "result",
      "id": 100,
      "headline": "Old value",
      "subtitle": "",
      "fields": []
    }
  }
}
```
