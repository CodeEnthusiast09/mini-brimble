package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const defaultServerID = "srv0"

type Caddy struct {
	adminURL string
	client   *http.Client
}

func New(adminURL string) *Caddy {
	return &Caddy{
		adminURL: strings.TrimRight(adminURL, "/"),
		client:   &http.Client{},
	}
}

func (c *Caddy) AddRoute(ctx context.Context, host string, upstreamHost string, upstreamPort int) error {
	body, err := json.Marshal(route{
		ID: routeID(host),
		Match: []match{
			{Host: []string{host}},
		},
		Handle: []handle{
			{
				Handler: "reverse_proxy",
				Upstreams: []upstream{
					{Dial: fmt.Sprintf("%s:%d", upstreamHost, upstreamPort)},
				},
			},
		},
		Terminal: true,
	})
	if err != nil {
		return fmt.Errorf("marshal caddy route: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint("/config/apps/http/servers/"+defaultServerID+"/routes"),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create add route request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("add caddy route for %q: %w", host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("add caddy route for %q: %s: %s", host, resp.Status, readBody(resp.Body))
	}

	return nil
}

func (c *Caddy) RemoveRoute(ctx context.Context, host string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		c.endpoint("/id/"+routeID(host)),
		nil,
	)
	if err != nil {
		return fmt.Errorf("create remove route request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("remove caddy route for %q: %w", host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remove caddy route for %q: %s: %s", host, resp.Status, readBody(resp.Body))
	}

	return nil
}

func (c *Caddy) endpoint(path string) string {
	u, err := url.JoinPath(c.adminURL, path)
	if err != nil {
		return c.adminURL + path
	}
	return u
}

func routeID(host string) string {
	replacer := strings.NewReplacer(".", "-", ":", "-", "/", "-", "@", "-", "*", "wildcard")
	return "route-" + replacer.Replace(host)
}

func readBody(r io.Reader) string {
	body, err := io.ReadAll(r)
	if err != nil {
		return "unable to read response body"
	}

	return strings.TrimSpace(string(body))
}

type route struct {
	ID       string   `json:"@id"`
	Match    []match  `json:"match"`
	Handle   []handle `json:"handle"`
	Terminal bool     `json:"terminal"`
}

type match struct {
	Host []string `json:"host"`
}

type handle struct {
	Handler   string     `json:"handler"`
	Upstreams []upstream `json:"upstreams"`
}

type upstream struct {
	Dial string `json:"dial"`
}
