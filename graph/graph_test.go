package graph

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusPoller(t *testing.T) {
	testPoller := func(sf config.DeployFilterFlags) *sous.StatusPoller {
		shc := sous.SourceHostChooser{}
		f, err := sf.BuildFilter(shc.ParseSourceLocation)
		require.NoError(t, err)

		// func newRefinedResolveFilter(f *sous.ResolveFilter, discovered *SourceContextDiscovery) (*RefinedResolveFilter, error) {

		disc := &SourceContextDiscovery{
			SourceContext: &sous.SourceContext{
				PrimaryRemoteURL: "github.com/somewhere/thing",
				NearestTag:       sous.Tag{Name: "1.2.3", Revision: "cabbage"},
				Tags:             []sous.Tag{},
			},
		}
		rf, err := newRefinedResolveFilter(f, disc)
		require.NoError(t, err)
		cl := newDummyHTTPClient()
		user := sous.User{}

		//newStatusPoller(cl HTTPClient, rf *RefinedResolveFilter, user sous.User) *sous.StatusPoller {
		return newStatusPoller(cl, rf, user, LogSink{logging.SilentLogSet()})
	}

	p := testPoller(config.DeployFilterFlags{})
	assert.False(t, p.ResolveFilter.Flavor.All())
}

func TestBuildGraph(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	g := BuildGraph(semv.MustParse("0.0.0"), &bytes.Buffer{}, ioutil.Discard, ioutil.Discard)
	g.Add(DryrunBoth)
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.PolicyFlags{}) //provided by SousBuild
	g.Add(&config.OTPLFlags{})   //provided by SousInit and SousDeploy

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestLogSink(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	g := BuildGraph(semv.MustParse("0.0.0"), &bytes.Buffer{}, ioutil.Discard, ioutil.Discard)
	g.Add(DryrunBoth)
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.PolicyFlags{}) //provided by SousBuild
	g.Add(&config.OTPLFlags{})   //provided by SousInit and SousDeploy

	tg := &psyringe.TestPsyringe{g.Psyringe}
	rawConfig := RawConfig{Config: &config.Config{}}
	logcfg := &rawConfig.Config.Logging
	logcfg.Basic.Level = "debug"
	//logcfg.Kafka.Enabled = true
	logcfg.Kafka.DefaultLevel = "debug"
	logcfg.Kafka.Topic = "logging"
	logcfg.Kafka.BrokerList = "kafka.example.com:9292"
	logcfg.Graphite.Enabled = true
	logcfg.Graphite.Server = "localhost:3333"

	tg.Replace(rawConfig)

	scoop := struct{ LogSink }{}

	tg.MustInject(&scoop)

	set, is := scoop.LogSink.LogSink.(*logging.LogSet)

	assert.True(t, is)
	assert.NoError(t, logging.AssertConfiguration(set, "localhost:3333"))

}

func injectedStateManager(t *testing.T, cfg *config.Config) *StateManager {
	g := newSousGraph()
	g.Add(newUser)
	g.Add(LogSink{logging.SilentLogSet()})
	g.Add(MetricsHandler{})
	g.Add(newStateManager)
	g.Add(LocalSousConfig{Config: cfg})
	g.Add(newServerComponentLocator)
	g.Add(newResolveFilter)
	g.Add(newSourceHostChooser)
	g.Add(DryrunBoth)
	g.Add(newDeployer)
	g.Add(newLazyNameCache)
	g.Add(newNameCache)
	g.Add(newRegistry)
	g.Add(newInserter)
	g.Add(newDockerClient)
	g.Add(newServerStateManager)
	g.Add(&config.DeployFilterFlags{})
	g.Add(newResolver)
	g.Add(newAutoResolver)
	g.Add(newServerHandler)
	g.Add(newHTTPClient)
	g.Add(g)

	smRcvr := struct {
		Sm *StateManager
	}{}
	err := g.Inject(&smRcvr)
	if err != nil {
		t.Fatalf("Injection err: %+v", err)
	}

	if smRcvr.Sm == nil {
		t.Fatal("StateManager not injected")
	}
	return smRcvr.Sm
}

func TestStateManagerSelectsServer(t *testing.T) {
	smgr := injectedStateManager(t, &config.Config{Server: "http://example.com", StateLocation: "/tmp/sous"})

	if _, ok := smgr.StateManager.(*sous.HTTPStateManager); !ok {
		t.Errorf("Injected %#v which isn't a HTTPStateManager", smgr.StateManager)
	}
}

func TestStateManagerSelectsDuplex(t *testing.T) {
	smgr := injectedStateManager(t, &config.Config{Server: "", StateLocation: "/tmp/sous"})

	if _, ok := smgr.StateManager.(*storage.DuplexStateManager); !ok {
		t.Errorf("Injected %#v which isn't a DuplexStateManager", smgr.StateManager)
	}
}

func TestNewBuildConfig(t *testing.T) {
	f := &config.DeployFilterFlags{}
	p := &config.PolicyFlags{}
	bc := &sous.BuildContext{
		Sh: &shell.Sh{},
		Source: sous.SourceContext{
			RemoteURL: "github.com/opentable/present",
			RemoteURLs: []string{
				"github.com/opentable/present",
				"github.com/opentable/also",
			},
			Revision:           "abcdef",
			NearestTagName:     "1.2.3",
			NearestTagRevision: "abcdef",
			Tags: []sous.Tag{
				sous.Tag{Name: "1.2.3"},
			},
		},
	}

	cfg := newBuildConfig(f, p, bc)
	if cfg.Tag != `1.2.3` {
		t.Errorf("Build config's tag wasn't 1.2.3: %#v", cfg.Tag)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Not valid build config: %+v", err)
	}

}
