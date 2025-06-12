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
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DT : just maps a source attribute to a dest attribute, interpreted based on
// how it is invoked whether declared or referenced
type AutoMapperVariableMap struct {
	// ENUM['static', 'referenced'] - Defines whether the variable will be defined
	// by the user or will reference a resource attribute from the source
	// +kubebuilder:validation:Enum=static;referenced
	Type string `json:"type"`
	// Either the static value or JSON path to the attribute in accordance to the above
	SourceVar apiv1.JSON `json:"sourceVar"`
	// The JSON path where the value will be deposited in the resultant resource
	DestinationVar string `json:"destinationVar"`
}

// AutoMapperRelationshipSpec defines the desired state of AutoMapperRelationship
type AutoMapperRelationshipSpec struct {
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
	// If the basis is equal to label, this map will have label key value pair that is to be
	// watched for this relationship
	Label *AutoMapperDefinitionLabel `json:"labels,omitempty"`
	// If the basis is equal to namespace, this slice will contain the namespace that is to
	// be watched for this relationship
	Namespace *string `json:"namespace,omitempty"`
	// The mappings of variables to the result resources both static and dynamic/referenced
	VarMap *[]AutoMapperVariableMap `json:"varMap"`
}

// AutoMapperRelationshipStatusResource is for inventory purposes on currently maintained resources
type AutoMapperRelationshipStatusResource struct {
	// The namespace of the resultant resource
	Namespace string `json:"namespace"`
	// The name of the resultant resource
	Name string `json:"name"`
	// The GVK of the resultant resource
	GVK AutoMapperDefinitionGVK `json:"gvk"`
}

// AutoMapperRelationshipStatusMappings is for holding the inventory details of the source and dest
// resources in the given relationship
type AutoMapperRelationshipStatusMappings struct {
	// The GVK and namespaced name of the source resource
	Source AutoMapperRelationshipStatusResource `json:"source"`
	// The GVK and namespaced name of the result resource
	Result AutoMapperRelationshipStatusResource `json:"result"`
}

// AutoMapperRelationshipStatus defines the observed state of AutoMapperRelationship
type AutoMapperRelationshipStatus struct {
	// A map of all resources being managed by this relationship as a result of the relationship,
	// with the string key being a hash of the resource attributes
	Resources []AutoMapperRelationshipStatusMappings `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AutoMapperRelationship is the Schema for the automapperrelationships API
type AutoMapperRelationship struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutoMapperRelationshipSpec   `json:"spec,omitempty"`
	Status AutoMapperRelationshipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AutoMapperRelationshipList contains a list of AutoMapperRelationship
type AutoMapperRelationshipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AutoMapperRelationship `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AutoMapperRelationship{}, &AutoMapperRelationshipList{})
}
