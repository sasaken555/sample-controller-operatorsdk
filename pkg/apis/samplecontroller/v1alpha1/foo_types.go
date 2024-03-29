package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FooSpec defines the desired state of Foo
// +k8s:openapi-gen=true
type FooSpec struct {
	// +kubebuilder:validation:Format=string
	DeploymentName string `json:"deploymentName"`

	// +kubebuilder:validation:Minimum=1
	Replicas *int32 `json:"replicas"`
}

// FooStatus defines the observed state of Foo
// +k8s:openapi-gen=true
type FooStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Foo is the Schema for the foos API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FooSpec   `json:"spec,omitempty"`
	Status FooStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FooList contains a list of Foo
type FooList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Foo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Foo{}, &FooList{})
}
