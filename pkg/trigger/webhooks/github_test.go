package webhooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-github/v42/github"
	"github.com/onsi/gomega"
	"github.com/otaviof/shipwright-trigger/test/stubs"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

func jsonMarshal(t *testing.T, value interface{}) []byte {
	g := gomega.NewWithT(t)

	jsonBytes, err := json.Marshal(value)
	g.Expect(err).To(gomega.BeNil())
	return jsonBytes
}

func TestGitHubWebHook_ExtractRequestPayload(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		eventType string
		want      *RequestPayload
		wantErr   bool
	}{{
		name:      "empty request won't parse into github event",
		body:      jsonMarshal(t, struct{}{}),
		eventType: "",
		want:      nil,
		wantErr:   true,
	}, {
		name:      "github ping event",
		body:      jsonMarshal(t, stubs.GitHubPingEvent()),
		eventType: "ping",
		want: &RequestPayload{
			EventType: "ping",
			Signature: "",
			Payload:   jsonMarshal(t, stubs.GitHubPingEvent()),
		},
		wantErr: false,
	}, {
		name:      "github push event",
		body:      jsonMarshal(t, stubs.GitHubPushEvent()),
		eventType: "push",
		want: &RequestPayload{
			EventType: "push",
			Signature: "",
			Payload:   jsonMarshal(t, stubs.GitHubPushEvent()),
		},
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(tt.body))
			if err != nil {
				t.Errorf("GitHubWebHook.ExtractRequestPayload() NewRequest() error = %v", err)
			}
			if tt.eventType != "" {
				req.Header.Set(github.EventTypeHeader, tt.eventType)
			}

			g := &GitHubWebHook{}
			got, err := g.ExtractRequestPayload(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubWebHook.ExtractRequestPayload() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if tt.want != nil && got == nil {
				t.Error("GitHubWebHook.ExtractRequestPayload() got nil")
				return
			}
			if got != nil && got.EventType != tt.want.EventType {
				t.Errorf("GitHubWebHook.ExtractRequestPayload() EvenType = '%s', want '%s'",
					got.EventType, tt.want.EventType)
			}
			if got != nil && got.Signature != tt.want.Signature {
				t.Errorf("GitHubWebHook.ExtractRequestPayload() Signature = '%s', want '%s'",
					got.Signature, tt.want.Signature)
			}
			if got != nil && !reflect.DeepEqual(got.Payload, tt.want.Payload) {
				t.Errorf("GitHubWebHook.ExtractRequestPayload() Payload = '%s', want '%s'",
					got.Payload, tt.want.Payload)
			}
		})
	}
}

func TestGitHubWebHook_ExtractBuildSelector(t *testing.T) {
	tests := []struct {
		name    string
		rp      *RequestPayload
		want    *BuildSelector
		wantErr bool
	}{{
		name: "unsupported event",
		rp: &RequestPayload{
			EventType: "Unsupported",
		},
		want:    nil,
		wantErr: true,
	}, {
		name: "ping event",
		rp: &RequestPayload{
			EventType: "ping",
			Signature: "",
			Payload:   jsonMarshal(t, stubs.GitHubPingEvent()),
		},
		want:    &BuildSelector{},
		wantErr: false,
	}, {
		name: "push event",
		rp: &RequestPayload{
			EventType: "push",
			Signature: "",
			Payload:   jsonMarshal(t, stubs.GitHubPushEvent()),
		},
		want: &BuildSelector{
			WhenType:     v1alpha1.WhenTypeGitHub,
			EventName:    string(v1alpha1.GitHubPushEvent),
			RepoURL:      stubs.RepoURL,
			RepoFullName: stubs.RepoFullName,
			Revision:     "main",
		},
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GitHubWebHook{}
			got, err := g.ExtractBuildSelector(tt.rp)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubWebHook.ExtractBuildSelector() error = %q, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GitHubWebHook.ExtractBuildSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
