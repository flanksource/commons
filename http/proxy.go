package http

// configureProxy creates an HTTP proxy if the given config specifies one
// func configureProxy(config *Config) (func(*http.Request) (*url.URL, error), error) {
// 	if config.ProxyHost == "" || config.ProxyPort <= 0 {
// 		return nil, nil
// 	}

// 	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%d", config.ProxyHost, config.ProxyPort))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return http.ProxyURL(proxyURL), nil
// }
