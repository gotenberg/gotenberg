---
title: Ping
---

Gotenberg provides the endpoint `/ping` for checking the API availability with
a simple `GET` request.

This feature is especially useful for liveness/readiness probes in Kubernetes:

* [Pod lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
* [Configure Liveness and Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/)

If `LOG_LEVEL` is `"DEBUG"`, it also returns details about the PM2 processes in JSON format.
For instance:

```json
[
    {
        "name": "google-chrome-stable",
        "pm2_env": {
            "status": "online"
        },
        "monit": {
            "memory": 73826304,
            "cpu": 0
        }
    },
    {
        "name": "unoconv",
        "pm2_env": {
            "status": "online"
        },
        "monit": {
            "memory": 70914048,
            "cpu": 0
        }
    }
]
```
