// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/navikt/mutatingflow/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Notebooks returns a NotebookInformer.
	Notebooks() NotebookInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Notebooks returns a NotebookInformer.
func (v *version) Notebooks() NotebookInformer {
	return &notebookInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
