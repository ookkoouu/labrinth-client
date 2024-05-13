package labrinth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

const (
	APIBaseURL = "https://api.modrinth.com/v2"

	headerRateLimit     = "X-Ratelimit-Limit"
	headerRateRemaining = "X-Ratelimit-Remaining"
	headerRateReset     = "X-Ratelimit-Reset"
)

var (
	appVersion       = "dev"
	defaultUserAgent = "github.com/ookkoouu/labrinth-client/" + appVersion

	supportedImageExt = []string{
		"png",
		"jpg",
		"jpeg",
		"bmp",
		"gif",
		"webp",
		"svg",
		"svgz",
		"rgb",
	}
)

type JSONMarshaler = func(v interface{}) ([]byte, error)
type JSONUnmarshaler = func(data []byte, v interface{}) error

type service struct {
	client *Client
}

// TODO: implement services
type NotificationsService service
type MiscService service
type TagsService service
type TeamsService service
type ThreadsService service
type UsersService service
type VersionFilesService service
type VersionsService service

type Client struct {
	hc        *http.Client
	BaseURL   *neturl.URL
	UserAgent string
	AuthToken string
	JSONMarshaler
	JSONUnmarshaler

	common service

	Notifications *NotificationsService
	Projects      *ProjectsService
	Misc    *MiscService
	Tags          *TagsService
	Teams         *TeamsService
	Threads       *ThreadsService
	Users         *UsersService
	VersionFiles  *VersionFilesService
	Versions      *VersionsService
}

func NewClient() *Client {
	c := &Client{}
	c.init()
	return c
}

func (c *Client) init() {
	if c.hc == nil {
		c.hc = createClient()
	}
	if c.BaseURL == nil {
		c.BaseURL, _ = neturl.ParseRequestURI(APIBaseURL)
	}
	if c.UserAgent == "" {
		c.UserAgent = defaultUserAgent
	}

	c.common.client = c
	c.Notifications = (*NotificationsService)(&c.common)
	c.Projects = (*ProjectsService)(&c.common)
	c.Misc = (*MiscService)(&c.common)
	c.Tags = (*TagsService)(&c.common)
	c.Teams = (*TeamsService)(&c.common)
	c.Threads = (*ThreadsService)(&c.common)
	c.Users = (*UsersService)(&c.common)
	c.VersionFiles = (*VersionFilesService)(&c.common)
	c.Versions = (*VersionsService)(&c.common)
}

type RequestOption func(req *http.Request)

func (c *Client) NewRequest(method, path string, body any, opts ...RequestOption) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	var rd *bytes.Reader
	if body != nil {
		data, err := c.JSONMarshaler(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, u.String(), rd)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", c.AuthToken)
	}

	for _, opt := range opts {
		opt(req)
	}

	return req, nil
}

func (c *Client) NewFormRequest(method, path string, body io.Reader, opts ...RequestOption) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "multipart/form-data")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", c.AuthToken)
	}

	for _, opt := range opts {
		opt(req)
	}

	return req, nil
}

func (c *Client) NewUploadRequest(method, path, contentType string, body io.Reader, opts ...RequestOption) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", c.AuthToken)
	}

	for _, opt := range opts {
		opt(req)
	}

	return req, nil
}

type ErrorResponse struct {
	Response    *http.Response `json:"-"`
	Code        string         `json:"error"`
	Description string         `json:"description"`
}

func (r *ErrorResponse) Error() string {
	if r.Response != nil && r.Response.Request != nil {
		return fmt.Sprintf("%v %v: %d %v",
			r.Response.Request.Method, r.Response.Request.URL.String(),
			r.Response.StatusCode, r.Description)
	}
	if r.Response != nil {
		return fmt.Sprintf("%d %v", r.Response.StatusCode, r.Description)
	}
	return fmt.Sprintf("%v", r.Description)
}

type Rate struct {
	// Maximum number of requests that can be made in a minute
	Limit int
	// Number of requests remaining in the current ratelimit window
	Remaining int
	// Time in seconds until the ratelimit window resets
	Reset int
}

type Response struct {
	*http.Response
	Rate Rate
	Data any
}

func (c *Client) checkResponse(r *http.Response) error {
	if 200 <= r.StatusCode && r.StatusCode <= 299 {
		return nil
	}

	errResp := &ErrorResponse{Response: r}
	if 500 <= r.StatusCode {
		errResp.Description = "internal server error"
		return errResp
	}

	data, err := io.ReadAll(r.Body)
	if err == nil && data != nil {
		err = c.JSONUnmarshaler(data, errResp)
		if err != nil {
			errResp = &ErrorResponse{Response: r}
		}
	}

	return errResp
}

func parseRate(r *http.Response) Rate {
	var rate Rate
	if limit := r.Header.Get(headerRateLimit); limit != "" {
		rate.Limit, _ = strconv.Atoi(limit)
	}
	if remaining := r.Header.Get(headerRateRemaining); remaining != "" {
		rate.Remaining, _ = strconv.Atoi(remaining)
	}
	if reset := r.Header.Get(headerRateReset); reset != "" {
		rate.Reset, _ = strconv.Atoi(reset)
	}
	return rate

}

func (c *Client) Do(ctx context.Context, req *http.Request, respData any) (*Response, error) {
	req = req.WithContext(ctx)
	res, err := c.hc.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		return nil, err
	}

	defer res.Body.Close()

	response := &Response{
		Response: res,
		Rate:     parseRate(res),
		Data:     respData,
	}

	err = c.checkResponse(res)
	if err != nil {
		return response, err
	}

	bodyData, err := io.ReadAll(res.Body)
	if err != nil {
		return response, err
	}

	err = c.JSONUnmarshaler(bodyData, response.Data)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (c *Client) SetHTTPClient(hc *http.Client) *Client {
	c.hc = hc
	return c
}

func (c *Client) SetToken(token string) *Client {
	c.AuthToken = token
	return c
}

func (c *Client) SetBaseURL(url string) *Client {
	u, _ := neturl.Parse(url)
	c.BaseURL = u
	return c
}

func (c *Client) SetUserAgent(ua string) *Client {
	c.UserAgent = ua
	return c
}

func (c *Client) SetJSONMarshaler(marshaler JSONMarshaler) *Client {
	c.JSONMarshaler = marshaler
	return c
}

func (c *Client) SetJSONUnmarshaler(unmarshaler JSONUnmarshaler) *Client {
	c.JSONUnmarshaler = unmarshaler
	return c
}

func createTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}

	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.NumCPU() + 1,
	}
}

func createClient() *http.Client {
	return &http.Client{
		Transport: createTransport(),
		Timeout:   5 * time.Minute,
	}
}

func queryArray(v []string) string {
	s := lo.Map(v, func(item string, i int) string {
		return `"` + item + `"`
	})
	return fmt.Sprintf("[%s]", strings.Join(s, ","))
}
