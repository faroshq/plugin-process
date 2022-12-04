package v1alpha1

import (
	conditionsv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +crd
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:object:root=true

// Systemd plugin helps to manage systemd services on the device.
type Systemd struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemdSpec   `json:"spec,omitempty"`
	Status SystemdStatus `json:"status,omitempty"`
}

// SystemdSpec defines the desired state of plugin
type SystemdSpec struct {
	Units []Unit `json:"services,omitempty"`
}

type Unit struct {
	// Name of the service
	Name string `json:"name,omitempty"`
	// DesiredStatus is desired status of the service
	DesiredStatus ServiceStatus `json:"desiredState,omitempty"`

	// ActivationMode of the service
	// +optional
	ActivationMode ActivationMode `json:"activationMode,omitempty"`

	// EnableMode of the service
	// +optional
	EnableMode EnableMode `json:"enableMode,omitempty"`
}

type ServiceStatus string

func (s ServiceStatus) String() string {
	return string(s)
}

const (
	ServiceStatusEnabled            ServiceStatus = "enabled"
	ServiceStatusDisabled           ServiceStatus = "disabled"
	ServiceStatusStopped            ServiceStatus = "stopped"
	ServiceStatusStarted            ServiceStatus = "started"
	ServiceStatusEnabledAndStarted  ServiceStatus = "enabled-and-started"
	ServiceStatusDisabledAndStopped ServiceStatus = "disabled-and-stopped"
)

// Takes the unit to activate, plus a mode string. The mode needs to be one of
// replace, fail, isolate, ignore-dependencies, ignore-requirements. If
// "replace" the call will start the unit and its dependencies, possibly
// replacing already queued jobs that conflict with this. If "fail" the call
// will start the unit and its dependencies, but will fail if this would change
// an already queued job. If "isolate" the call will start the unit in question
// and terminate all units that aren't dependencies of it. If
// "ignore-dependencies" it will start a unit but ignore all its dependencies.
// If "ignore-requirements" it will start a unit but only ignore the
// requirement dependencies. It is not recommended to make use of the latter
// two options.
type ActivationMode string

func (s ActivationMode) String() string {
	return string(s)
}

const (
	ActivationModeReplace            ActivationMode = "replace"
	ActivationModeFail               ActivationMode = "fail"
	ActivationModeIsolate            ActivationMode = "isolate"
	ActivationModeIgnoreDependencies ActivationMode = "ignore-dependencies"
	ActivationModeIgnoreRequirements ActivationMode = "ignore-requirements"
)

type EnableMode string

func (s EnableMode) String() string {
	return string(s)
}

const (
	// Enable runtime only ( /run )
	EnableModeRuntimeOnly EnableMode = "runtime"
	// Enable persistently ( /etc )
	EnableModePersistent EnableMode = "persistent"
)

// SystemDStatus defines the observed state of plugin
type SystemdStatus struct {
	// Current processing state of the Agent.
	// +optional
	Conditions conditionsv1alpha1.Conditions `json:"conditions,omitempty"`

	// Units is the list of units managed by the plugin
	// +optional
	Units []UnitStatus `json:"services,omitempty"`
}

type UnitStatus struct {
	// Name of the service
	Name string `json:"name,omitempty"`
	// State defines current state of the service
	Status string `json:"state,omitempty"`
	// DesiredStatus of the service
	DesiredStatus string `json:"desiredState,omitempty"`
	// Error message if the service failed to start
	// +optional
	Error string `json:"error,omitempty"`
}

func (in *Systemd) SetConditions(c conditionsv1alpha1.Conditions) {
	in.Status.Conditions = c
}

func (in *Systemd) GetConditions() conditionsv1alpha1.Conditions {
	return in.Status.Conditions
}

// SystemdList contains a list of plugins
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
type SystemdList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Systemd `json:"items"`
}
