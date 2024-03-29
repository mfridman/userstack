package userstack

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	secureURL = "https://api.userstack.com/detect"

	// this is sad, but here we are. See NewClient for more info.
	unsecureURL = "http://api.userstack.com/detect"
)

// defaultClient returns an http client with sane defaults. Users can
// instantiate a NewClient with their own http handler, but this should
// suffice for more use-cases.
func defaultClient() *http.Client {
	client := http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,

			ExpectContinueTimeout: 10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &client
}

type Client struct {
	client HTTPClient
	apiKey string
	url    string
}

type Stack struct {
	Ua    string     `json:"ua"`
	Type  EntityType `json:"type"`
	Brand string     `json:"brand"` // Is this device.brand ?
	Name  string     `json:"name"`  // Is this device.name ?
	URL   string     `json:"url"`
	Os    struct {
		Name         string `json:"name"`
		Code         string `json:"code"`
		URL          string `json:"url"`
		Family       string `json:"family"`
		FamilyCode   string `json:"family_code"`
		FamilyVendor string `json:"family_vendor"`
		Icon         string `json:"icon"`
		IconLarge    string `json:"icon_large"`
	} `json:"os"`
	Device struct {
		IsMobileDevice bool       `json:"is_mobile_device"`
		Type           DeviceType `json:"type"`
		Brand          string     `json:"brand"`
		BrandCode      string     `json:"brand_code"`
		BrandURL       string     `json:"brand_url"`
		Name           string     `json:"name"`
	} `json:"device"`
	Browser struct {
		Name         string `json:"name"`
		Version      string `json:"version"`
		VersionMajor string `json:"version_major"`
		Engine       string `json:"engine"`
	} `json:"browser"`
	Crawler struct {
		IsCrawler bool         `json:"is_crawler"`
		Category  CategoryType `json:"category"`
		LastSeen  interface{}  `json:"last_seen"` // TODO(mf): find out the type of this. string?
	} `json:"crawler"`

	ApiErr
}

type ApiErr struct {
	Success bool `json:"success,omitempty"`
	Err     struct {
		Code int    `json:"code,omitempty"`
		Type string `json:"type,omitempty"`
		Info string `json:"info,omitempty"`
	} `json:"error,omitempty"`
}

func (e ApiErr) Error() string {
	return fmt.Sprintf("success: %t err: %+v", e.Success, e.Err)
}

func (c *Client) Detect(userAgent string) (*Stack, error) {
	req, err := http.NewRequest(http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("access_key", c.apiKey)
	q.Add("ua", userAgent)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var st *Stack
	if err := c.decode(resp.Body, &st); err != nil {
		return nil, err
	}

	if !st.Success && st.Ua == "" {
		return nil, st.ApiErr
	}

	return st, nil
}

func (c *Client) decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient returns a userstack client. If nil http client is supplied
// a default client with sane defaults is used.
//
// Note: if you have a non-paying account, you must specify secure: false. Only paid accounts
// get access to `https`.
func NewClient(apiKey string, client HTTPClient, secure bool) (*Client, error) {
	if apiKey == "" {
		e := ApiErr{Success: false}
		e.Err.Code = 101
		e.Err.Type = "missing_access_key"
		e.Err.Info = "User did not supply an access key."
		return nil, e
	}
	c := Client{
		apiKey: apiKey,
		client: client,
		url:    secureURL,
	}
	if !secure {
		c.url = unsecureURL
	}
	if c.client == nil {
		c.client = defaultClient()
	}

	return &c, nil
}

type EntityType int

const (
	UnknownEntity EntityType = iota
	Browser
	MobileBrowser
	EmailClient
	App
	FeedReader
	Crawler
	OfflineBrowser
)

var entities = []string{
	"unknown",
	"browser",
	"mobile-browser",
	"email-client",
	"app",
	"feed-reader",
	"crawler",
	"offline-browser",
}

func (e EntityType) String() string {
	return entities[e]
}

// MarshalText satisfies TextMarshaler
func (e EntityType) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (e *EntityType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(entities); i++ {
		if enum == entities[i] {
			*e = EntityType(i)
			return nil
		}
	}
	return fmt.Errorf("unknown entity type: %s", enum)
}

type DeviceType int

const (
	UnknownDevice DeviceType = iota
	Desktop
	Tablet
	Smartphone
	Console
	Smarttv
	Wearable
)

var devices = []string{
	"unknown",
	"desktop",
	"tablet",
	"smartphone",
	"console",
	"smarttv",
	"wearable",
}

func (d DeviceType) String() string {
	return devices[d]
}

// MarshalText satisfies TextMarshaler
func (d DeviceType) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (d *DeviceType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(devices); i++ {
		if enum == devices[i] {
			*d = DeviceType(i)
			return nil
		}
	}
	return fmt.Errorf("unknown device type: %s", enum)
}

type CategoryType int

const (
	UnknownCategory CategoryType = iota
	SearchEngine
	Monitoring
	ScreenshotService
	Scraper
	SecurityScanner
)

var categories = []string{
	"unknown",
	"search-engine",
	"monitoring",
	"screenshot-service",
	"scraper",
	"security-scanner",
}

func (c CategoryType) String() string {
	return categories[c]
}

// MarshalText satisfies TextMarshaler
func (c CategoryType) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (c *CategoryType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(categories); i++ {
		if enum == categories[i] {
			*c = CategoryType(i)
			return nil
		}
	}
	return fmt.Errorf("unknown category type: %s", enum)
}
