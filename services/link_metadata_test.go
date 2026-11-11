package services

import (
	"context"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMetadataURLAndAddressPolicy(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "ftp://example.com", "https://user:pass@example.com/", "http://127.0.0.1/", "http://[::1]/", "http://[::ffff:127.0.0.1]/", "https://169.254.169.254/", "http://example.com:8080/", "https://example.com:80/", "https://[fe80::1%25eth0]/", "/relative"} {
		if _, err := validateMetadataURL(raw); err == nil {
			t.Errorf("accepted unsafe URL %q", raw)
		}
	}
	for _, raw := range []string{"https://example.com/path?a=1", "http://example.com:80/", "https://example.com:443/", "https://[2606:4700:4700::1111]/"} {
		if _, err := validateMetadataURL(raw); err != nil {
			t.Errorf("rejected valid URL %q: %v", raw, err)
		}
	}
	for _, ip := range []string{"0.0.0.0", "10.1.2.3", "127.0.0.1", "100.100.100.200", "169.254.1.1", "172.16.0.1", "192.168.1.1", "192.0.2.1", "192.88.99.1", "198.19.1.1", "224.0.0.1", "255.255.255.255", "::", "::1", "::ffff:10.0.0.1", "fc00::1", "fe80::1", "ff02::1", "2001:db8::1", "2001:20::1", "2002:7f00:1::", "64:ff9b::7f00:1", "3fff::1"} {
		if publicMetadataIP(netip.MustParseAddr(ip)) {
			t.Errorf("accepted non-public IP %s", ip)
		}
	}
	for _, ip := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if !publicMetadataIP(netip.MustParseAddr(ip)) {
			t.Errorf("rejected public IP %s", ip)
		}
	}
}

func TestMetadataParser(t *testing.T) {
	base, _ := url.Parse("https://example.com/articles/page?x=1")
	cases := []struct{ name, html, title, icon string }{
		{"open graph preferred", `<head><title>HTML &amp; title</title><meta property="og:title" content=" 网站   标题 &amp; 中文 "><link rel="shortcut icon" href="../images/favicon.svg"></head><body><title>ignored</title>`, "网站 标题 & 中文", "https://example.com/images/favicon.svg"},
		{"apple fallback", `<title>普通标题</title><link rel="apple-touch-icon" href="//cdn.example.com/icon.png">`, "普通标题", "https://cdn.example.com/icon.png"},
		{"hostname and favicon fallback", `<html><body>No metadata</body></html>`, "example.com", "https://example.com/favicon.ico"},
		{"unsafe icon", `<title>Safe</title><link rel="icon" href="javascript:alert(1)">`, "Safe", "https://example.com/favicon.ico"},
		{"loopback icon", `<title>Safe</title><link rel="icon" href="http://127.0.0.1/icon">`, "Safe", "https://example.com/favicon.ico"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseLinkMetadata(strings.NewReader(tc.html), base)
			if result.Title != tc.title || result.Icon != tc.icon || result.URL != base.String() {
				t.Fatalf("unexpected metadata: %+v", result)
			}
		})
	}
	result := parseLinkMetadata(strings.NewReader("<title>"+strings.Repeat("中", 500)+"</title>"), base)
	if len([]rune(result.Title)) != 300 {
		t.Fatal("title limit must count Unicode characters")
	}
}

type metadataRoundTrip func(*http.Request) (*http.Response, error)

func (f metadataRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestFetchMetadataLimitsAndHeaders(t *testing.T) {
	for _, tc := range []struct {
		name              string
		status            int
		contentType, body string
		wantError         bool
	}{
		{"valid", 200, "text/html; charset=utf-8", "<title>Metadata</title>", false},
		{"not HTML", 200, "application/json", "{}", true},
		{"not success", 403, "text/html", "Denied", true},
		{"large body keeps head metadata", 200, "text/html", "<head><title>Metadata</title></head><body>" + strings.Repeat("x", 1024*1024+1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" {
					t.Fatal("credentials leaked to metadata destination")
				}
				if req.Header.Get("User-Agent") == "" {
					t.Fatal("missing metadata user agent")
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": []string{tc.contentType}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
			})}
			result, err := fetchLinkMetadata(context.Background(), "https://example.com/path", client)
			if (err != nil) != tc.wantError {
				t.Fatalf("result %+v, error %v", result, err)
			}
			if !tc.wantError && result.Title != "Metadata" {
				t.Fatalf("wrong title %+v", result)
			}
		})
	}
}

func TestMetadataTransportAndRedirectPolicy(t *testing.T) {
	client := metadataHTTPClient()
	transport := client.Transport.(*http.Transport)
	if transport.Proxy != nil || client.Timeout != 8*time.Second || transport.DialContext == nil {
		t.Fatal("metadata client must use bounded direct connections")
	}
	if _, err := transport.DialContext(context.Background(), "tcp", "127.0.0.1:80"); err == nil {
		t.Fatal("transport connected to private address")
	}
	for _, raw := range []string{"http://127.0.0.1/admin", "file:///etc/passwd", "https://example.com:8080"} {
		u, _ := url.Parse(raw)
		if err := client.CheckRedirect(&http.Request{URL: u}, []*http.Request{{}}); err == nil {
			t.Errorf("allowed unsafe redirect to %s", raw)
		}
	}
	u, _ := url.Parse("https://example.com/next")
	if err := client.CheckRedirect(&http.Request{URL: u}, make([]*http.Request, 3)); err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(&http.Request{URL: u}, make([]*http.Request, 4)); err == nil {
		t.Fatal("allowed fourth redirect")
	}
}

func TestMetadataDescription(t *testing.T) {
	base, _ := url.Parse("https://example.com/page")
	for _, tc := range []struct{ name, html, want string }{
		{"Open Graph preferred", `<head><meta name="description" content="Default description"><meta property="og:description" content=" OG   中文 &amp; 摘要 "></head>`, "OG 中文 & 摘要"},
		{"standard fallback", `<head><meta name="DESCRIPTION" content=" 普通   页面摘要 "></head>`, "普通 页面摘要"},
		{"empty OG fallback", `<head><meta name="description" content="Default description"><meta property="og:description" content="   "></head>`, "Default description"},
		{"no description", `<head><title>Example</title></head>`, ""},
		{"ignore body metadata", `<head></head><body><meta name="description" content="Ignored">`, ""},
		{"character limit", `<head><meta name="description" content="` + strings.Repeat("中", 1001) + `"></head>`, strings.Repeat("中", 1000)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := parseLinkMetadata(strings.NewReader(tc.html), base)
			if result.Description != tc.want {
				t.Fatalf("unexpected description %q", result.Description)
			}
		})
	}
}
