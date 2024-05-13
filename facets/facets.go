package facets

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

var (
	ProjectType = newFacetProp("project_type")
	Versions    = newFacetProp("versions")
	Categories  = newFacetProp("categories")
	ClientSide  = newFacetProp("client_side")
	ServerSide  = newFacetProp("server_side")
	OpenSource  = newFacetProp("open_source")
)

func New(props ...FacetProp) FacetsBuilder {
	f := &facets{}
	if len(props) != 0 {
		f.And(props...)
	}
	return f
}

type FacetsBuilder interface {
	And(props ...FacetProp) FacetsBuilder
	// Or(props ...FacetProp) FacetsBuilder
	String() string
}

type FacetProp interface {
	Equal(s ...string) FacetProp
	NotEqual(s ...string) FacetProp
	Less(s ...string) FacetProp
	LessOrEqual(s ...string) FacetProp
	Greater(s ...string) FacetProp
	GreaterOrEqual(s ...string) FacetProp
	String() string
}

func newFacetProp(key string) func() FacetProp {
	return func() FacetProp {
		return &facetProp{
			key: key,
		}
	}
}

// Implements of FacetsProp
type facetProp struct {
	key   string
	state []string // [ `"categories=fabric"`, `"categories=quilt"` ]
}

// Examlpe: `"categories=fabric","categories=quilt"`
func (f *facetProp) String() string {
	return strings.Join(f.state, `,`)
}
func (f *facetProp) Equal(s ...string) FacetProp {
	for _, v := range s {
		f.state = append(f.state, fmt.Sprintf(`"%s=%s"`, f.key, v))
	}
	return f
}
func (f *facetProp) NotEqual(s ...string) FacetProp {
	for _, v := range s {
		f.state = append(f.state, fmt.Sprintf(`"%s!=%s"`, f.key, v))
	}
	return f
}
func (f *facetProp) Less(s ...string) FacetProp {
	for _, v := range s {
		f.state = append(f.state, fmt.Sprintf(`"%s<%s"`, f.key, v))
	}
	return f
}
func (f *facetProp) LessOrEqual(s ...string) FacetProp {
	for _, v := range s {
		f.state = append(f.state, fmt.Sprintf(`"%s<=%s"`, f.key, v))
	}
	return f
}
func (f *facetProp) Greater(s ...string) FacetProp {
	for _, v := range s {
		f.state = append(f.state, fmt.Sprintf(`"%s>%s"`, f.key, v))
	}
	return f
}
func (f *facetProp) GreaterOrEqual(s ...string) FacetProp {
	for _, v := range s {
		f.state = append(f.state, fmt.Sprintf(`"%s>=%s"`, f.key, v))
	}
	return f
}

type propList []FacetProp

func (p propList) String() string {
	s := lo.Map(p, func(p FacetProp, _ int) string {
		return p.String()
	})
	return strings.Join(s, ",")
}

// Implements of FacetsBuilder
type facets struct {
	state []string   // { `["versions:1.19.4"]`, `["categories:fabric","categories:quilt"]` }
	props []propList // [ [P,P], [P] ]
}

// [ [OR,OR,OR] ]
// func (f *facets) Or(props ...FacetProp) FacetsBuilder {
// 	f.props = append(f.props, props)
// 	return f
// }

// [ [AND],[AND],[AND] ]
func (f *facets) And(props ...FacetProp) FacetsBuilder {
	f.props = append(f.props, props)
	// for _, p := range props {
	// 	f.props = append(f.props, []FacetProp{p})
	// }
	return f
}
func (f *facets) String() string {
	s := lo.Map(f.props, func(e propList, _ int) string {
		return fmt.Sprintf(`[%s]`, e.String())
	})
	return fmt.Sprintf("[%s]", strings.Join(s, ","))
}
