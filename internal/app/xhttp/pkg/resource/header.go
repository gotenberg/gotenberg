package resource

import (
	"strings"
)

const (
	// RemoteURLCustomHTTPHeaderCanonicalBaseKey is the base key
	// of custom headers send to the remote URL.
	RemoteURLCustomHTTPHeaderCanonicalBaseKey string = "Gotenberg-Remoteurl-"
	// WebhookURLCustomHTTPHeaderCanonicalBaseKey is the base key
	// of custom headers send to the webhook URL.
	WebhookURLCustomHTTPHeaderCanonicalBaseKey string = "Gotenberg-Webhookurl-"
)

func fetchCustomHTTPHeaders(r Resource, baseKey string) map[string]string {
	customHeaders := make(map[string]string)
	for key, value := range r.customHeaders {
		if strings.Contains(key, baseKey) {
			realKey := strings.Replace(key, baseKey, "", 1)
			customHeaders[realKey] = value
		}
	}
	return customHeaders
}

// RemoteURLCustomHTTPHeaders is a helper for retrieving
// the custom headers for the URL conversion.
func RemoteURLCustomHTTPHeaders(r Resource) map[string]string {
	return fetchCustomHTTPHeaders(r, RemoteURLCustomHTTPHeaderCanonicalBaseKey)
}

// WebhookURLCustomHTTPHeaders is a helper for retrieving
// the custom headers for the webhook URL.
func WebhookURLCustomHTTPHeaders(r Resource) map[string]string {
	return fetchCustomHTTPHeaders(r, WebhookURLCustomHTTPHeaderCanonicalBaseKey)
}
