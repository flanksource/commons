package logger

import (
	"encoding/base64"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func basicAuthorization(userPassword string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(userPassword))
}

var _ = DescribeTable("redacting echoed authorization credentials", func(authorization, response, expected string) {
	Expect(RedactAuthorization(response, authorization)).To(Equal(expected))
},
	Entry("bearer token", "Bearer example-private-token", "denied example-private-token", "denied [REDACTED]"),
	Entry("basic password", basicAuthorization("api:example-private-token"), "denied example-private-token", "denied [REDACTED]"),
	Entry("basic encoded credentials", basicAuthorization("api:example-private-token"), "denied YXBpOmV4YW1wbGUtcHJpdmF0ZS10b2tlbg==", "denied [REDACTED]"),
	Entry("absent authorization", "", "permission denied", "permission denied"),
	Entry("empty basic password", basicAuthorization("api:"), "permission denied", "permission denied"),
	Entry("single-character bearer token inside words", "Bearer x", "max retries exceeded", "max retries exceeded"),
	Entry("single-character basic password echoed as a word", basicAuthorization("api:a"), "access denied for a user", "access denied for [REDACTED] user"),
	Entry("short basic password echoed after a delimiter", basicAuthorization("api:abc"), `{"password":"abc"}`, `{"password":"[REDACTED]"}`),
	Entry("short bearer token echoed at the end", "Bearer abc", "rejected token abc", "rejected token [REDACTED]"),
	Entry("short basic password inside a longer word", basicAuthorization("api:abc"), "abcdef failed", "abcdef failed"),
	Entry("minimum-length basic password", basicAuthorization("api:abcd"), "rejected abcd", "rejected [REDACTED]"),
	Entry("JSON-escaped quote in basic password", basicAuthorization(`api:abc"def`), `{"error":"bad password abc\"def"}`, `{"error":"bad password [REDACTED]"}`),
	Entry("JSON-escaped backslash in bearer token", `Bearer to\ken-value`, `{"token":"to\\ken-value"}`, `{"token":"[REDACTED]"}`),
	Entry("JSON HTML-escaped basic password", basicAuthorization("api:p<ss>word"), `{"error":"bad password p<ss>word"}`, `{"error":"bad password [REDACTED]"}`),
	Entry("JSON delimiter as the whole basic password", basicAuthorization(`api:"`), `{"echo":"\"","password":"other-secret"}`, `{"echo":"[REDACTED]","password":"other-secret"}`),
	Entry("JSON delimiter inside a longer basic password", basicAuthorization(`api:ab"cd`), `{"echo":"ab\"cd","other":"x"}`, `{"echo":"[REDACTED]","other":"x"}`),
	Entry("JSON backslash as the whole bearer token", `Bearer \`, `{"a":"\\","b":"c"}`, `{"a":"[REDACTED]","b":"c"}`),
	Entry("JSON numeric credential", "Bearer 12345678", `{"pin": 12345678, "ok": 1}`, `{"pin": "[REDACTED]", "ok": 1}`),
	Entry("JSON credential used as an object key", "Bearer example-private-token", `{"example-private-token":true}`, `{"[REDACTED]":true}`),
	Entry("JSON whitespace and nesting preserved", "Bearer example-private-token", "{\n  \"a\": [\"example-private-token\", {\"b\": null}]\n}", "{\n  \"a\": [\"[REDACTED]\", {\"b\": null}]\n}"),
)

var _ = It("keeps redacted JSON parseable", func() {
	redacted := RedactAuthorization(`{"echo":"\"","password":"other-secret"}`, basicAuthorization(`api:"`))
	Expect(json.Valid([]byte(redacted))).To(BeTrue())
})
