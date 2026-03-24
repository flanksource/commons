package middlewares

import (
	"fmt"
	"net/http"
	"time"
)

func NewAccessLog() Middleware {
	return func(rt http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			start := time.Now()
			resp, err := rt.RoundTrip(req)
			elapsed := time.Since(start)
			if err != nil {
				fmt.Printf("%s %s error %s\n", req.Method, req.URL, elapsed.Truncate(time.Millisecond))
				return nil, err
			}
			fmt.Printf("%s %s %d %s\n", req.Method, req.URL, resp.StatusCode, elapsed.Truncate(time.Millisecond))
			return resp, nil
		})
	}
}
