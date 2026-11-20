package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// fakeApplicationEnvServer serves application.one and application.saveEnvironment
// for a single application, recording every saveEnvironment payload.
type fakeApplicationEnvServer struct {
	t        *testing.T
	mu       sync.Mutex
	app      map[string]any
	payloads []map[string]any
}

func newFakeApplicationEnvServer(t *testing.T, app map[string]any) (*fakeApplicationEnvServer, *httptest.Server) {
	t.Helper()

	fake := &fakeApplicationEnvServer{t: t, app: app}
	server := httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(server.Close)

	return fake, server
}

func (f *fakeApplicationEnvServer) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch r.URL.Path {
	case "/application.one":
		if err := json.NewEncoder(w).Encode(f.app); err != nil {
			f.t.Errorf("encode application: %v", err)
		}
	case "/application.saveEnvironment":
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			f.t.Errorf("decode saveEnvironment payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.payloads = append(f.payloads, payload)
		for _, field := range []string{"env", "buildArgs", "buildSecrets", "createEnvFile"} {
			f.app[field] = payload[field]
		}
		_, _ = w.Write([]byte("true"))
	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestUpdateApplicationEnvPreservesUnmanagedFields(t *testing.T) {
	t.Parallel()

	fake, server := newFakeApplicationEnvServer(t, map[string]any{
		"applicationId": "app-id",
		"env":           "EXISTING=1",
		"buildArgs":     "ARG=value",
		"buildSecrets":  "SECRET=hunter2",
		"createEnvFile": true,
	})

	c := NewDokployClient(server.URL, "test-api-key")
	err := c.UpdateApplicationEnv("app-id", func(envMap map[string]string) {
		envMap["NEW"] = "2"
	}, nil)
	if err != nil {
		t.Fatalf("UpdateApplicationEnv: %v", err)
	}

	if len(fake.payloads) != 1 {
		t.Fatalf("saveEnvironment calls = %d, want 1", len(fake.payloads))
	}
	payload := fake.payloads[0]

	env, ok := payload["env"].(string)
	if !ok {
		t.Fatalf("env = %#v, want a string", payload["env"])
	}
	if got := ParseEnv(env); got["EXISTING"] != "1" || got["NEW"] != "2" || len(got) != 2 {
		t.Errorf("env = %q, want EXISTING=1 and NEW=2", payload["env"])
	}
	if got, want := payload["buildArgs"], "ARG=value"; got != want {
		t.Errorf("buildArgs = %#v, want %#v", got, want)
	}
	if got, want := payload["buildSecrets"], "SECRET=hunter2"; got != want {
		t.Errorf("buildSecrets = %#v, want %#v", got, want)
	}
	if got, want := payload["createEnvFile"], true; got != want {
		t.Errorf("createEnvFile = %#v, want %#v", got, want)
	}
}

func TestUpdateApplicationEnvOverridesCreateEnvFile(t *testing.T) {
	t.Parallel()

	fake, server := newFakeApplicationEnvServer(t, map[string]any{
		"applicationId": "app-id",
		"env":           "EXISTING=1",
		"buildArgs":     "ARG=value",
		"buildSecrets":  "SECRET=hunter2",
		"createEnvFile": true,
	})

	createEnvFile := false
	c := NewDokployClient(server.URL, "test-api-key")
	err := c.UpdateApplicationEnv("app-id", func(envMap map[string]string) {
		delete(envMap, "EXISTING")
	}, &createEnvFile)
	if err != nil {
		t.Fatalf("UpdateApplicationEnv: %v", err)
	}

	if len(fake.payloads) != 1 {
		t.Fatalf("saveEnvironment calls = %d, want 1", len(fake.payloads))
	}
	payload := fake.payloads[0]

	if got, want := payload["env"], ""; got != want {
		t.Errorf("env = %#v, want %#v", got, want)
	}
	if got, want := payload["buildArgs"], "ARG=value"; got != want {
		t.Errorf("buildArgs = %#v, want %#v", got, want)
	}
	if got, want := payload["buildSecrets"], "SECRET=hunter2"; got != want {
		t.Errorf("buildSecrets = %#v, want %#v", got, want)
	}
	if got, want := payload["createEnvFile"], false; got != want {
		t.Errorf("createEnvFile = %#v, want %#v", got, want)
	}
}
