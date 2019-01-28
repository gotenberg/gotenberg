---
title: Liveness
---

Gotenberg provides the endpoint `/ping` for checking the API availability with
a simple `GET` request.

This feature is especially useful for liveness/readiness probes in Kubernetes:

* [Pod lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
* [Configure Liveness and Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/)