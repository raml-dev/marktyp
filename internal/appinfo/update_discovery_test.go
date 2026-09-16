package appinfo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoveryClientReturnsNewestEligibleRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
			{"tag_name":"v1.3.0-rc1","prerelease":true},
			{"tag_name":"v1.2.0","name":"Marktyp 1.2.0","html_url":"https://example.test/v1.2.0"},
			{"tag_name":"v1.1.0"}
		]`))
	}))
	defer server.Close()

	client := &DiscoveryClient{client: server.Client(), endpoint: server.URL}
	response, err := client.GetUpdatesFromRepo(context.Background(), "v1.0.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || response.Release == nil || response.Release.TagName != "v1.2.0" {
		t.Fatalf("unexpected update: %+v", response)
	}
}

func TestFindLatestReleaseVersionRules(t *testing.T) {
	tests := []struct {
		name               string
		current            string
		includePrereleases bool
		want               string
	}{
		{name: "stable excludes release candidates", current: "v1.0.0", want: "v1.2.0"},
		{name: "configured prereleases", current: "v1.0.0", includePrereleases: true, want: "v1.3.0-rc2"},
		{name: "prerelease channel continues", current: "v1.3.0-rc1", want: "v1.3.0-rc2"},
		{name: "development build sees latest", current: "dev", want: "v1.3.0-rc2"},
		{name: "current version has no update", current: "v2.0.0", want: ""},
	}

	releases := []GitHubRelease{
		{TagName: "v1.3.0-rc2", PreRelease: true},
		{TagName: "v1.2.0"},
		{TagName: "v1.1.0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := findLatestRelease(releases, test.current, test.includePrereleases)
			if result == nil {
				if test.want != "" {
					t.Fatalf("expected %s, got no update", test.want)
				}
				return
			}
			if result.TagName != test.want {
				t.Fatalf("got %s, want %s", result.TagName, test.want)
			}
		})
	}
}

func TestDiscoveryClientReportsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	client := &DiscoveryClient{client: server.Client(), endpoint: server.URL}
	if _, err := client.GetUpdatesFromRepo(context.Background(), "v1.0.0", false); err == nil {
		t.Fatal("expected request failure")
	}
}
