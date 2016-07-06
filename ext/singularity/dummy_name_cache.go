package singularity

import (
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
)

// DummyNameCache implements the Builder interface by returning a
// computed image name for a given source version
type DummyNameCache struct {
}

// NewDummyNameCache builds a new DummyNameCache
func NewDummyNameCache() *DummyNameCache {
	return &DummyNameCache{}
}

// TODO: Factor out name cache concept from core sous lib & get rid of this func.
func (dc *DummyNameCache) Build(*sous.BuildContext, sous.Buildpack, *sous.DetectResult) (*sous.BuildResult, error) {
	return nil, nil
	panic("not implemented")
}

// TODO: Factor out name cache concept from core sous lib & get rid of this func.
func (dc *DummyNameCache) GetArtifact(sv sous.SourceVersion) (*sous.BuildArtifact, error) {
	imageName, err := dc.GetImageName(sv)
	if err != nil {
		return nil, err
	}
	return docker.DockerBuildArtifact(imageName), nil
}

// GetImageName implements part of the interface for ImageMapper
func (dc *DummyNameCache) GetImageName(sv sous.SourceVersion) (string, error) {
	return sv.String(), nil
}

// GetCanonicalName implements part of the interface for ImageMapper
// It simply returns whatever it was given
func (dc *DummyNameCache) GetCanonicalName(in string) (string, error) {
	return in, nil
}

// Insert implements part of ImageMapper
// it drops the sv/in pair on the floor
func (dc *DummyNameCache) Insert(sv sous.SourceVersion, in, etag string) error {
	return nil
}

// GetSourceVersion implements part of ImageMapper
func (dc *DummyNameCache) GetSourceVersion(*sous.BuildArtifact) (sous.SourceVersion, error) {
	return sous.SourceVersion{}, nil
}
