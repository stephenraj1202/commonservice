package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// NewProxy returns a Gin handler that reverse-proxies requests to target.
// The path forwarded to the upstream is taken from the Gin wildcard param
// "*path", so the gateway prefix is automatically stripped.
// All request headers (including Authorization) are forwarded unchanged.
func NewProxy(target string) gin.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic("proxy: invalid target URL: " + err.Error())
	}

	return func(c *gin.Context) {
		// c.Param("path") contains the remaining path after the registered
		// prefix, e.g. "/upload" or "/jobs/1/pause".
		upstreamPath := c.Param("path")
		if upstreamPath == "" {
			upstreamPath = "/"
		}

		// Build the upstream URL.
		upstreamURL := *targetURL
		upstreamURL.Path = upstreamPath
		upstreamURL.RawQuery = c.Request.URL.RawQuery

		// Clone the incoming request and point it at the upstream.
		outReq := c.Request.Clone(c.Request.Context())
		outReq.URL = &upstreamURL
		outReq.Host = targetURL.Host
		// Remove RequestURI — it must not be set on outbound client requests.
		outReq.RequestURI = ""

		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				// Director is a no-op here; we already set everything above.
			},
		}

		rp.ServeHTTP(c.Writer, outReq)
	}
}
