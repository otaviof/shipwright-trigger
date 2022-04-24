package inventory

import (
	"log"
	"sync"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

// Inventory keeps track of Build object details, on which it can find objects that match the
// repository URL and trigger rules.
type Inventory struct {
	m sync.Mutex

	cache map[types.NamespacedName]TriggerRules // cache storage
}

var _ Interface = &Inventory{}

// TriggerRules keeps the source and webhook trigger information for each Build instance.
type TriggerRules struct {
	source  v1alpha1.Source
	trigger v1alpha1.Trigger
}

// SearchFn search function signature.
type SearchFn func(TriggerRules) bool

// Add insert or update an existing record.
func (i *Inventory) Add(b *v1alpha1.Build) {
	i.m.Lock()
	defer i.m.Unlock()

	if b.Spec.Trigger == nil {
		b.Spec.Trigger = &v1alpha1.Trigger{}
	}
	buildName := types.NamespacedName{Namespace: b.GetNamespace(), Name: b.GetName()}
	log.Printf("Storing Build %q (generation %d) on the inventory", buildName, b.GetGeneration())
	i.cache[buildName] = TriggerRules{
		source:  b.Spec.Source,
		trigger: *b.Spec.Trigger,
	}
}

// Remove the informed entry from the cache.
func (i *Inventory) Remove(buildName types.NamespacedName) {
	i.m.Lock()
	defer i.m.Unlock()

	log.Printf("Removing Build %q from the inventory", buildName)
	if _, ok := i.cache[buildName]; !ok {
		log.Printf("Inventory entry is not found, skipping deletion!")
		return
	}
	delete(i.cache, buildName)
}

// loopByWhenType execute the search function informed against each inventory entry, when it returns
// true it returns the build name on the search results instance.
func (i *Inventory) loopByWhenType(whenType v1alpha1.WhenTypeName, fn SearchFn) []SearchResult {
	found := []SearchResult{}
	for k, v := range i.cache {
		for _, when := range v.trigger.When {
			if whenType != when.Type {
				continue
			}
			if fn(v) {
				secretName := types.NamespacedName{}
				if v.trigger.SecretRef != nil {
					secretName.Namespace = k.Namespace
					secretName.Name = v.trigger.SecretRef.Name
				}
				found = append(found, SearchResult{
					BuildName:  k,
					SecretName: secretName,
				})
			}
		}
	}
	log.Printf("Found %d Build(s) for %q", len(found), whenType)
	return found
}

// SearchForObjectRef search for builds using the ObjectRef as query parameters.
func (i *Inventory) SearchForObjectRef(
	whenType v1alpha1.WhenTypeName,
	objectRef *v1alpha1.WhenObjectRef,
) []SearchResult {
	i.m.Lock()
	defer i.m.Unlock()

	return i.loopByWhenType(whenType, func(tr TriggerRules) bool {
		for _, w := range tr.trigger.When {
			if w.ObjectRef == nil {
				continue
			}

			// checking the desired status, it must what's informed on the Build object
			if len(w.ObjectRef.Status) > 0 && len(objectRef.Status) > 0 {
				status := objectRef.Status[0]
				if !StringSliceContains(status, w.ObjectRef.Status) {
					continue
				}
			}

			// when name is informed it will try to match it first, otherwise the label selector
			// matching will take place
			if w.ObjectRef.Name != "" {
				if objectRef.Name != w.ObjectRef.Name {
					continue
				}
			} else {
				if len(w.ObjectRef.Selector) == 0 || len(objectRef.Selector) == 0 {
					continue
				}
				// transforming the matching labels passed to this method as a regular label selector
				// instance, which is employed to match against the Build trigger definition
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: w.ObjectRef.Selector,
				})
				if err != nil {
					log.Printf("Unable to parse '%#v' as label-selector: %q",
						w.ObjectRef.Selector, err)
					continue
				}
				if !selector.Matches(labels.Set(objectRef.Selector)) {
					continue
				}
			}
			return true
		}
		return false
	})
}

// SearchForGit search for builds using the Git repository details, like the URL, branch name and
// such type of information.
func (i *Inventory) SearchForGit(whenType v1alpha1.WhenTypeName, repoURL, branch string) []SearchResult {
	i.m.Lock()
	defer i.m.Unlock()

	return i.loopByWhenType(whenType, func(tr TriggerRules) bool {
		// first thing to compare, is the repository URL, it must match in order to define the actual
		// builds that are representing the repository
		if !CompareURLs(repoURL, *tr.source.URL) {
			return false
		}

		// second part is to search for event-type and compare the informed branch, with the allowed
		// branches, configured for that build
		for _, w := range tr.trigger.When {
			branches := w.GetBranches(whenType)
			for _, b := range branches {
				if branch == b {
					log.Printf("Repository URL %q (%q) matches criteria", repoURL, branch)
					return true
				}
			}
		}

		return false
	})
}

// NewInventory instantiate the inventory.
func NewInventory() *Inventory {
	return &Inventory{cache: map[types.NamespacedName]TriggerRules{}}
}
