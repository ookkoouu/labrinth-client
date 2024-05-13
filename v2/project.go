package labrinth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	neturl "net/url"
	"slices"
	"time"

	"github.com/google/go-querystring/query"
)

type ProjectsService service

type SearchParams struct {
	// The keyword to search for.
	Query string `url:"query,omitempty"`
	// The expression to filter search results.
	// Example: `[["categories:forge"],["versions:1.17.1"],["project_type:mod"],["license:mit"]]`
	Facets string `url:"facets,omitempty"`
	// The sorting method used for sorting search results.
	// Default: "relevance"
	Index SearchIndex `url:"index,omitempty"`
	// The offset into the search. Skips this number of results.
	// Default: 0
	Offset int `url:"offset,omitempty"`
	// The number of results returned by the search.
	// Default: 10
	Limit int `url:"limit,omitempty"`
}

type SearchIndex string

const (
	SearchIndex_Relevance = SearchIndex("relevance")
	SearchIndex_Downloads = SearchIndex("downloads")
	SearchIndex_Follows   = SearchIndex("follows")
	SearchIndex_Newest    = SearchIndex("newest")
	SearchIndex_Updated   = SearchIndex("updated")
)

func (s *ProjectsService) Search(ctx context.Context, params *SearchParams) (*SearchResult, *Response, error) {
	q := neturl.Values{}
	var err error
	if params != nil {
		q, err = query.Values(params)
		if err != nil {
			return nil, nil, err
		}
	}

	req, err := s.client.NewRequest(http.MethodGet, "search?"+q.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}

	var searchRes = new(SearchResult)
	res, err := s.client.Do(ctx, req, searchRes)
	if err != nil {
		return nil, res, err
	}

	return searchRes, res, nil
}

func (s *ProjectsService) Get(ctx context.Context, idSlug string) (*Project, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "project/"+idSlug, nil)
	if err != nil {
		return nil, nil, err
	}

	var proj = new(Project)
	res, err := s.client.Do(ctx, req, proj)
	if err != nil {
		return nil, res, err
	}

	return proj, res, nil
}

func (s *ProjectsService) GetAll(ctx context.Context, idSlugs []string) ([]*Project, *Response, error) {
	q := neturl.Values{}
	q.Add("ids", queryArray(idSlugs))

	req, err := s.client.NewRequest(http.MethodGet, "projects?"+q.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}

	var projs = []*Project{}
	res, err := s.client.Do(ctx, req, projs)
	if err != nil {
		return nil, nil, err
	}
	return projs, res, nil
}

func (s *ProjectsService) GetRandom(ctx context.Context, count int) ([]*Project, *Response, error) {
	if count < 0 {
		count = 0
	}
	if count > 100 {
		count = 100
	}

	q := neturl.Values{}
	q.Add("count", fmt.Sprint(count))

	req, err := s.client.NewRequest(http.MethodGet, "projects_random?"+q.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}

	var projs = []*Project{}
	res, err := s.client.Do(ctx, req, projs)
	if err != nil {
		return nil, nil, err
	}
	return projs, res, nil
}

type ValidityResponse struct {
	ID string `json:"id"`
}

func (s *ProjectsService) ValidSlugID(ctx context.Context, idSlug string) (*ValidityResponse, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, fmt.Sprintf("project/%s/check", idSlug), nil)
	if err != nil {
		return nil, nil, err
	}

	var proj = new(ValidityResponse)
	res, err := s.client.Do(ctx, req, proj)
	if err != nil {
		return nil, res, err
	}

	return proj, res, nil
}

type creatableProject struct {
	Slug                 string                `json:"slug"`         // Required
	Title                string                `json:"title"`        // Required
	Description          string                `json:"description"`  // Required
	ProjectType          ProjectType           `json:"project_type"` // Required
	Categories           []string              `json:"categories"`   // Required
	ClientSide           ProjectSideSupport    `json:"client_side"`  // Required
	ServerSide           ProjectSideSupport    `json:"server_side"`  // Required
	Body                 string                `json:"body"`         // Required
	LicenseID            string                `json:"license_id"`   // Required
	RequestedStatus      ProjectStatus         `json:"requested_status,omitempty"`
	AdditionalCategories []string              `json:"additional_categories,omitempty"`
	IssuesURL            string                `json:"issues_url,omitempty"`
	SourceURL            string                `json:"source_url,omitempty"`
	WikiURL              string                `json:"wiki_url,omitempty"`
	DiscordURL           string                `json:"discord_url,omitempty"`
	DonationUrls         []*ProjectDonationURL `json:"donation_urls,omitempty"`
	LicenseURL           string                `json:"license_url,omitempty"`
}

