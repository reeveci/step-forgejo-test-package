package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/google/shlex"
)

func main() {
	reeveAPI := os.Getenv("REEVE_API")
	if reeveAPI == "" {
		fmt.Println("This docker image is a Reeve CI pipeline step and is not intended to be used on its own.")
		os.Exit(1)
	}

	apiUrl := os.Getenv("API_URL")
	if apiUrl == "" {
		panic("missing API url")
	}

	apiUser := os.Getenv("API_USER")
	if apiUser == "" {
		panic("missing API user")
	}

	apiPassword := os.Getenv("API_PASSWORD")
	if apiPassword == "" {
		panic("missing API password")
	}

	packageOwner := os.Getenv("PACKAGE_OWNER")
	if packageOwner == "" {
		panic("missing package owner")
	}

	packageName := os.Getenv("PACKAGE_NAME")
	if packageName == "" {
		panic("missing package name")
	}

	packageVersion := os.Getenv("PACKAGE_VERSION")
	if packageVersion == "" {
		panic("missing package version")
	}

	fail := os.Getenv("FAIL")
	resultVar := os.Getenv("RESULT_VAR")

	result := "failure"

	if resultVar != "" {
		defer func() {
			response, err := http.Post(fmt.Sprintf("%s/api/v1/var?key=%s", reeveAPI, url.QueryEscape(resultVar)), "text/plain", strings.NewReader(result))
			if err != nil {
				panic(fmt.Sprintf("error setting result var - %s", err))
			}
			if response.StatusCode != http.StatusOK {
				panic(fmt.Sprintf("setting result var returned status %v", response.StatusCode))
			}
			fmt.Printf("Set %s=%s\n", resultVar, result)
		}()
	}

	var packageFiles []string
	requestUrl := fmt.Sprintf("%s/api/v1/packages/%s/generic/%s/%s/files", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageVersion))
	request, err := http.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		panic(fmt.Sprintf(`error fetching package file list - %s`, err))
	}
	request.SetBasicAuth(apiUser, apiPassword)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(fmt.Sprintf(`error fetching package file list - %s`, err))
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			panic(fmt.Sprintf("fetching package file list returned status %v", response.StatusCode))
		}
	} else {
		var packageList []struct {
			Name string `json:"name"`
		}
		err := json.NewDecoder(response.Body).Decode(&packageList)
		response.Body.Close()
		if err != nil {
			panic(fmt.Sprintf(`error fetching package file list - %s`, err))
		}
		packageFiles = make([]string, 0, len(packageList))
		for _, packageFile := range packageList {
			packageFiles = append(packageFiles, packageFile.Name)
		}
	}

	filePatterns, err := shlex.Split(os.Getenv("FILES"))
	if err != nil {
		panic(fmt.Sprintf("error parsing file pattern list - %s", err))
	}
	var found bool
L:
	for _, pattern := range filePatterns {
		for _, filename := range packageFiles {
			found, err = doublestar.Match(filepath.ToSlash(pattern), filename)
			if err != nil {
				panic(fmt.Sprintf(`error parsing file pattern "%s" - %s`, pattern, err))
			}
			if found {
				break L
			}
		}
	}

	if found {
		result = "exists"

		if fail == "exists" {
			panic("a file matching the pattern already exists in the package")
		} else {
			fmt.Println("A file matching the pattern does exist in the package")
		}
	} else {
		result = "does-not-exist"

		if fail == "does-not-exist" {
			panic("a file matching the pattern does not exist in the package")
		} else {
			fmt.Println("A file matching the pattern does not exist in the package")
		}
	}
}
