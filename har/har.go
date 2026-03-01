// Package har provides HAR 1.2 types and an HTTP client middleware for
// capturing outbound request/response pairs for troubleshooting.
package har

const defaultMaxBodySize = 64 * 1024 // 64 KB

// HARConfig controls what the HAR middleware captures and how it redacts.
type HARConfig struct {
	// MaxBodySize is the maximum number of bytes captured per body.
	// Bodies exceeding this are truncated and Content.Truncated is set to true.
	// Default: 65536 (64 KB).
	MaxBodySize int64

	// CaptureContentTypes lists MIME type prefixes for which body capture is enabled.
	// Default: ["application/json", "application/x-www-form-urlencoded"].
	CaptureContentTypes []string

	// RedactedHeaders lists additional header name glob patterns to redact,
	// on top of logger.CommonRedactedHeaders.
	RedactedHeaders []string
}

// DefaultConfig returns a HARConfig with sensible defaults.
func DefaultConfig() HARConfig {
	return HARConfig{
		MaxBodySize:         defaultMaxBodySize,
		CaptureContentTypes: []string{"application/json", "application/x-www-form-urlencoded"},
	}
}

// File is the outermost HAR 1.2 envelope: {"log": {...}}.
// Use this when writing .har files for import into browser DevTools.
type File struct {
	Log Log `json:"log"`
}

// Log is the top-level HAR 1.2 container.
type Log struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Pages   []Page  `json:"pages"`
	Entries []Entry `json:"entries"`
}

// Page is included for HAR 1.2 spec compliance; hx leaves it empty.
type Page struct {
	StartedDateTime string      `json:"startedDateTime"`
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	PageTimings     PageTimings `json:"pageTimings"`
}

// PageTimings holds page-level timing data (unused by hx, present for spec compliance).
type PageTimings struct {
	OnLoad int `json:"onLoad,omitempty"`
}

// Creator identifies the application that created the HAR log.
type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Entry represents a single HTTP request/response pair.
type Entry struct {
	StartedDateTime string   `json:"startedDateTime"`
	Time            float64  `json:"time"`
	Request         Request  `json:"request"`
	Response        Response `json:"response"`
	Cache           Cache    `json:"cache"`
	Timings         Timings  `json:"timings"`
}

// Cache holds cache information for an entry (required by spec; left empty by hx).
type Cache struct{}

// Request holds HAR request data.
type Request struct {
	Method      string        `json:"method"`
	URL         string        `json:"url"`
	HTTPVersion string        `json:"httpVersion"`
	Cookies     []Cookie      `json:"cookies"`
	Headers     []Header      `json:"headers"`
	QueryString []QueryString `json:"queryString"`
	PostData    *PostData     `json:"postData,omitempty"`
	HeadersSize int           `json:"headersSize"`
	BodySize    int64         `json:"bodySize"`
}

// Response holds HAR response data.
type Response struct {
	Status      int      `json:"status"`
	StatusText  string   `json:"statusText"`
	HTTPVersion string   `json:"httpVersion"`
	Cookies     []Cookie `json:"cookies"`
	Headers     []Header `json:"headers"`
	Content     Content  `json:"content"`
	RedirectURL string   `json:"redirectURL"`
	HeadersSize int      `json:"headersSize"`
	BodySize    int64    `json:"bodySize"`
}

// Cookie is a name/value pair from a Cookie or Set-Cookie header.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Content holds the response body details.
type Content struct {
	Size      int64  `json:"size"`
	MimeType  string `json:"mimeType,omitempty"`
	Text      string `json:"text,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

// PostData holds the request body details.
type PostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// Header is a name/value pair.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// QueryString is a name/value pair from the URL query string.
type QueryString struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Timings records durations (in milliseconds) for the request lifecycle.
type Timings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}
