package objectset

import (
	"time"

	"github.com/google/uuid"
	"github.com/rancher/wrangler/pkg/objectset"
	"github.com/rancher/wrangler/pkg/schemes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var (
	internalGroupVersion = schema.GroupVersion{Group: "internal.cattle.io", Version: "v1alpha1"}
)

// init adds the internal type to the default scheme for wrangler.apply to be able to use to add owner key details
func init() {
	schemes.Register(addInternalTypes)
}

// Adds the list of known types to Scheme.
func addInternalTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(internalGroupVersion, &objectSetState{}, &objectSetStateList{})
	metav1.AddToGroupVersion(scheme, internalGroupVersion)
	return nil
}

// newObjectSetState returns a new objectSetState for internal consumption
func newObjectSetState(namespace, name string, obj objectSetState) *objectSetState {
	obj.APIVersion, obj.Kind = internalGroupVersion.WithKind("objectSetState").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	// set values normally set by the server for an internal object
	obj.Generation = 0
	obj.UID = types.UID(uuid.New().String())
	obj.CreationTimestamp = metav1.NewTime(time.Now())
	obj.ResourceVersion = "0"
	return &obj
}

// objectSetState represents the state of an object set. This resource is only intended for
// internal use by this controller
type objectSetState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ObjectSet is a pointer to the underlying ObjectSet whose state is being tracked
	ObjectSet *objectset.ObjectSet `json:"objectSet,omitempty"`

	// Locked represents whether the ObjectSet should be locked in the cluster or not
	Locked bool `json:"locked"`
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *objectSetState) DeepCopyInto(out *objectSetState) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.ObjectSet = in.ObjectSet
	out.Locked = in.Locked
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new objectSetState.
func (in *objectSetState) DeepCopy() *objectSetState {
	if in == nil {
		return nil
	}
	out := new(objectSetState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is a deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *objectSetState) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// objectSetStateList represents a list of objectSetStates
type objectSetStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items are the objectSetStates tracked by this list
	Items []objectSetState `json:"items"`
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *objectSetStateList) DeepCopyInto(out *objectSetStateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]objectSetState, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new objectSetStateList.
func (in *objectSetStateList) DeepCopy() *objectSetStateList {
	if in == nil {
		return nil
	}
	out := new(objectSetStateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is a deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *objectSetStateList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
