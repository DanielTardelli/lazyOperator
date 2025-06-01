/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AutoMapperDefinitionSpec defines the desired state of AutoMapperDefinition
type AutoMapperDefinitionSpec struct {
	// Maps: List of maps of relationships between resources and their config
	Maps []AutoMapperObject `json:"autoMapperObjects"`
}

// DT: Required to define the various resources we'll be referring to so we
// don't get tightly coupled versioning dependencies
type AutoMapperDefinitionGVK struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// DT: AutoMapperObject defines a singular relationship between two resources
type AutoMapperObject struct {
	// Source:          Resource type that is being watched
	// Result:          Resource type that is provisioned as a result
	// Basis:           Relationship basis, by label, namespace or cluster-wide
	//	LabelBasis:     Array of labels to watch
	//  NamespaceBasis: Array of namespaces to watch
	// DeclaredVars:    Values in resulting resource declared statically
	// VarMap:          Values in resulting resource dependent on source (both static and dynamic)
	Source AutoMapperDefinitionGVK `json:"source"`
	Result AutoMapperDefinitionGVK `json:"result"`

	// +kubebuilder:validation:Enum=cluster;label;namespace
	Basis          string    `json:"basis"`
	LabelBasis     *[]string `json:"labelBasis"`
	NamespaceBasis *[]string `json:"namespaceBasis"`

	VarMap *[]AutoMapperVariableMap `json:"varMap,omitempty"`
}

// DT : just maps a source attribute to a dest attribute, interpreted based on
// how it is invoked whether declared or referenced
type AutoMapperVariableMap struct {
	// Type:           The variable static or referenced from source
	// SourceVar:      Can be a json path to source resource or a static value
	// DestinationVar: JSON path for where to put var in result resource

	// +kubebuilder:validation:Enum=static;referenced
	Type           string `json:"type"`
	SourceVar      string `json:"sourceVar"`
	DestinationVar string `json:"destinationVar"`
}

// AutoMapperDefinitionStatus defines the observed state of AutoMapperDefinition
type AutoMapperDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	MappedItems map[string]AutoMapperDefinitionStatusMapped `json:"mapped,omitempty"`
}

// DT: AutoMapperDefinitionStatusMapped is to keep a list of all relationships defined by
// user
type AutoMapperDefinitionStatusMapped struct {
	// Source:    What resource is the controller looking for
	// Result:    What resource is provisioned as a result
	// Basis:     Is it based on namespace, cluster-wide, label?
	// BasisVars: Associated variables with the basis
	// Created:   Whether the relationship and resultant resources have been created
	// Resources: The provisioned resources as a result of this relationship
	Source    AutoMapperDefinitionGVK  `json:"source"`
	Result    AutoMapperDefinitionGVK  `json:"result"`
	Basis     string                   `json:"basis"`
	BasisVars []string                 `json:"basisVars"`
	Created   bool                     `json:"created"`
	VarMap    *[]AutoMapperVariableMap `json:"varMap"`

	Resources map[string]AutoMapperDefinitionStatusResource `json:"resources,omitempty"`
}

// DT: AutoMapperDefinitionStatusResource is just the name and namespace of
// the resource to keep track of it like an inventory
type AutoMapperDefinitionStatusResource struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`

	GVK AutoMapperDefinitionGVK `json:"gvk"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AutoMapperDefinition is the Schema for the automapperdefinitions API
type AutoMapperDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutoMapperDefinitionSpec   `json:"spec,omitempty"`
	Status AutoMapperDefinitionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AutoMapperDefinitionList contains a list of AutoMapperDefinition
type AutoMapperDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AutoMapperDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AutoMapperDefinition{}, &AutoMapperDefinitionList{})
}
