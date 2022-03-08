package inventory

import "testing"

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    string
		wantErr bool
	}{{
		"complete URL, with prefix and suffix",
		"https://github.com/username/repository.git",
		"github.com/username/repository",
		false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
