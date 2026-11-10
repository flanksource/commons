package http

import (
	"fmt"
	stdhttp "net/http"
)

// TransportWrapper is a RoundTripper that decorates another one, such as an
// instrumentation layer installed around http.DefaultTransport. Clients that
// need to reconfigure the concrete *http.Transport (TLS, proxy, keep-alives)
// unwrap to it, configure a clone, and rewrap the clone so the decoration
// survives.
type TransportWrapper interface {
	Unwrap() stdhttp.RoundTripper
	Rewrap(stdhttp.RoundTripper) stdhttp.RoundTripper
}

// unwrapTransport walks rt through TransportWrapper layers to the innermost
// *http.Transport and returns it, plus a rewrap func that re-applies the layers
// around a replacement, outermost layer outermost. Callers that modify the
// returned transport must Clone it first.
func unwrapTransport(rt stdhttp.RoundTripper) (*stdhttp.Transport, func(stdhttp.RoundTripper) stdhttp.RoundTripper, error) {
	var wrappers []TransportWrapper
	for {
		switch current := rt.(type) {
		case *stdhttp.Transport:
			return current, func(inner stdhttp.RoundTripper) stdhttp.RoundTripper {
				for i := len(wrappers) - 1; i >= 0; i-- {
					inner = wrappers[i].Rewrap(inner)
				}
				return inner
			}, nil
		case misconfiguredRoundTripper:
			return nil, nil, current.err
		case TransportWrapper:
			wrappers = append(wrappers, current)
			rt = current.Unwrap()
		default:
			return nil, nil, fmt.Errorf("transport %T does not unwrap to an *http.Transport: wrap it in a type that implements http.TransportWrapper", rt)
		}
	}
}
