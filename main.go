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
		fmt.Println("Missing API url")
		os.Exit(1)
	}

	apiUser := os.Getenv("API_USER")
	if apiUser == "" {
		fmt.Println("Missing API user")
		os.Exit(1)
	}

	apiPassword := os.Getenv("API_PASSWORD")
	if apiPassword == "" {
		fmt.Println("Missing API password")
		os.Exit(1)
	}

	packageOwner := os.Getenv("PACKAGE_OWNER")
	if packageOwner == "" {
		fmt.Println("Missing package owner")
		os.Exit(1)
	}

	packageName := os.Getenv("PACKAGE_NAME")
	if packageName == "" {
		fmt.Println("Missing package name")
		os.Exit(1)
	}

	packageVersion := os.Getenv("PACKAGE_VERSION")
	if packageVersion == "" {
		fmt.Println("Missing package version")
		os.Exit(1)
	}

	fail := os.Getenv("FAIL")
	resultVar := os.Getenv("RESULT_VAR")

	result := "failure"

	if resultVar != "" {
		defer func() {
			response, err := http.Post(fmt.Sprintf("%s/api/v1/var?key=%s", reeveAPI, url.QueryEscape(resultVar)), "text/plain", strings.NewReader(result))
			if err != nil {
				fmt.Printf("Error setting result var - %s\n", err)
				os.Exit(1)
			}
			if response.StatusCode != http.StatusOK {
				fmt.Printf("Setting result var returned status %v\n", response.StatusCode)
				os.Exit(1)
			}
			fmt.Printf("Set %s=%s\n", resultVar, result)
		}()
	}

	var packageFiles []string
	requestUrl := fmt.Sprintf("%s/api/v1/packages/%s/generic/%s/%s/files", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageVersion))
	request, err := http.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		fmt.Printf("Error fetching package file list - %s\n", err)
		os.Exit(1)
	}
	request.SetBasicAuth(apiUser, apiPassword)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Printf("Error fetching package file list - %s\n", err)
		os.Exit(1)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			fmt.Printf("Fetching package file list returned status %v\n", response.StatusCode)
			os.Exit(1)
		}
	} else {
		var packageList []struct {
			Name string `json:"name"`
		}
		err := json.NewDecoder(response.Body).Decode(&packageList)
		response.Body.Close()
		if err != nil {
			fmt.Printf("Error fetching package file list - %s\n", err)
			os.Exit(1)
		}
		packageFiles = make([]string, 0, len(packageList))
		for _, packageFile := range packageList {
			packageFiles = append(packageFiles, packageFile.Name)
		}
	}

	filePatterns, err := shlex.Split(os.Getenv("FILES"))
	if err != nil {
		fmt.Printf("Error parsing file pattern list - %s\n", err)
		os.Exit(1)
	}
	var found bool
L:
	for _, pattern := range filePatterns {
		for _, filename := range packageFiles {
			found, err = doublestar.Match(filepath.ToSlash(pattern), filename)
			if err != nil {
				fmt.Printf("Error parsing file pattern \"%s\" - %s\n", pattern, err)
				os.Exit(1)
			}
			if found {
				break L
			}
		}
	}

	if found {
		result = "exists"
		fmt.Println("A file matching the pattern already exists in the package")
	} else {
		result = "does-not-exist"
		fmt.Println("A file matching the pattern does not exist in the package")
	}

	if fail == result {
		os.Exit(1)
	}
}
