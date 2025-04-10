package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/snyk/snyk-code-review-exercise/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageHandler(t *testing.T) {
	// Review Comments:
	// This test will test packageHandler, which will downstream call fetchPacakge, and fetchPackageMetadata.
	// This means we are making calls directly to NPM every time we run this test.
	// We should mock these calls because:
	//	- What is NPM registry is down. Our tests will fail.
	//	- It is not polite to NPM registryt
	//	- It is slow
	// We can write out own mocks or generate mocks with GOA: https://pkg.go.dev/goa.design/clue/mock
	handler := api.New()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/package/react/16.13.0") // calls npm
	require.Nil(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)

	var data api.NpmPackageVersion
	err = json.Unmarshal(body, &data)
	require.Nil(t, err)

	assert.Equal(t, "react", data.Name)
	assert.Equal(t, "16.13.0", data.Version)

	fixture, err := os.Open(filepath.Join("testdata", "react-16.13.0.json"))
	require.Nil(t, err)
	var fixtureObj api.NpmPackageVersion
	require.Nil(t, json.NewDecoder(fixture).Decode(&fixtureObj))

	// Review Comments:
	// These tests are not passing, for two reasons.
	// The versions are different: V15.7.2 vs V15.8.1
	// 	- NPM resolves versions dynamically
	// 	- Different versions of a package at different times, different points in processing
	//  - Can fix this with mocks
	// fixtureObj contains dependancies which have pointers. We cannot compare pointers like this.
	// Pointers are just a reference to where the data is store, not the actual data.
	// We should marshal fixtureObj and Data to Bytes and then compare the bytes.
	assert.Equal(t, fixtureObj, data)
}

// Review Comments:
// We should test failure paths too
// Such as:
// - invalid package name
// - invalid version
// - npm is unreachable
// - circular dependancy