func (s *ProjectsService) Create(ctx context.Context, proj *Project) (*Project, *Response, error) {
	projReq := &creatableProject{
		Slug:                 proj.Slug,
		Title:                proj.Title,
		Description:          proj.Description,
		ProjectType:          proj.ProjectType,
		Categories:           proj.Categories,
		ClientSide:           proj.ClientSide,
		ServerSide:           proj.ServerSide,
		Body:                 proj.Body,
		LicenseID:            proj.License.ID,
		RequestedStatus:      *proj.RequestedStatus,
		AdditionalCategories: proj.AdditionalCategories,
		IssuesURL:            *proj.IssuesURL,
		SourceURL:            *proj.SourceURL,
		WikiURL:              *proj.WikiURL,
		DiscordURL:           *proj.DiscordURL,
		DonationUrls:         proj.DonationUrls,
		LicenseURL:           *proj.License.URL,
	}

	data, err := s.client.JSONMarshaler(projReq)
	if err != nil {
		return nil, nil, err
	}

	bodyBuf := new(bytes.Buffer)
	mw := multipart.NewWriter(bodyBuf)
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/json")
	pw, err := mw.CreatePart(mh)
	if err != nil {
		return nil, nil, err
	}

	_, err = pw.Write(data)
	if err != nil {
		return nil, nil, err
	}
	if err = mw.Close(); err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewFormRequest(http.MethodPost, "project", bodyBuf)
	if err != nil {
		return nil, nil, err
	}

	var projRes = new(Project)
	res, err := s.client.Do(ctx, req, projRes)
	if err != nil {
		return nil, res, err
	}
	return projRes, res, nil
}

type editableProject struct {
	Slug                 string                `json:"slug,omitempty"`
	Title                string                `json:"title,omitempty"`
	Description          string                `json:"description,omitempty"`
	Categories           []string              `json:"categories,omitempty"`
	ClientSide           ProjectSideSupport    `json:"client_side,omitempty"`
	ServerSide           ProjectSideSupport    `json:"server_side,omitempty"`
	Body                 string                `json:"body,omitempty"`
	RequestedStatus      ProjectStatus         `json:"requested_status,omitempty"`
	AdditionalCategories []string              `json:"additional_categories,omitempty"`
	IssuesURL            string                `json:"issues_url,omitempty"`
	SourceURL            string                `json:"source_url,omitempty"`
	WikiURL              string                `json:"wiki_url,omitempty"`
	DiscordURL           string                `json:"discord_url,omitempty"`
	DonationUrls         []*ProjectDonationURL `json:"donation_urls,omitempty"`
	LicenseID            string                `json:"license_id,omitempty"`
	LicenseURL           string                `json:"license_url,omitempty"`
}

func (s *ProjectsService) Edit(ctx context.Context, idSlug string, proj *Project) (*Project, *Response, error) {
	projReq := &editableProject{
		Slug:                 proj.Slug,
		Title:                proj.Title,
		Description:          proj.Description,
		Categories:           proj.Categories,
		ClientSide:           proj.ClientSide,
		ServerSide:           proj.ServerSide,
		Body:                 proj.Body,
		LicenseID:            proj.License.ID,
		RequestedStatus:      *proj.RequestedStatus,
		AdditionalCategories: proj.AdditionalCategories,
		IssuesURL:            *proj.IssuesURL,
		SourceURL:            *proj.SourceURL,
		WikiURL:              *proj.WikiURL,
		DiscordURL:           *proj.DiscordURL,
		DonationUrls:         proj.DonationUrls,
		LicenseURL:           *proj.License.URL,
	}

	req, err := s.client.NewRequest(http.MethodPatch, "project/"+idSlug, projReq)
	if err != nil {
		return nil, nil, err
	}

	var projRes = new(Project)
	res, err := s.client.Do(ctx, req, projRes)
	if err != nil {
		return nil, res, err
	}

	return projRes, res, nil
}

type ProjectEditAll struct {
	Categories                 []string `json:"categories,omitempty"`
	AddCategories              []string `json:"add_categories,omitempty"`
	RemoveCategories           []string `json:"remove_categories,omitempty"`
	AdditionalCategories       []string `json:"additional_categories,omitempty"`
	AddAdditionalCategories    []string `json:"add_additional_categories,omitempty"`
	RemoveAdditionalCategories []string `json:"remove_additional_categories,omitempty"`
	DonationUrls               []*struct {
		ID       string `json:"id,omitempty"`
		Platform string `json:"platform,omitempty"`
		URL      string `json:"url,omitempty"`
	} `json:"donation_urls,omitempty"`
	AddDonationUrls []*struct {
		ID       string `json:"id,omitempty"`
		Platform string `json:"platform,omitempty"`
		URL      string `json:"url,omitempty"`
	} `json:"add_donation_urls,omitempty"`
	RemoveDonationUrls []*struct {
		ID       string `json:"id,omitempty"`
		Platform string `json:"platform,omitempty"`
		URL      string `json:"url,omitempty"`
	} `json:"remove_donation_urls,omitempty"`
	IssuesURL  string `json:"issues_url,omitempty"`
	SourceURL  string `json:"source_url,omitempty"`
	WikiURL    string `json:"wiki_url,omitempty"`
	DiscordURL string `json:"discord_url,omitempty"`
}

