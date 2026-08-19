// Package presigned recognises the pre-signed storage URLs the file upload uses.
//
// The API reserves space for a file and answers with a URL that authenticates
// itself; the contents then go straight to storage. That request rides the same
// HTTP client as every API call, so the middlewares have to tell it apart: it
// must not be rerouted, since the signature covers the host, its TLS is genuine,
// and neither its URL nor its body belongs in a log.
package presigned

import (
	"bytes"
	"net/url"
	"regexp"
)

// signatureParams are the query parameters that make such a URL usable on its
// own. Both the SigV4 and the older SigV2 names are listed, since which appears
// depends on how the installation's bucket is addressed.
var signatureParams = []string{
	"X-Amz-Signature",
	"X-Amz-Credential",
	"X-Amz-Security-Token",
	"Signature",
	"AWSAccessKeyId",
}

// signatureValue matches one of signatureParams and its value. The value class
// stops wherever a query parameter can end here: another parameter, the end of a
// JSON string, or whitespace. Casing is the signer's, so the pre-check can
// compare bytes without lowercasing a whole body.
var signatureValue = regexp.MustCompile(
	`([?&](?:X-Amz-Signature|X-Amz-Credential|X-Amz-Security-Token|Signature|AWSAccessKeyId)=)[^&"'\s\\]*`)

// IsURL reports whether u carries a storage signature, which is what separates
// the upload from every other request the engine sends: it goes to the storage
// service, not the Teamwork API.
func IsURL(u *url.URL) bool {
	if u == nil || u.RawQuery == "" {
		return false
	}
	query := u.Query()
	for _, param := range signatureParams {
		if query.Get(param) != "" {
			return true
		}
	}
	return false
}

// RedactSignatures replaces what authenticates a pre-signed URL wherever one
// appears in b: a request URL on its way to a log, or the JSON body that
// delivered it. The URL still identifies the upload afterwards.
func RedactSignatures(b []byte) []byte {
	if !containsSignatureParam(b) {
		return b
	}
	return signatureValue.ReplaceAll(b, []byte("${1}<redacted>"))
}

// containsSignatureParam reports whether b mentions any signature parameter, so
// the regex scan is skipped for the payloads that carry no pre-signed URL, which
// is almost all of them.
func containsSignatureParam(b []byte) bool {
	for _, param := range signatureParams {
		if bytes.Contains(b, []byte(param)) {
			return true
		}
	}
	return false
}
