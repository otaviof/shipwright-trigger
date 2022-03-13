package inventory

import (
	"reflect"
	"testing"

	"github.com/onsi/gomega"
	"github.com/otaviof/shipwright-trigger/test/stubs"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var buildWithTrigger = stubs.ShipwrightBuildWithTriggers("name", stubs.TriggerWhenPushToMain)

func TestInventory(t *testing.T) {
	g := gomega.NewWithT(t)

	i := NewInventory()

	t.Run("adding empty inventory item", func(_ *testing.T) {
		i.Add(&buildWithTrigger)
		g.Expect(len(i.cache)).To(gomega.Equal(1))

		_, exists := i.cache[types.NamespacedName{Namespace: "namespace", Name: "name"}]
		g.Expect(exists).To(gomega.BeTrue())
	})

	t.Run("remove inventory item", func(_ *testing.T) {
		i.Remove(types.NamespacedName{Namespace: "namespace", Name: "name"})
		g.Expect(len(i.cache)).To(gomega.Equal(0))
	})
}

func TestInventorySearchForgit(t *testing.T) {
	g := gomega.NewWithT(t)

	i := NewInventory()
	i.Add(&buildWithTrigger)

	t.Run("should not find any results", func(_ *testing.T) {
		found := i.SearchForGit(v1alpha1.WhenPush, "", "")
		g.Expect(len(found)).To(gomega.Equal(0))

		found = i.SearchForGit(v1alpha1.WhenPush, stubs.RepoURL, "")
		g.Expect(len(found)).To(gomega.Equal(0))
	})

	t.Run("should find the build object", func(_ *testing.T) {
		found := i.SearchForGit(v1alpha1.WhenPush, stubs.RepoURL, "main")
		g.Expect(len(found)).To(gomega.Equal(1))
	})
}

func TestInventory_SearchForObjectRef(t *testing.T) {
	buildWithObjectRefName := v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "buildname",
		},
		Spec: v1alpha1.BuildSpec{
			Trigger: &v1alpha1.Trigger{
				When: []v1alpha1.TriggerWhen{{
					Type: v1alpha1.WhenPipeline,
					ObjectRef: &v1alpha1.ObjectRef{
						Name:   "name",
						Status: []string{"Successful"},
					},
				}},
			},
		},
	}
	buildWithObjectRefSelector := v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"k": "v"},
			Namespace: "namespace",
			Name:      "buildname",
		},
		Spec: v1alpha1.BuildSpec{
			Trigger: &v1alpha1.Trigger{
				When: []v1alpha1.TriggerWhen{{
					Type: v1alpha1.WhenPipeline,
					ObjectRef: &v1alpha1.ObjectRef{
						Status:   []string{"Successful"},
						Selector: map[string]string{"k": "v"},
					},
				}},
			},
		},
	}

	tests := []struct {
		name      string
		builds    []v1alpha1.Build
		whenType  v1alpha1.WhenType
		objectRef v1alpha1.ObjectRef
		want      []SearchResult
	}{{
		name:     "find build by name",
		builds:   []v1alpha1.Build{buildWithObjectRefName},
		whenType: v1alpha1.WhenPipeline,
		objectRef: v1alpha1.ObjectRef{
			Name:   "name",
			Status: []string{"Successful"},
		},
		want: []SearchResult{{
			BuildName: types.NamespacedName{Namespace: "namespace", Name: "buildname"},
		}},
	}, {
		name:     "find build by label selector",
		builds:   []v1alpha1.Build{buildWithObjectRefSelector},
		whenType: v1alpha1.WhenPipeline,
		objectRef: v1alpha1.ObjectRef{
			Status:   []string{"Successful"},
			Selector: map[string]string{"k": "v"},
		},
		want: []SearchResult{{
			BuildName: types.NamespacedName{Namespace: "namespace", Name: "buildname"},
		}},
	}, {
		name:     "does not find builds, due to wrong selector",
		builds:   []v1alpha1.Build{buildWithObjectRefSelector},
		whenType: v1alpha1.WhenPipeline,
		objectRef: v1alpha1.ObjectRef{
			Status:   []string{"Successful"},
			Selector: map[string]string{"wrong": "label"},
		},
		want: []SearchResult{},
	}, {
		name:     "does not find builds, due to wrong name",
		builds:   []v1alpha1.Build{buildWithObjectRefSelector},
		whenType: v1alpha1.WhenPipeline,
		objectRef: v1alpha1.ObjectRef{
			Name:   "wrong",
			Status: []string{"Successful"},
		},
		want: []SearchResult{},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewInventory()
			for _, b := range tt.builds {
				i.Add(&b)
			}

			got := i.SearchForObjectRef(tt.whenType, &tt.objectRef)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Inventory.SearchForObjectRef() = %v, want %v", got, tt.want)
			}
		})
	}
}
