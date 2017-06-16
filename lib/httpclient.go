package sous

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type (
	// LiveHTTPClient interacts with a Sous http server.
	//   It's designed to handle basic CRUD operations in a safe and restful way.
	LiveHTTPClient struct {
		serverURL *url.URL
		http.Client
	}

	resourceState struct {
		etag         string
		body         io.Reader
		resourceJSON io.Reader
	}

	// HTTPClient interacts with a HTTPServer
	//   It's designed to handle basic CRUD operations in a safe and restful way.
	HTTPClient interface {
		Create(urlPath string, qParms map[string]string, rqBody interface{}, user User) error
		Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, user User) error
		RetrieveWithState(urlPath string, qParms map[string]string, rzBody interface{}, user User) (*resourceState, error)
		Update(urlPath string, qParms map[string]string, from *resourceState, qBody Comparable, user User) error
		Delete(urlPath string, qParms map[string]string, from *resourceState, user User) error
	}

	// DummyHTTPClient doesn't really make HTTP requests.
	DummyHTTPClient struct{}

	// Comparable is a required interface for Update and Delete, which provides
	// the mechanism for comparing the remote resource to the local data.
	Comparable interface {
		// EmptyReceiver should return a pointer to an "zero value" for the recieving type.
		// For example:
		//   func (x *X) EmptyReceiver() { return &X{} }
		EmptyReceiver() Comparable

		// VariancesFrom returns a list of differences from another Comparable.
		// If the two structs are equivalent, it should return an empty list.
		// Usually, the first check will be for identical type, and return "types differ."
		VariancesFrom(Comparable) Variances
	}

	// Variances is a list of differences between two structs.
	Variances []string

	retryableError string
)

func (re retryableError) Error() string {
	return string(re)
}

// Retryable is a predicate on error that returns true if the error indicates
// that a subsequent attempt at e.g. an Update might succeed.
func Retryable(err error) bool {
	_, is := errors.Cause(err).(retryableError)
	return is
}

// NewClient returns a new LiveHTTPClient for a particular serverURL.
func NewClient(serverURL string) (*LiveHTTPClient, error) {
	u, err := url.Parse(serverURL)

	client := &LiveHTTPClient{
		serverURL: u,
	}

	// XXX: This is in response to a mysterious issue surrounding automatic gzip
	// and Etagging The client receives a gzipped response with "--gzip" appended
	// to the original Etag The --gzip isn't stripped by whatever does it,
	// although the body is decompressed on the server side.  This is a hack to
	// address that issue, which should be resolved properly.
	client.Client.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, DisableCompression: true}

	return client, errors.Wrapf(err, "new Sous REST client")
}

// ****

// Retrieve makes a GET request on urlPath, after transforming qParms into ?&=
// style query params. It deserializes the returned JSON into rzBody. Errors
// are returned if anything goes wrong, including a non-Success HTTP result
// (but note that there may be a response anyway.
func (client *LiveHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, user User) error {
	_, err := client.RetrieveWithState(urlPath, qParms, rzBody, user)
	return err
}

// RetrieveWithState makes a GET request on urlPath, after transforming qParms into ?&=
// style query params. It deserializes the returned JSON into rzBody. Errors
// are returned if anything goes wrong, including a non-Success HTTP result
// (but note that there may be a response anyway.
// It returns an opaque state object for use with Update
func (client *LiveHTTPClient) RetrieveWithState(urlPath string, qParms map[string]string, rzBody interface{}, user User) (*resourceState, error) {
	url, err := client.buildURL(urlPath, qParms)
	rq, err := client.buildRequest("GET", url, user, nil, nil, nil, err)
	rz, err := client.sendRequest(rq, err)
	state, err := client.getBody(rz, rzBody, err)
	return state, errors.Wrapf(err, "Retrieve %s", urlPath)
}

// Create uses the contents of qBody to create a new resource at the server at urlPath/qParms
// It issues a PUT with "If-No-Match: *", so if a resource already exists, it'll return an error.
func (client *LiveHTTPClient) Create(urlPath string, qParms map[string]string, qBody interface{}, user User) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		rq, err := client.buildRequest("PUT", url, user, noMatchStar(), nil, qBody, err)
		rz, err := client.sendRequest(rq, err)
		_, err = client.getBody(rz, nil, err)
		return err
	}(), "Create %s", urlPath)
}

// Update changes the representation of a given resource.
// It compares the known value to from, and rejects if they're different (on
// the grounds that the client is going to clobber a value it doesn't know
// about.) Then it issues a PUT with "If-Match: <etag of from>" so that the
// server can check that we're changing from a known value.
func (client *LiveHTTPClient) Update(urlPath string, qParms map[string]string, from *resourceState, qBody Comparable, user User) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		//	etag := from.etag
		etag := from.etag
		rq, err := client.buildRequest("PUT", url, user, ifMatch(etag), from, qBody, err)
		rz, err := client.sendRequest(rq, err)
		_, err = client.getBody(rz, nil, err)
		return err
	}(), "Update %s", urlPath)
}

// Delete removes a resource from the server, granted that we know the resource that we're removing.
// It functions similarly to Update, but issues DELETE requests.
func (client *LiveHTTPClient) Delete(urlPath string, qParms map[string]string, from *resourceState, user User) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		etag := from.etag
		rq, err := client.buildRequest("DELETE", url, user, ifMatch(etag), nil, nil, err)
		rz, err := client.sendRequest(rq, err)
		_, err = client.getBody(rz, nil, err)
		return err
	}(), "Delete %s", urlPath)
}

