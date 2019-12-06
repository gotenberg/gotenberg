package resource

import (
	"strings"
)

const (
	// RemoteURLCustomHeaderCanonicalBaseKey is the base key
	// of custom headers send to the remote URL.
	RemoteURLCustomHeaderCanonicalBaseKey string = "Gotenberg-Remoteurl-"
	// WebhookURLCustomHeaderCanonicalBaseKey is the base key
	// of custom headers send to the webhook URL.
	WebhookURLCustomHeaderCanonicalBaseKey string = "Gotenberg-Webhookurl-"
)

func fetchCustomHeaders(r Resource, baseKey string) map[string]string {
	customHeaders := make(map[string]string)
	for key, value := range r.customHeaders {
		if strings.Contains(key, baseKey) {
			realKey := strings.Replace(key, baseKey, "", 1)
			customHeaders[realKey] = value
		}
	}
	return customHeaders
}

// RemoteURLCustomHeaders is a helper for retrieving
// the custom headers for the URL conversion.
func RemoteURLCustomHeaders(r Resource) map[string]string {
	return fetchCustomHeaders(r, RemoteURLCustomHeaderCanonicalBaseKey)
}

// WebhookURLCustomHeaders is a helper for retrieving
// the custom headers for the webhook URL.
func WebhookURLCustomHeaders(r Resource) map[string]string {
	return fetchCustomHeaders(r, WebhookURLCustomHeaderCanonicalBaseKey)
}
