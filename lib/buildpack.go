package sous

import (
	"fmt"
	"strings"
	"time"
)

type (
	// A Selector selects the buildpack for a given build context
	Selector interface {
		SelectBuildpack(*BuildContext) (Buildpack, error)
	}

	// Labeller defines a container-based build system.
	Labeller interface {
		ApplyMetadata(*BuildResult) error
	}

	// Registrar defines the interface to register build results to be deployed
	// later
	Registrar interface {
		// Register takes a BuildResult and makes it available for the deployment
		// target system to find during deployment
		Register(*BuildResult) error
	}

	// Strpairs is a slice of Strpair.
	Strpairs []Strpair

	// Strpair is a pair of strings.
	Strpair [2]string

	// BuildArtifact describes the actual built binary Sous will deploy
	BuildArtifact struct {
		Name, Type string
		Qualities  Qualities
	}

	// A Quality represents a characteristic of a BuildArtifact that needs to be recorded.
	Quality struct {
		Name string
		// Kind is the the kind of this quality
		// Known kinds include: advisory
		Kind string
	}

	// Qualities is a list of Quality
	Qualities []Quality

	// Buildpack is a set of instructions used to build a particular
	// kind of project.
	Buildpack interface {
		Detect(*BuildContext) (*DetectResult, error)
		Build(*BuildContext) (*BuildResult, error)
	}

	// DetectResult represents the result of a detection.
	DetectResult struct {
		// Compatible is true when the buildpack is compatible with the source
		// context.
		Compatible bool
		// Description is a human-readable description of what will be built.
		// It may for instance report back the base image that will be used,
		// or detected runtime version etc.
		Description string
		// Data is an arbitrary value. It can be used to pass interesting
		// detected information to the build step.
		Data interface{}
	}
	// BuildResult represents the result of a build made with a Buildpack.
	BuildResult struct {
		Elapsed  time.Duration
		Products []*BuildProduct
	}

	// A BuildProduct is one of the individual outputs of a buildpack.
	BuildProduct struct {
		// Source and Kind identify the build - the source inputs and the kind of build product
		Source SourceID
		Kind   string

		// ID is the artifact identifier - specific to product kind; e.g. docker
		// products have image ids.
		// Advisories contain the QA advisories determined on the build, and convey
		// prescriptive advice about how the image might be deployed.
		ID         string // was ImageID
		Advisories []string

		// VersionName and RevisionName cache computations about how to refer to the image.
		VersionName  string
		RevisionName string
	}
)

func (qs Qualities) String() string {
	strs := []string{}
	for _, q := range qs {
		strs = append(strs, q.Name)
	}
	return strings.Join(strs, ",")
}

// Contextualize records details from the BuildContext into the BuildResult
func (br *BuildResult) Contextualize(c *BuildContext) {
	advs := c.Advisories
	for _, prdt := range br.Products {
		if prdt.Source.Location.Repo == "" { // i.e. the buildstrat hasn't set the Source
			prdt.Source = c.Version() // ugh, yeah - Source and Version are both SourceID
		}
		if prdt.Advisories == nil {
			prdt.Advisories = make([]string, 0, len(advs))
		}
		prdt.Advisories = append(prdt.Advisories, advs...)
	}
}

func (br *BuildResult) String() string {
	str := ""
	for _, p := range br.Products {
		str = str + p.String() + "\n"
	}
	return str + fmt.Sprintf("Elapsed: %s", br.Elapsed)
}

func (bp *BuildProduct) String() string {
	str := fmt.Sprintf("Built: %q %q", bp.VersionName, bp.Kind)
	if len(bp.Advisories) > 0 {
		str = str + "\nAdvisories:\n  " + strings.Join(bp.Advisories, "  \n")
	}
	return str
}

// NewBuildArtifact creates a new BuildArtifact representing a Docker
// image.
func NewBuildArtifact(imageName string, qstrs Strpairs) *BuildArtifact {
	var qs []Quality
	for _, qstr := range qstrs {
		qs = append(qs, Quality{Name: qstr[0], Kind: qstr[1]})
	}

	return &BuildArtifact{Name: imageName, Type: "docker", Qualities: qs}
}