// EditAll edits specified fields in all projects at once
func (s *ProjectsService) EditAll(ctx context.Context, idSlugs []string, params *ProjectEditAll) (*Response, error) {
	q := neturl.Values{}
	q.Add("ids", queryArray(idSlugs))

	req, err := s.client.NewRequest(http.MethodPatch, "projects?"+q.Encode(), params)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

func (s *ProjectsService) Delete(ctx context.Context, idSlug string) (*Response, error) {
	req, err := s.client.NewRequest(http.MethodDelete, "project/"+idSlug, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

type EditProjectIconParams struct {
	// Extension of icon file to upload.
	// Example: "jpeg"
	Ext  string `url:"ext"`
	File io.Reader
}

func (s *ProjectsService) ChangeIcon(ctx context.Context, idSlug string, params *EditProjectIconParams) (*Response, error) {
	if !slices.Contains(supportedImageExt, params.Ext) {
		return nil, errors.New("unsupported iamge file")
	}
	if params.Ext == "jpg" {
		params.Ext = "jpeg"
	}
	q, err := query.Values(params)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewUploadRequest(
		http.MethodPatch,
		fmt.Sprintf("project/%s/icon?%s", idSlug, q.Encode()),
		"image/"+params.Ext,
		params.File)

	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

func (s *ProjectsService) DeleteIcon(ctx context.Context, idSlug string) (*Response, error) {
	req, err := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("project/%s/icon", idSlug), nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

type AddGalleryImageParams struct {
	Ext         string    `url:"ext"` // Required
	Featured    bool      `url:"featured"`
	Title       string    `url:"title,omitempty"`
	Description string    `url:"description,omitempty"`
	Ordering    int       `url:"ordering,omitempty"`
	File        io.Reader // Required
}

func (s *ProjectsService) AddGalleryImage(ctx context.Context, idSlug string, params *AddGalleryImageParams) (*Response, error) {
	if !slices.Contains(supportedImageExt, params.Ext) {
		return nil, errors.New("unsupported iamge file")
	}
	if params.Ext == "jpg" {
		params.Ext = "jpeg"
	}
	q, err := query.Values(params)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewUploadRequest(
		http.MethodPost,
		fmt.Sprintf("project/%s/gallery?%s", idSlug, q.Encode()),
		"image/"+params.Ext,
		params.File)

	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

type EditGalleryImageParams struct {
	URL         string `url:"url"`
	Featured    bool   `url:"featured"`
	Title       string `url:"title,omitempty"`
	Description string `url:"description,omitempty"`
	Ordering    int    `url:"ordering,omitempty"`
}

func (s *ProjectsService) EditGalleryImage(ctx context.Context, idSlug string, params *AddGalleryImageParams) (*Response, error) {
	q, err := query.Values(params)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("project/%s/gallery?%s", idSlug, q.Encode()),
		nil)

	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

func (s *ProjectsService) DeleteGalleryImage(ctx context.Context, idSlug string, url string) (*Response, error) {
	q := neturl.Values{}
	q.Add("url", url)

	req, err := s.client.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("project/%s/gallery?%s", idSlug, q.Encode()),
		nil)

	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

type ProjectDependencies struct {
	Projects []*Project `json:"projects"`
	Versions []*Version `json:"versions"`
}

func (s *ProjectsService) GetDependencies(ctx context.Context, idSlug string) (*ProjectDependencies, *Response, error) {
	req, err := s.client.NewRequest(
		http.MethodGet,
		fmt.Sprintf("project/%s/dependencies",
			idSlug), nil)
	if err != nil {
		return nil, nil, err
	}

	var deps = new(ProjectDependencies)
	res, err := s.client.Do(ctx, req, deps)
	if err != nil {
		return nil, res, err
	}

	return deps, res, nil
}

func (s *ProjectsService) Follow(ctx context.Context, idSlug string) (*Response, error) {
	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("project/%s/follow", idSlug), nil)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (s *ProjectsService) Unfollow(ctx context.Context, idSlug string) (*Response, error) {
	req, err := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("project/%s/follow", idSlug), nil)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return res, err
	}

	return res, nil
}

type ScheduleProjectParams struct {
	Time            time.Time     `json:"time"`
	RequestedStatus ProjectStatus `json:"requested_status"`
}

func (s *ProjectsService) Schedule(ctx context.Context, idSlug string, params *ScheduleProjectParams) (*Response, error) {
	if !params.RequestedStatus.IsRequestable() {
		return nil, errors.New("project_status is not requestable")
	}

	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("project/%s/schedule", idSlug), params)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return res, err
	}

	return res, nil
}
