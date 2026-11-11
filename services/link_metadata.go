package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type LinkMetadata struct {
	Description string `json:"description"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Icon        string `json:"icon"`
}

// Only public HTTP(S) destinations on standard ports are eligible for previews.
func validateMetadataURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("invalid public http/https URL")
	}
	if port := u.Port(); port != "" && !((u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80")) {
		return nil, errors.New("nonstandard port is not allowed")
	}
	if strings.ContainsAny(u.Hostname(), "%\\") {
		return nil, errors.New("invalid hostname")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !publicMetadataIP(ip) {
		return nil, errors.New("private address is not allowed")
	}
	return u, nil
}

var metadataBlockedRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("3fff::/20"), netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"),
}

func publicMetadataIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return false
	}
	// Public IPv6 is currently within 2000::/3; reject local/transition ranges.
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, prefix := range metadataBlockedRanges {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func metadataHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy:                  nil,
		TLSHandshakeTimeout:    4 * time.Second,
		ResponseHeaderTimeout:  5 * time.Second,
		MaxResponseHeaderBytes: 64 * 1024,
		DisableKeepAlives:      true,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, errors.New("no destination addresses")
			}
			// Validate all answers, then dial the checked IP directly to prevent DNS rebinding.
			for _, ip := range ips {
				if !publicMetadataIP(ip) {
					return nil, errors.New("non-public destination is not allowed")
				}
			}
			dialer := net.Dialer{Timeout: 4 * time.Second}
			var lastErr error
			for _, ip := range ips {
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
	}
	return &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 4 {
			return errors.New("too many redirects")
		}
		_, err := validateMetadataURL(req.URL.String())
		return err
	}}
}

func FetchLinkMetadata(ctx context.Context, raw string) (*LinkMetadata, error) {
	client := metadataHTTPClient()
	defer client.CloseIdleConnections()
	return fetchLinkMetadata(ctx, raw, client)
}

func fetchLinkMetadata(ctx context.Context, raw string, client *http.Client) (*LinkMetadata, error) {
	u, err := validateMetadataURL(raw)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Skyli-LinkPreview/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("cannot fetch link metadata")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("website returned HTTP %d", response.StatusCode)
	}
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml+xml") {
		return nil, errors.New("URL does not contain an HTML page")
	}
	// Parse at most the first MiB and stop at </head>/<body>; metadata from large
	// pages remains usable without downloading the rest of the document.
	reader := &metadataResponseReader{Reader: io.LimitReader(response.Body, 1024*1024)}
	result := parseLinkMetadata(reader, response.Request.URL)
	if reader.err != nil {
		return nil, errors.New("cannot read metadata response")
	}
	return result, nil
}

type metadataResponseReader struct {
	io.Reader
	err error
}

func (r *metadataResponseReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err != nil && err != io.EOF {
		r.err = err
	}
	return n, err
}

func parseLinkMetadata(reader io.Reader, base *url.URL) *LinkMetadata {
	result := &LinkMetadata{URL: base.String(), Title: base.Hostname(), Icon: base.ResolveReference(&url.URL{Path: "/favicon.ico"}).String()}
	tokenizer := html.NewTokenizer(reader)
	var title, ogTitle, icon, appleIcon, description, ogDescription string
	inTitle := false
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
			if token.Data == "body" {
				break
			}
			if token.Data == "title" {
				inTitle = true
			}
			attrs := map[string]string{}
			for _, attr := range token.Attr {
				attrs[strings.ToLower(attr.Key)] = attr.Val
			}
			if token.Data == "meta" && (strings.EqualFold(attrs["property"], "og:title") || strings.EqualFold(attrs["name"], "og:title")) {
				ogTitle = attrs["content"]
			}
			if token.Data == "meta" {
				if strings.EqualFold(attrs["property"], "og:description") || strings.EqualFold(attrs["name"], "og:description") {
					ogDescription = attrs["content"]
				} else if strings.EqualFold(attrs["name"], "description") {
					description = attrs["content"]
				}
			}
			if token.Data == "link" {
				rels := strings.Fields(strings.ToLower(attrs["rel"]))
				for _, rel := range rels {
					if rel == "icon" && icon == "" {
						icon = attrs["href"]
					}
					if (rel == "apple-touch-icon" || rel == "apple-touch-icon-precomposed") && appleIcon == "" {
						appleIcon = attrs["href"]
					}
				}
			}
		}
		if tokenType == html.TextToken && inTitle {
			title += token.Data
		}
		if tokenType == html.EndTagToken && token.Data == "title" {
			inTitle = false
		}
		if tokenType == html.EndTagToken && token.Data == "head" {
			break
		}
	}
	if strings.TrimSpace(ogTitle) != "" {
		title = ogTitle
	}
	title = strings.Join(strings.Fields(title), " ")
	if title != "" {
		result.Title = string([]rune(title)[:min(len([]rune(title)), 300)])
	}
	if strings.TrimSpace(ogDescription) != "" {
		description = ogDescription
	}
	description = strings.Join(strings.Fields(description), " ")
	runes := []rune(description)
	result.Description = string(runes[:min(len(runes), 1000)])
	if icon == "" {
		icon = appleIcon
	}
	if icon != "" {
		if reference, err := url.Parse(icon); err == nil {
			resolved := base.ResolveReference(reference)
			if _, err = validateMetadataURL(resolved.String()); err == nil {
				result.Icon = resolved.String()
			}
		}
	}
	return result
}
