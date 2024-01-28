/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MemoryLimitSpec defines the desired state of MemoryLimit
type MemoryLimitSpec struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	LimitPercentage   int32                `json:"limitPercentage,omitempty"`
	RequeueTime       metav1.Duration      `json:"requeueTime,omitempty"`
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

// MemoryLimitStatus defines the observed state of MemoryLimit
type MemoryLimitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// MemoryLimit is the Schema for the memorylimits API
type MemoryLimit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemoryLimitSpec   `json:"spec,omitempty"`
	Status MemoryLimitStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MemoryLimitList contains a list of MemoryLimit
type MemoryLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MemoryLimit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MemoryLimit{}, &MemoryLimitList{})
}
