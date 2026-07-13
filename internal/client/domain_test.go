package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDomainMiddlewarePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		endpoint    string
		middlewares []string
		request     func(*DokployClient) (*Domain, error)
	}{
		{
			name:     "create",
			endpoint: "/domain.create",
			middlewares: []string{
				"auth@file",
				"compress@file",
			},
			request: func(client *DokployClient) (*Domain, error) {
				return client.CreateDomain(Domain{
					Host:        "example.com",
					Path:        "/",
					Port:        3000,
					HTTPS:       true,
					Middlewares: []string{"auth@file", "compress@file"},
				})
			},
		},
		{
			name:     "update",
			endpoint: "/domain.update",
			middlewares: []string{
				"auth@file",
				"compress@file",
			},
			request: func(client *DokployClient) (*Domain, error) {
				return client.UpdateDomain(Domain{
					ID:          "domain-id",
					Host:        "example.com",
					Path:        "/",
					Port:        3000,
					HTTPS:       true,
					Middlewares: []string{"auth@file", "compress@file"},
				})
			},
		},
		{
			name:     "update defaults empty path",
			endpoint: "/domain.update",
			middlewares: []string{
				"auth@file",
				"compress@file",
			},
			request: func(client *DokployClient) (*Domain, error) {
				return client.UpdateDomain(Domain{
					ID:          "domain-id",
					Host:        "example.com",
					Port:        3000,
					HTTPS:       true,
					Middlewares: []string{"auth@file", "compress@file"},
				})
			},
		},
		{
			name:        "clear",
			endpoint:    "/domain.update",
			middlewares: []string{},
			request: func(client *DokployClient) (*Domain, error) {
				return client.UpdateDomain(Domain{
					ID:          "domain-id",
					Host:        "example.com",
					Path:        "/",
					Port:        3000,
					HTTPS:       true,
					Middlewares: []string{},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.endpoint {
					t.Errorf("request path = %q, want %q", r.URL.Path, tt.endpoint)
				}

				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request payload: %v", err)
				}
				want := make([]any, len(tt.middlewares))
				for index, middleware := range tt.middlewares {
					want[index] = middleware
				}
				if got := payload["middlewares"]; !reflect.DeepEqual(got, want) {
					t.Errorf("middlewares = %#v, want %#v", got, want)
				}
				if got, want := payload["path"], "/"; got != want {
					t.Errorf("path = %#v, want %#v", got, want)
				}

				if err := json.NewEncoder(w).Encode(map[string]any{
					"domain": map[string]any{
						"domainId":    "domain-id",
						"middlewares": tt.middlewares,
					},
				}); err != nil {
					t.Errorf("encode response payload: %v", err)
				}
			}))
			defer server.Close()

			domain, err := tt.request(NewDokployClient(server.URL, "test-api-key"))
			if err != nil {
				t.Fatalf("request domain: %v", err)
			}
			if got, want := domain.Middlewares, tt.middlewares; !reflect.DeepEqual(got, want) {
				t.Errorf("response middlewares = %#v, want %#v", got, want)
			}
		})
	}
}
