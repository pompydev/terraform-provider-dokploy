package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pompydev/terraform-provider-dokploy/internal/client"
)

// Regression tests for saveEnvironment: application.saveEnvironment overwrites
// env, buildArgs, buildSecrets and createEnvFile together, so any of them not
// set in Terraform must be sent back with the value Dokploy already has.

const (
	remoteEnv          = "UI_MANAGED=1\nRAILPACK_START_CMD=node build"
	remoteBuildArgs    = "ARG=value"
	remoteBuildSecrets = "SECRET=hunter2"
)

type envRequestRecorder struct {
	t        *testing.T
	mu       sync.Mutex
	reads    int
	payloads []map[string]any
}

func newEnvTestResource(t *testing.T) (*ApplicationResource, *envRequestRecorder) {
	t.Helper()

	rec := &envRequestRecorder{t: t}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		defer rec.mu.Unlock()

		switch r.URL.Path {
		case "/application.one":
			rec.reads++
			if err := json.NewEncoder(w).Encode(map[string]any{
				"applicationId": "app-id",
				"env":           remoteEnv,
				"buildArgs":     remoteBuildArgs,
				"buildSecrets":  remoteBuildSecrets,
				"createEnvFile": true,
			}); err != nil {
				t.Errorf("encode application: %v", err)
			}
		case "/application.saveEnvironment":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode saveEnvironment payload: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			rec.payloads = append(rec.payloads, payload)
			_, _ = w.Write([]byte("true"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return &ApplicationResource{client: client.NewDokployClient(server.URL, "test-api-key")}, rec
}

func TestSaveEnvironmentPreservesUnmanagedFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		plan  ApplicationResourceModel
		want  map[string]any
		calls int
	}{
		{
			name: "only create_env_file set",
			plan: ApplicationResourceModel{
				Env:           types.StringNull(),
				BuildArgs:     types.StringNull(),
				BuildSecrets:  types.StringNull(),
				CreateEnvFile: types.BoolValue(false),
			},
			want: map[string]any{
				"env":           remoteEnv,
				"buildArgs":     remoteBuildArgs,
				"buildSecrets":  remoteBuildSecrets,
				"createEnvFile": false,
			},
			calls: 1,
		},
		{
			name: "only env set, create_env_file unknown",
			plan: ApplicationResourceModel{
				Env:           types.StringValue("MANAGED=1"),
				BuildArgs:     types.StringNull(),
				BuildSecrets:  types.StringNull(),
				CreateEnvFile: types.BoolUnknown(),
			},
			want: map[string]any{
				"env":           "MANAGED=1",
				"buildArgs":     remoteBuildArgs,
				"buildSecrets":  remoteBuildSecrets,
				"createEnvFile": true,
			},
			calls: 1,
		},
		{
			name: "unknown values are not sent",
			plan: ApplicationResourceModel{
				Env:           types.StringUnknown(),
				BuildArgs:     types.StringUnknown(),
				BuildSecrets:  types.StringValue("NEW_SECRET=1"),
				CreateEnvFile: types.BoolNull(),
			},
			want: map[string]any{
				"env":           remoteEnv,
				"buildArgs":     remoteBuildArgs,
				"buildSecrets":  "NEW_SECRET=1",
				"createEnvFile": true,
			},
			calls: 1,
		},
		{
			name: "explicit empty strings clear",
			plan: ApplicationResourceModel{
				Env:           types.StringValue(""),
				BuildArgs:     types.StringValue(""),
				BuildSecrets:  types.StringValue(""),
				CreateEnvFile: types.BoolValue(true),
			},
			want: map[string]any{
				"env":           "",
				"buildArgs":     "",
				"buildSecrets":  "",
				"createEnvFile": true,
			},
			calls: 1,
		},
		{
			name: "nothing set sends nothing",
			plan: ApplicationResourceModel{
				Env:           types.StringNull(),
				BuildArgs:     types.StringNull(),
				BuildSecrets:  types.StringNull(),
				CreateEnvFile: types.BoolNull(),
			},
			calls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, rec := newEnvTestResource(t)
			if err := r.saveEnvironment("app-id", &tt.plan); err != nil {
				t.Fatalf("saveEnvironment: %v", err)
			}

			if len(rec.payloads) != tt.calls {
				t.Fatalf("saveEnvironment calls = %d, want %d", len(rec.payloads), tt.calls)
			}
			if tt.calls == 0 {
				if rec.reads != 0 {
					t.Errorf("application.one reads = %d, want 0", rec.reads)
				}
				return
			}

			payload := rec.payloads[0]
			if got, want := payload["applicationId"], "app-id"; got != want {
				t.Errorf("applicationId = %#v, want %#v", got, want)
			}
			for field, want := range tt.want {
				if got := payload[field]; got != want {
					t.Errorf("%s = %#v, want %#v", field, got, want)
				}
			}
		})
	}
}
