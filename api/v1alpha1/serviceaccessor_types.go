/*


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

// ServiceAccessorSpec defines the desired state of ServiceAccessor
type ServiceAccessorSpec struct {
	AccessHosts []string `json:"accessHosts"`
}

// ServiceAccessorStatus defines the observed state of ServiceAccessor
type ServiceAccessorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// ServiceAccessor is the Schema for the serviceaccessors API
type ServiceAccessor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceAccessorSpec   `json:"spec,omitempty"`
	Status ServiceAccessorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// ServiceAccessorList contains a list of ServiceAccessor
type ServiceAccessorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccessor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceAccessor{}, &ServiceAccessorList{})
}
