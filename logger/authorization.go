package logger

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// minRedactedCredentialLength is the shortest credential replaced wherever it
// appears in free text. Shorter credentials are replaced only as standalone
// tokens, so they do not corrupt the surrounding words.
const minRedactedCredentialLength = 4

const redactedCredential = "[REDACTED]"

// RedactAuthorization removes credentials echoed outside recognized secret
// fields, including decoded Basic-auth passwords in HTTP error responses, in
// both raw and JSON-escaped form.
func RedactAuthorization(text, authorization string) string {
	parts := strings.Fields(authorization)
	if len(parts) == 0 {
		return text
	}
	credentials := []string{parts[len(parts)-1]}
	if len(parts) == 2 && strings.EqualFold(parts[0], "basic") {
		if decoded, err := base64.StdEncoding.DecodeString(credentials[0]); err == nil {
			credentials = append(credentials, string(decoded))
			if _, password, ok := strings.Cut(string(decoded), ":"); ok {
				credentials = append(credentials, password)
			}
		}
	}
	if json.Valid([]byte(text)) {
		return redactJSONCredentials(text, credentials)
	}
	for _, credential := range credentials {
		for _, form := range serializedCredentialForms(credential) {
			text = redactCredential(text, form)
		}
	}
	return text
}

// redactJSONCredentials redacts credentials inside decoded string values and
// bare tokens without touching delimiters, so the result stays valid JSON and
// structured secret scrubbing still applies to it.
func redactJSONCredentials(text string, credentials []string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		if text[i] == '"' {
			end := jsonStringEnd(text, i)
			out.WriteString(redactJSONString(text[i:end], credentials))
			i = end
			continue
		}
		start := i
		for i < len(text) && text[i] != '"' && !isJSONSeparator(text[i]) {
			i++
		}
		if start == i {
			out.WriteByte(text[i])
			i++
			continue
		}
		token := text[start:i]
		if redactAll(token, credentials) != token {
			out.WriteString(strconv.Quote(redactedCredential))
		} else {
			out.WriteString(token)
		}
	}
	return out.String()
}

// jsonStringEnd returns the index just past the closing quote of the string
// literal starting at text[start]; text must be valid JSON.
func jsonStringEnd(text string, start int) int {
	for i := start + 1; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	panic(fmt.Errorf("unterminated JSON string at offset %d", start))
}

func redactJSONString(literal string, credentials []string) string {
	var decoded string
	if err := json.Unmarshal([]byte(literal), &decoded); err != nil {
		panic(fmt.Errorf("decode JSON string %s: %w", literal, err))
	}
	redacted := redactAll(decoded, credentials)
	if redacted == decoded {
		return literal
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(redacted); err != nil {
		panic(fmt.Errorf("JSON-encode redacted string: %w", err))
	}
	return strings.TrimSuffix(encoded.String(), "\n")
}

func redactAll(text string, credentials []string) string {
	for _, credential := range credentials {
		text = redactCredential(text, credential)
	}
	return text
}

func isJSONSeparator(b byte) bool {
	return strings.IndexByte("{}[]:, \t\r\n", b) >= 0
}

// serializedCredentialForms returns the credential as it can appear in text:
// raw, and escaped inside a JSON string with and without HTML escaping.
func serializedCredentialForms(credential string) []string {
	forms := []string{credential}
	for _, escapeHTML := range []bool{true, false} {
		var encoded bytes.Buffer
		encoder := json.NewEncoder(&encoded)
		encoder.SetEscapeHTML(escapeHTML)
		if err := encoder.Encode(credential); err != nil {
			panic(fmt.Errorf("JSON-encode credential string: %w", err))
		}
		quoted := strings.TrimSuffix(encoded.String(), "\n")
		if escaped := quoted[1 : len(quoted)-1]; !slices.Contains(forms, escaped) {
			forms = append(forms, escaped)
		}
	}
	return forms
}

func redactCredential(text, credential string) string {
	if credential == "" {
		return text
	}
	if len(credential) >= minRedactedCredentialLength {
		return strings.ReplaceAll(text, credential, redactedCredential)
	}
	var redacted strings.Builder
	written := 0
	for offset := 0; offset < len(text); {
		index := strings.Index(text[offset:], credential)
		if index < 0 {
			break
		}
		start, end := offset+index, offset+index+len(credential)
		if (start == 0 || !isWordByte(text[start-1])) && (end == len(text) || !isWordByte(text[end])) {
			redacted.WriteString(text[written:start])
			redacted.WriteString(redactedCredential)
			written, offset = end, end
		} else {
			offset = start + 1
		}
	}
	redacted.WriteString(text[written:])
	return redacted.String()
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}
