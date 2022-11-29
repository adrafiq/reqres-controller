/*
Copyright 2022.

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

// USERSpec defines the desired state of USER
type USERSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	Email string `json:"email"`
	// +kubebuilder:validation:Required
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
}

// USERStatus defines the observed state of USER
type USERStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// USER is the Schema for the users API
type USER struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   USERSpec   `json:"spec,omitempty"`
	Status USERStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// USERList contains a list of USER
type USERList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []USER `json:"items"`
}

func init() {
	SchemeBuilder.Register(&USER{}, &USERList{})
}
