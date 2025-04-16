package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gorilla/mux"
)

func New() http.Handler {
	router := mux.NewRouter()
	// Review Comments: Beyond Scope
	// Its best practice to version our own api routes
	router.Handle("/package/{package}/{version}", http.HandlerFunc(packageHandler))
	return router
}

type npmPackageMetaResponse struct {
	Versions map[string]npmPackageResponse `json:"versions"`
}

type npmPackageResponse struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

// Review Comments:
// We should use JSON tag: omitempty for Dependencies.
// When we unmarshal JSON where Dependancies is empty, our json will omit the field all together
// instead of displaying it like Dependancies{}.
// Not go best practice to pass around empty structs

type NpmPackageVersion struct {
	Name         string                        `json:"name"`
	Version      string                        `json:"version"`
	Dependencies map[string]*NpmPackageVersion `json:"dependencies"`
}

// Review Comments: Beyond Scope
// We could add some sort of "cache" so that we do not need walk trees for packages we have already seen
func packageHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	vars := mux.Vars(r)

	// Review Comments:
	// We should validate pkgName and pkgVersion to make sure that they are not empty.
	// If either var is empty, we should log an error, letting the user know that that we do not accept empty input
	pkgName := vars["package"]
	pkgVersion := vars["version"]

	rootPkg := &NpmPackageVersion{Name: pkgName, Dependencies: map[string]*NpmPackageVersion{}}
	// Review Comment: Nitpick about style consistancy
	// If err := ...; err != nil {} is an okay syntax, but otherplaces of the code do a more standard
	// 	err := ...
	// 	if err != nil {}
	// Just pick a error check/return pattern and make it uniform throughout the code

	if err := resolveDependencies(rootPkg, pkgVersion, []string{}); err != nil {
		// Review Comment:
		// We shouldnt print errors. Should introduce logger.
		println(err.Error())
		// Review Comment: Nitpick
		// Its better practice to return Http status codes via const
		// See here: https://go.dev/src/net/http/status.go
		w.WriteHeader(500)
		return
	}

	stringified, err := json.MarshalIndent(rootPkg, "", "  ")
	if err != nil {
		println(err.Error())
		w.WriteHeader(500)
		return
	}

	filename := fmt.Sprintf("%s@%s.json", pkgName, pkgVersion)
	if err := os.WriteFile(filename, stringified, 0644); err != nil {
		fmt.Printf("Failed to write file: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	// Review Comment:
	// We shouldnt ignore write errors
	_, _ = w.Write(stringified)

	fmt.Printf("Execution time for %s@%s:  %vs\n", pkgName, pkgVersion, time.Since(start))
}

func resolveDependencies(pkg *NpmPackageVersion, versionConstraint string, stack []string) error {

	pkgMeta, err := fetchPackageMeta(pkg.Name)
	if err != nil {
		return err
	}
	concreteVersion, err := highestCompatibleVersion(versionConstraint, pkgMeta)
	if err != nil {
		return err
	}
	pkg.Version = concreteVersion

	npmPkg, err := fetchPackage(pkg.Name, pkg.Version)
	if err != nil {
		return err
	}

	// Check for circular dependency:

	// Generate key for stack
	key := fmt.Sprintf("%s@v%s", pkg.Name, pkg.Version) // will be react@v16.13.0

	// check if key is in stack
	for i, k := range stack {
		if k == key {
			// Circular dependency detected
			circularPath := append(stack[i:], key) // Include the cycle in the path
			fmt.Printf("Circular dependency detected: %s\n", circularPath)

			return nil
		}
	}
	// add to stack
	stack = append(stack, key)

	// Review Comments:
	//
	// When testing via curl, I noticed a few things:
	// 1. This is not optomized for large pacakges.
	// 	The terminal hangs with no feedback to the user, that there is still processing happening.
	// 		- We can optomize by doing this concurrently. Each dependacy can be processed in a go routine.
	//		- This package takes a while: curl -s http://localhost:3000/package/express/5.1.0 | jq .
	// 2. We are gonna have circular dependancies
	// If we have dependancy structure: pkg a -> pkg b -> pkg c -> pkg a.
	// With no clear endpoint we can hit infinite recursion/ be performing duplicate work
	// - We could use a stack
	// - curl -s http://localhost:3000/package/trucolor/4.0.4 | jq .
	for dependencyName, dependencyVersionConstraint := range npmPkg.Dependencies {
		// Review Comments: Nitpick
		// It is technically better to not set Dependancies to empty struct here.
		// It will be assigned later.
		// Also means we can check for nil, instead of empty.
		dep := &NpmPackageVersion{Name: dependencyName, Dependencies: map[string]*NpmPackageVersion{}}
		pkg.Dependencies[dependencyName] = dep
		if err := resolveDependencies(dep, dependencyVersionConstraint, stack); err != nil {
			return err // slow, could add concurrancy
		}
	}

	// Remove the current package from the stack after processing
	//stack = stack[:len(stack)-1]

	return nil
}

func highestCompatibleVersion(constraintStr string, versions *npmPackageMetaResponse) (string, error) {
	constraint, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return "", err
	}
	filtered := filterCompatibleVersions(constraint, versions)
	sort.Sort(filtered)
	if len(filtered) == 0 {
		// Review Comments: Beyond Scope
		// We should error with fields here, it will make debugging easier
		// 	EX: return "", fmt.Errorf("no compatible versions found for %s with constraint %q", pkgName, constraintStr)
		return "", errors.New("no compatible versions found")
	}
	return filtered[len(filtered)-1].String(), nil
}

func filterCompatibleVersions(constraint *semver.Constraints, pkgMeta *npmPackageMetaResponse) semver.Collection {
	var compatible semver.Collection
	for version := range pkgMeta.Versions {
		semVer, err := semver.NewVersion(version)
		if err != nil {
			continue
		}
		if constraint.Check(semVer) {
			compatible = append(compatible, semVer)
		}
	}
	return compatible
}

func fetchPackage(name, version string) (*npmPackageResponse, error) {
	// Review Comments: Beyond Scope
	// It is best practice to save this URL: https://registry.npmjs.org in a const/ envVar
	// Especially since it is being reused below.
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/%s/%s", name, version))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed npmPackageResponse
	_ = json.Unmarshal(body, &parsed)
	return &parsed, nil
}

func fetchPackageMeta(p string) (*npmPackageMetaResponse, error) {
	// Review Comments: Beyond Scope
	// It is best practice to save this URL: https://registry.npmjs.org in a const/ envVar
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/%s", p))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed npmPackageMetaResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil, err
	}

	return &parsed, nil
}