// Create implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Create(urlPath string, qParms map[string]string, rqBody interface{}, user User) error {
	return nil
}

// Retrieve implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, user User) error {
	return nil
}

// RetrieveWithState implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) RetrieveWithState(urlPath string, qParms map[string]string, rzBody interface{}, user User) (*resourceState, error) {
	return nil, nil
}

// Update implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Update(urlPath string, qParms map[string]string, from *resourceState, qBody Comparable, user User) error {
	return nil
}

// Delete implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Delete(urlPath string, qParms map[string]string, from *resourceState, user User) error {
	return nil
}

// ***

func noMatchStar() map[string]string {
	return map[string]string{"If-None-Match": "*"}
}

func ifMatch(etag string) map[string]string {
	return map[string]string{"If-Match": etag}
}

// ****

func (client *LiveHTTPClient) buildURL(urlPath string, qParms map[string]string) (urlS string, err error) {
	URL, err := client.serverURL.Parse(urlPath)
	if err != nil {
		return
	}
	if qParms == nil {
		return URL.String(), nil
	}
	qry := url.Values{}
	for k, v := range qParms {
		qry.Set(k, v)
	}
	URL.RawQuery = qry.Encode()
	return client.serverURL.ResolveReference(URL).String(), nil
}

func (client *LiveHTTPClient) buildRequest(method, url string, user User, headers map[string]string, resource *resourceState, rqBody interface{}, ierr error) (*http.Request, error) {
	if ierr != nil {
		return nil, ierr
	}

	Log.Debug.Printf("Sending %s %q", method, url)

	JSON := &bytes.Buffer{}

	if rqBody != nil {
		JSON = encodeJSON(rqBody)
		if resource != nil {
			JSON = putbackJSON(resource.body, resource.resourceJSON, JSON)
		}
		Log.Debug.Printf("  body: %s", JSON.String())
	}

	rq, err := http.NewRequest(method, url, JSON)

	rq.Header.Add("Sous-User-Name", user.Name)
	rq.Header.Add("Sous-User-Email", user.Email)

	if headers != nil {
		for k, v := range headers {
			rq.Header.Add(k, v)
		}
	}

	return rq, err
}

func (client *LiveHTTPClient) sendRequest(rq *http.Request, ierr error) (*http.Response, error) {
	if ierr != nil {
		return nil, ierr
	}
	rz, err := client.httpRequest(rq)
	if err != nil {
		Log.Debug.Printf("Received %v", err)
		return rz, err
	}
	if rz != nil {
		Log.Debug.Printf("Received \"%s %s\" -> %d", rq.Method, rq.URL, rz.StatusCode)
	}
	return rz, err
}

func (client *LiveHTTPClient) getBody(rz *http.Response, rzBody interface{}, err error) (*resourceState, error) {
	if err != nil {
		return nil, err
	}
	defer rz.Body.Close()

	if rzBody != nil {
		dec := json.NewDecoder(rz.Body)
		err = dec.Decode(rzBody)
	}

	b, e := ioutil.ReadAll(rz.Body)
	if e != nil {
		Log.Debug.Printf("error reading from body: %v", e)
		b = []byte{}
	}

	switch {
	default:
		rzJSON, merr := json.Marshal(rzBody)
		if err == nil {
			err = merr
		}
		return &resourceState{
			etag:         rz.Header.Get("ETag"),
			body:         bytes.NewBuffer(b),
			resourceJSON: bytes.NewBuffer(rzJSON),
		}, errors.Wrapf(err, "processing response body")
	case rz.StatusCode < 200 || rz.StatusCode >= 300:
		return nil, errors.Errorf("%s: %#v", rz.Status, string(b))
	case rz.StatusCode == http.StatusConflict:
		return nil, errors.Wrap(retryableError(fmt.Sprintf("%s: %#v", rz.Status, string(b))), "getBody")
	}

}

func logBody(dir, chName string, req *http.Request, b []byte, n int, err error) {
	Log.Vomit.Printf("%s %s %q", chName, req.Method, req.URL)
	comp := &bytes.Buffer{}
	if err := json.Compact(comp, b[0:n]); err != nil {
		Log.Vomit.Print(string(b))
		Log.Vomit.Printf("(problem compacting JSON for logging: %s)", err)
	} else {
		Log.Vomit.Print(comp.String())
	}
	Log.Vomit.Printf("%s %d bytes, result: %v", dir, n, err)
}

func (client *LiveHTTPClient) httpRequest(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		Log.Vomit.Printf("Client -> %s %q <empty request body>", req.Method, req.URL)
	} else {
		req.Body = NewReadDebugger(req.Body, func(b []byte, n int, err error) {
			logBody("Sent", "Client ->", req, b, n, err)
		})
	}
	rz, err := client.Client.Do(req)
	if rz == nil {
		return rz, err
	}
	if rz.Body == nil {
		Log.Vomit.Printf("Client <- %s %q %d <empty response body>", req.Method, req.URL, rz.StatusCode)
		return rz, err
	}

	rz.Body = NewReadDebugger(rz.Body, func(b []byte, n int, err error) {
		logBody("Read", "Client <-", req, b, n, err)
	})
	return rz, err
}
