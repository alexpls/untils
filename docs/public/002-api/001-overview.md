---
title: API overview
sidebar_title: Overview
url: /api
section: API
description: Use the untils API to access monitor data as JSON.
last_updated: 17 May 2026
---

The untils API uses RPC-style endpoints over HTTP.
Endpoint names are verbs on resources, such as `/api/results.list_latest`.

Responses are always JSON and structured as:

```json
{
  "data": {}, // result data here, null on error
  "error": {} // error data here, null on success
}
```

Every resource type includes a `type` property which can be used to identify
the shape of the nested objects without relying on where they appear in the
response.

## Authorization

API requests use bearer token authorization:

```sh
Authorization: Bearer untils.api...
```

Tokens can be created and revoked from **[Settings → API tokens](/app/settings/api_tokens)**.
Store tokens as secrets. The full token is only shown once when it is created.

## Errors

Errors are returned as JSON with an `error` object:

```json
{
  "data": null,
  "error": {
    "code": "unauthorized",
    "message": "A valid API token is required."
  }
}
```

The `code` value is stable enough for programmatic handling. The `message` value
is intended for logs and debugging, and may change.

Common error codes include:

| Code                 | Description                                         |
| -------------------- | --------------------------------------------------- |
| `unauthorized`       | The request is missing a valid API token.           |
| `method_not_allowed` | The endpoint does not support the HTTP method used. |
| `not_found`          | The requested API path does not exist.              |
| `internal_error`     | An unexpected server error occurred.                |

## Reference

- The [API reference](/docs/api/reference) lists available methods and types.
- The [OpenAPI spec](https://github.com/alexpls/untils/blob/master/docs/public/openapi.yml)
  contains the machine-readable API definition, which you can use to generate API clients.
