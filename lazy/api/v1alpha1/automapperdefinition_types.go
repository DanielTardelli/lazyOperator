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
	// An array of all currently defined relationships between any cluster resources specified
	Relationships []AutoMapperDefinitionObject `json:"autoMapperObjects"`
}

// Defines the GVK of a resource
type AutoMapperDefinitionGVK struct {
	// The group metadata value for the resource as specified in the CRD
	Group string `json:"group"`
	// The version metadata value for the resource as specified in the CRD
	Version string `json:"version"`
	// The kind metadata value (AKA the name) for the resource as specified in the CRD
	Kind string `json:"kind"`
}

// Defines a label kv pair
type AutoMapperDefinitionLabel struct {
	// Key for the label
	Key string `json:"key"`
	// Value for the label
	Value string `json:"value"`
}

// Defines a singular relationship and its desired state at all time
type AutoMapperDefinitionObject struct {
	// The GVK of the source resource which will be watched by the controller to decide
	// how to administer the result resources
	Source AutoMapperDefinitionGVK `json:"source"`
	// The GVK of the result resource that will determine what resource will be provisioned
	// in response to a source resource dependent on the basis of the relationship
	Result AutoMapperDefinitionGVK `json:"result"`

	// ENUM['cluster', 'label', 'namespace'] - Defines the basis of the relationship, its
	// either cluster wide, based on resource labels or based on resource namespaces.
	// +kubebuilder:validation:Enum=cluster;label;namespace
	Basis string `json:"basis"`
	// If the basis is equal to label, this map will have the key value pair of the label
	// that is to be watched for this relationship
	Label *AutoMapperDefinitionLabel `json:"labelBasis"`
	// If the basis is equal to namespace, this slice will contain the namespace that will be
	// watched for this relationship
	Namespace *string `json:"namespaceBasis"`

	// The mappings of variables to the result resources both static and dynamic/referenced
	VarMap *[]AutoMapperVariableMap `json:"varMap,omitempty"`

	// The unique name for this object
	ObjectName string `json:"objectName"`
}

// Object designed to make locating relationships easier in controller loop
type AutoMapperRelLocator struct {
	// The namespace of the resultant resource
	Namespace string `json:"namespace"`
	// The name of the resultant resource
	Name string `json:"name"`
	// Assigned objectname from relationship
	ObjectName string `json:"objectName"`
}

// AutoMapperDefinitionStatus defines the observed state of AutoMapperDefinition
type AutoMapperDefinitionStatus struct {
	// Whether or not the cluster's most recently analysed state was able to be
	// brought into alignment with active relationships
	Reconciled bool `json:"reconciled"`
	// An array of all relationships maintained by this AutoMapperDefinition identified
	// by ObjectName
	Relationships []AutoMapperRelLocator `json:"relationships"`
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
