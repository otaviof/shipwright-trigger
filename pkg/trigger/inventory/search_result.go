package inventory

import (
	"k8s.io/apimachinery/pkg/types"
)

type SearchResult struct {
	BuildName  types.NamespacedName
	SecretName types.NamespacedName
}

func (s *SearchResult) HasSecret() bool {
	return s.SecretName.Namespace != "" && s.SecretName.Name != ""
}
