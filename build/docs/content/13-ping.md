---
title: Ping
---

Gotenberg provides the endpoint `/ping` for checking the API availability with
a simple `GET` request.

If `LOG_LEVEL` is `"DEBUG"`, it also returns details about the PM2 processes in JSON format.
For instance:

```json
[
    {
        "name": "google-chrome-stable",
        "pm2_env": {
            "status": "online",
            "restart_time": 0
        },
        "monit": {
            "memory": 72294400,
            "cpu": 0
        }
    },
    {
        "name": "unoconv",
        "pm2_env": {
            "status": "online",
            "restart_time": 0
        },
        "monit": {
            "memory": 71000064,
            "cpu": 0
        }
    }
]
```

> if `LOG_LEVEL` is **not** `"DEBUG"`, no log entries are written.