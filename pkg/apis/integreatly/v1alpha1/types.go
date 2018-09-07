package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	ResyncPeriod int
	LogLevel     string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WalkthroughList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Walkthrough `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Walkthrough struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WalkthroughSpec   `json:"spec"`
	Status            WalkthroughStatus `json:"status,omitempty"`
}

type WalkthroughSpec struct {
	UserName string   `json:"username"`
	Services []string `json:"services,omitempty"`
}

type WalkthroughStatus struct {
	// marked as true when all work is done on it
	Ready     bool              `json:"ready"`
	Phase     StatusPhase       `json:"phase"`
	Namespace string            `json:"namespace"`
	Services  map[string]string `json:"services"`
}

type StatusPhase string

var (
	NoPhase                  StatusPhase = ""
	PhaseProvisionNamespace  StatusPhase = "namespace"
	PhaseProvisionServices   StatusPhase = "services"
	PhaseProvisionedServices StatusPhase = "provisioned"
	PhaseUserRoleBindings    StatusPhase = "roles"
	PhaseComplete            StatusPhase = "complete"
)
