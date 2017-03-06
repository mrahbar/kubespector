package kubernetes

type PodList struct {
	Items []Pod `json:"items"`
}

type Pod struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PodSpec `json:"spec,omitempty"`
}

type ObjectMeta struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ObjectReference contains enough information to let you inspect or modify the referred object.
type ObjectReference struct {
	Kind      string
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type SerializedReference struct {
	TypeMeta
	Reference ObjectReference
}

// PodSpec is a description of a pod.
type PodSpec struct {
	NodeName   string      `json:"nodeName"`
	Volumes    []Volume    `json:"volumes,omitempty"`
	Containers []Container `json:"containers"`
}

// Volume represents a named volume in a pod that may be accessed by any container in the pod.
type Volume struct {
	Name         string `json:"name"`
	VolumeSource `json:",inline"`
}

// Represents the source of a volume to mount.
// Only one of its members may be specified.
type VolumeSource struct {
	HostPath              *HostPathVolumeSource              `json:"hostPath,omitempty"`
	EmptyDir              *EmptyDirVolumeSource              `json:"emptyDir,omitempty"`
	PersistentVolumeClaim *PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
}

// Represents a host path mapped into a pod.
type HostPathVolumeSource struct {
	Path string `json:"path"`
}

// Represents an empty directory for a pod.
type EmptyDirVolumeSource struct {
	Medium string `json:"medium,omitempty"`
}

// PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace.
// This volume finds the bound PV and mounts that volume for the pod. A
// PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another
// type of volume that is owned by someone else (the system).
type PersistentVolumeClaimVolumeSource struct {
	ClaimName string `json:"claimName"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// A single application container that you want to run within a pod.
type Container struct {
	Name         string        `json:"name"`
	VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`
}

// VolumeMount describes a mounting of a Volume within a container.
type VolumeMount struct {
	Name      string `json:"name"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
	MountPath string `json:"mountPath"`
	// Path within the volume from which the container's volume should be mounted
	SubPath string `json:"subPath,omitempty"`
}

// PersistentVolumeList is a list of PersistentVolume items.
type PersistentVolumeList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata,omitempty"`
	Items    []PersistentVolume `json:"items"`
}

// PersistentVolume (PV) is a storage resource provisioned by an administrator.
// It is analogous to a node.
type PersistentVolume struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PersistentVolumeSpec   `json:"spec,omitempty"`
	Status     PersistentVolumeStatus `json:"status,omitempty"`
}

// Similar to VolumeSource but meant for the administrator who creates PVs.
type PersistentVolumeSource struct {
	// HostPath represents a directory on the host.
	HostPath *HostPathVolumeSource
}

// PersistentVolumeClaim is a user's request for and claim to a persistent volume
type PersistentVolumeClaim struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the volume requested by a pod author
	Spec PersistentVolumeClaimSpec
}

// PersistentVolumeClaimSpec describes the common attributes of storage devices
// and allows a Source for provider-specific attributes
type PersistentVolumeClaimSpec struct {
	// VolumeName is the binding reference to the PersistentVolume backing this
	// claim. When set to non-empty value Selector is not evaluated
	VolumeName string
}

// TypeMeta describes an individual object in an API response or request
// with strings representing the type of the object and its API schema version.
// Structures that are versioned or persisted should inline TypeMeta.
type TypeMeta struct {
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// ListMeta describes metadata that synthetic resources must have, including lists and
// various status objects. A resource may have only one of {ObjectMeta, ListMeta}.
type ListMeta struct {
	SelfLink        string `json:"selfLink,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

// PersistentVolumeSpec is the specification of a persistent volume.
type PersistentVolumeSpec struct {
	PersistentVolumeSource
	// ClaimRef is part of a bi-directional binding between PersistentVolume and PersistentVolumeClaim.
	// Expected to be non-nil when bound.
	// claim.VolumeName is the authoritative bind between PV and PVC.
	ClaimRef *ObjectReference `json:"claimRef,omitempty"`
}

type PersistentVolumePhase string

// PersistentVolumeStatus is the current status of a persistent volume.
type PersistentVolumeStatus struct {
	// Phase indicates if a volume is available, bound to a claim, or released by a claim.
	Phase PersistentVolumePhase `json:"phase,omitempty"`
}

// DaemonSetList is a collection of daemon sets.
type DaemonSetList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata,omitempty"`
	// Items is a list of daemon sets.
	Items []DaemonSet `json:"items"`
}

// DaemonSet represents the configuration of a daemon set.
type DaemonSet struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Status     DaemonSetStatus `json:"status,omitempty"`
}

// DaemonSetStatus represents the current status of a daemon set.
type DaemonSetStatus struct {
	// CurrentNumberScheduled is the number of nodes that are running at least 1
	// daemon pod and are supposed to run the daemon pod.
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`

	// NumberMisscheduled is the number of nodes that are running the daemon pod, but are
	// not supposed to run the daemon pod.
	NumberMisscheduled int32 `json:"numberMisscheduled"`

	// DesiredNumberScheduled is the total number of nodes that should be running the daemon
	// pod (including nodes correctly running the daemon pod).
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`

	// NumberReady is the number of nodes that should be running the daemon pod and have one
	// or more of the daemon pod running and ready.
	NumberReady int32 `json:"numberReady"`
}

// ReplicationController represents the configuration of a replication controller.
type ReplicationController struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`

	// Status is the current status of this replication controller. This data may be
	// out of date by some window of time.
	Status ReplicationControllerStatus
}

// ReplicationControllerStatus represents the current status of a replication
// controller.
type ReplicationControllerStatus struct {
	// Replicas is the number of actual replicas.
	Replicas int32
}

// ReplicaSet represents the configuration of a replica set.
type ReplicaSet struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`

	// Status is the current status of this ReplicaSet. This data may be
	// out of date by some window of time.
	Status ReplicaSetStatus
}

// ReplicaSetStatus represents the current status of a ReplicaSet.
type ReplicaSetStatus struct {
	// Replicas is the number of actual replicas.
	Replicas int32
}

// StatefulSet represents a set of pods with consistent identities.
type StatefulSet struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`

	Status StatefulSetStatus
}

// StatefulSetStatus represents the current status of a StatefulSet.
type StatefulSetStatus struct {
	// Replicas is the number of actual replicas.
	Replicas int32
}
