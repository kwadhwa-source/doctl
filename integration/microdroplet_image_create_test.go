package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/microdroplet/image/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/v2/microdroplets/images" {
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}
				t.Fatalf("received unknown request: %s", dump)
			}
			expect.Equal(http.MethodPost, req.Method)
			expect.Equal("Bearer some-magic-token", req.Header.Get("Authorization"))

			var got map[string]any
			expect.NoError(json.NewDecoder(req.Body).Decode(&got))
			expect.Equal(map[string]any{
				"name":   "hello-world",
				"region": "nyc1",
				"source": "docker.io/library/hello-world:latest",
			}, got)

			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{
				"image": {
					"id": "aa11bb22-cc33-dd44-ee55-ff6600000000",
					"name": "hello-world",
					"region": "nyc1",
					"source": "docker.io/library/hello-world:latest",
					"status": "IMAGE_IMPORTING",
					"created_at": "2026-07-16T09:00:00Z"
				}
			}`)
		}))
	})

	when("all required flags are passed", func() {
		it("includes the image region in the request", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"-o", "json",
				"compute", "microdroplet", "image", "create", "hello-world",
				"--region", "nyc1",
				"--source", "docker.io/library/hello-world:latest",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			var images []map[string]any
			expect.NoError(json.Unmarshal(output, &images))
			expect.Len(images, 1)
			expect.Equal("nyc1", images[0]["region"])
		})
	})

	when("region is omitted", func() {
		it("returns a required flag error", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microdroplet", "image", "create", "hello-world",
				"--source", "docker.io/library/hello-world:latest",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "microdroplet-image.create.region")
		})
	})
})
