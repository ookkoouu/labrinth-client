package labrinth

import (
	"slices"
	"time"
)

type Project struct {
	Slug                 string                `json:"slug"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	Categories           []string              `json:"categories"`
	ClientSide           ProjectSideSupport    `json:"client_side"`
	ServerSide           ProjectSideSupport    `json:"server_side"`
	Body                 string                `json:"body"`
	BodyURL              *string               `json:"body_url"` // Deprecated: Allways null.
	Status               ProjectStatus         `json:"status"`
	RequestedStatus      *ProjectStatus        `json:"requested_status"`
	AdditionalCategories []string              `json:"additional_categories"`
	IssuesURL            *string               `json:"issues_url"`
	SourceURL            *string               `json:"source_url"`
	WikiURL              *string               `json:"wiki_url"`
	DiscordURL           *string               `json:"discord_url"`
	DonationUrls         []*ProjectDonationURL `json:"donation_urls"`
	ProjectType          ProjectType           `json:"project_type"`
	Downloads            int                   `json:"downloads"`
	IconURL              *string               `json:"icon_url"`
	Color                *int                  `json:"color"`
	ThreadID             string                `json:"thread_id"`
	ModeratorMessage     *string               `json:"moderator_message"` // Deprecated: Allways null.
	MonetizationStatus   MonetizationStatus    `json:"monetization_status"`
	ID                   string                `json:"id"`
	Team                 string                `json:"team"`
	Organization         *string               `json:"organization"`
	Published            time.Time             `json:"published"`
	Updated              time.Time             `json:"updated"`
	Approved             *time.Time            `json:"approved"`
	Queued               *time.Time            `json:"queued"`
	Followers            int                   `json:"followers"`
	License              *ProjectLicense       `json:"license"`
	Versions             []string              `json:"versions"`
	GameVersions         []string              `json:"game_versions"`
	Loaders              []string              `json:"loaders"`
	Gallery              []*GalleryImage       `json:"gallery"`
}

type ProjectSideSupport string

const (
	ProjectSideSupport_Required    = ProjectSideSupport("required")
	ProjectSideSupport_Optional    = ProjectSideSupport("optional")
	ProjectSideSupport_Unsupported = ProjectSideSupport("unsupported")
	ProjectSideSupport_Unknown     = ProjectSideSupport("unknown")
)

type ProjectStatus string

const (
	ProjectStatus_Approved   = ProjectStatus("approved")
	ProjectStatus_Archived   = ProjectStatus("archived")
	ProjectStatus_Rejected   = ProjectStatus("rejected")
	ProjectStatus_Draft      = ProjectStatus("draft")
	ProjectStatus_Unlisted   = ProjectStatus("unlisted")
	ProjectStatus_Processing = ProjectStatus("processing")
	ProjectStatus_Withheld   = ProjectStatus("withheld")
	ProjectStatus_Scheduled  = ProjectStatus("scheduled")
	ProjectStatus_Private    = ProjectStatus("private")
	ProjectStatus_Unknown    = ProjectStatus("unknown")
)

var requestableProjectStatus = []string{"approved", "archived", "draft", "unlisted", "private"}

func (s ProjectStatus) IsRequestable() bool {
	return slices.Contains(requestableProjectStatus, string(s))
}

type ProjectDonationURL struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

type ProjectType string

const (
	ProjectType_Mod          = ProjectType("mod")
	ProjectType_Modpack      = ProjectType("modpack")
	ProjectType_Resourcepack = ProjectType("resourcepack")
	ProjectType_Shader       = ProjectType("shader")
	ProjectType_Unknown      = ProjectType("project")
)

type MonetizationStatus string

const (
	MonetizationStatus_Monetized        = MonetizationStatus("monetized")
	MonetizationStatus_Demonetized      = MonetizationStatus("demonetized")
	MonetizationStatus_ForceDemonetized = MonetizationStatus("force-demonetized")
)

const (
	ProjectLicenseID_Unknown = "LicenseRef-Unknown"
)

type ProjectLicense struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	URL  *string `json:"url"`
}

type GalleryImage struct {
	URL         string    `json:"url"`
	Featured    bool      `json:"featured"`
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Created     time.Time `json:"created"`
	Ordering    int       `json:"ordering"`
}
