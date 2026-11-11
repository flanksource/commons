package logger

import (
	"encoding/base64"

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
)
