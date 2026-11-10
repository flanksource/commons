package logger

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("redacting echoed authorization credentials", func(authorization, response, expected string) {
	Expect(RedactAuthorization(response, authorization)).To(Equal(expected))
},
	Entry("bearer token", "Bearer example-private-token", "denied example-private-token", "denied [REDACTED]"),
	Entry("basic password", "Basic YXBpOmV4YW1wbGUtcHJpdmF0ZS10b2tlbg==", "denied example-private-token", "denied [REDACTED]"),
	Entry("basic encoded credentials", "Basic YXBpOmV4YW1wbGUtcHJpdmF0ZS10b2tlbg==", "denied YXBpOmV4YW1wbGUtcHJpdmF0ZS10b2tlbg==", "denied [REDACTED]"),
	Entry("absent authorization", "", "permission denied", "permission denied"),
	Entry("single-character bearer token", "Bearer x", "max retries exceeded", "max retries exceeded"),
	Entry("single-character basic password", "Basic YXBpOmE=", "access denied for a user", "access denied for a user"),
	Entry("minimum-length basic password", "Basic YXBpOmFiY2Q=", "rejected abcd", "rejected [REDACTED]"),
)
