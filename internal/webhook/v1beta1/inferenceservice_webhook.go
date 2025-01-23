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

package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// nolint:unused
// log is for logging in this package.
var inferenceservicelog = logf.Log.WithName("inferenceservice-resource")

// SetupInferenceServiceWebhookWithManager registers the webhook for InferenceService in the manager.
func SetupInferenceServiceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kservev1beta1.InferenceService{}).
		WithValidator(&InferenceServiceCustomValidator{}).
		WithDefaulter(&InferenceServiceCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-serving-kserve-io-v1beta1-inferenceservice,mutating=true,failurePolicy=fail,sideEffects=None,groups=serving.kserve.io,resources=inferenceservices,verbs=create;update,versions=v1beta1,name=minferenceservice.kb.io,admissionReviewVersions=v1

// InferenceServiceCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind InferenceService when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type InferenceServiceCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &InferenceServiceCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind InferenceService.
func (d *InferenceServiceCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	inferenceservice, ok := obj.(*kservev1beta1.InferenceService)
	if !ok {
		return fmt.Errorf("expected an InferenceService object but got %T", obj)
	}

	// 添加标记避免重复处理
	if inferenceservice.Annotations == nil {
		inferenceservice.Annotations = make(map[string]string)
	}
	if _, exists := inferenceservice.Annotations["codeupdater.icedcocon.github.io/defaulted"]; exists {
		return nil
	}

	inferenceservicelog.Info("Defaulting for InferenceService", "name", inferenceservice.GetName())

	if inferenceservice.Spec.Transformer != nil {
		// Define shared volume
		sharedVolume := corev1.Volume{
			Name: "shared-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}

		// Define volume mount for containers
		sharedMount := corev1.VolumeMount{
			Name:      "shared-data",
			MountPath: "/shared", // 修改挂载路径，避免冲突
		}

		// Add volume to pod spec if not exists
		hasSharedVolume := false
		for _, vol := range inferenceservice.Spec.Transformer.PodSpec.Volumes {
			if vol.Name == "shared-data" {
				hasSharedVolume = true
				break
			}
		}
		if !hasSharedVolume {
			inferenceservice.Spec.Transformer.PodSpec.Volumes = append(
				inferenceservice.Spec.Transformer.PodSpec.Volumes,
				sharedVolume,
			)
		}

		// Add mount to existing containers if not exists
		for i := range inferenceservice.Spec.Transformer.PodSpec.Containers {
			container := &inferenceservice.Spec.Transformer.PodSpec.Containers[i]
			hasSharedMount := false
			for _, mount := range container.VolumeMounts {
				if mount.Name == "shared-data" {
					hasSharedMount = true
					break
				}
			}
			if !hasSharedMount {
				container.VolumeMounts = append(container.VolumeMounts, sharedMount)
			}
		}

		// Check if code-updator container exists
		hasCodeUpdator := false
		for _, container := range inferenceservice.Spec.Transformer.PodSpec.Containers {
			if container.Name == "nginx-server" {
				hasCodeUpdator = true
				break
			}
		}

		if !hasCodeUpdator {
			// Add code-updator container
			container := corev1.Container{
				Name:  "nginx-server", // 改名为 nginx-server 更清晰
				Image: "nginx:alpine",
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: 8080,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				Command: []string{"nginx"},
				Args: []string{
					"-g", "daemon off;",
					"-c", "/etc/nginx/nginx.conf",
					"-p", "/tmp/",
					"-e", "/tmp/error.log",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "NGINX_PORT",
						Value: "8080",
					},
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
				},
				VolumeMounts: []corev1.VolumeMount{sharedMount},
			}

			// 不修改现有容器的配置
			inferenceservice.Spec.Transformer.PodSpec.Containers = append(
				inferenceservice.Spec.Transformer.PodSpec.Containers,
				container,
			)
		}
	}

	// 添加处理完成标记
	inferenceservice.Annotations["codeupdater.icedcocon.github.io/defaulted"] = "true"

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-serving-kserve-io-v1beta1-inferenceservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=serving.kserve.io,resources=inferenceservices,verbs=create;update,versions=v1beta1,name=vinferenceservice.kb.io,admissionReviewVersions=v1

// InferenceServiceCustomValidator struct is responsible for validating the InferenceService resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type InferenceServiceCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &InferenceServiceCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type InferenceService.
func (v *InferenceServiceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	inferenceservice, ok := obj.(*kservev1beta1.InferenceService)
	if !ok {
		return nil, fmt.Errorf("expected a InferenceService object but got %T", obj)
	}
	inferenceservicelog.Info("Validation for InferenceService upon creation", "name", inferenceservice.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type InferenceService.
func (v *InferenceServiceCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	inferenceservice, ok := newObj.(*kservev1beta1.InferenceService)
	if !ok {
		return nil, fmt.Errorf("expected a InferenceService object for the newObj but got %T", newObj)
	}
	inferenceservicelog.Info("Validation for InferenceService upon update", "name", inferenceservice.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type InferenceService.
func (v *InferenceServiceCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	inferenceservice, ok := obj.(*kservev1beta1.InferenceService)
	if !ok {
		return nil, fmt.Errorf("expected a InferenceService object but got %T", obj)
	}
	inferenceservicelog.Info("Validation for InferenceService upon deletion", "name", inferenceservice.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
