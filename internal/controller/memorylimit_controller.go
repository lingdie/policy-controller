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

package controller

import (
	"context"
	"math"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	policyv1 "github.com/lingdie/policy-controller/api/v1"
)

// MemoryLimitReconciler reconciles a MemoryLimit object
type MemoryLimitReconciler struct {
	client.Client
	MetricsClient *metricsv.Clientset

	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=policy.sealos.io,resources=memorylimits,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy.sealos.io,resources=memorylimits/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=policy.sealos.io,resources=memorylimits/finalizers,verbs=update

func (r *MemoryLimitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// get the MemoryLimit policy
	policy := &policyv1.MemoryLimit{}
	err := r.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// get all namespaces that match the selector
	selector, err := metav1.LabelSelectorAsSelector(&policy.Spec.NamespaceSelector)
	if err != nil {
		logger.Error(err, "unable to create selector from namespace selector")
		return ctrl.Result{}, err
	}
	namespaces := &v1.NamespaceList{}
	err = r.List(ctx, namespaces, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return ctrl.Result{}, err
	}

	// get all pods in the namespaces
	for _, namespace := range namespaces.Items {
		logger.Info("Checking namespace", "name", namespace.Name)
		pods := &v1.PodList{}
		err = r.List(ctx, pods, &client.ListOptions{Namespace: namespace.Name})
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != v1.PodRunning {
				continue
			}
			// get the memory usage for the container
			metrics, err := r.MetricsClient.MetricsV1beta1().PodMetricses(namespace.Name).Get(ctx, pod.Name, metav1.GetOptions{})
			if err != nil {
				logger.Error(err, "unable to get pod metrics")
				continue
			}
			isExceeded := false
			for _, container := range pod.Spec.Containers {
				if container.Resources.Limits == nil || container.Resources.Limits.Memory() == nil || container.Resources.Limits.Memory().IsZero() {
					continue
				}
				for _, containerMetrics := range metrics.Containers {
					if containerMetrics.Name != container.Name {
						continue
					}
					// compare the memory usage to the limit
					podLimit := float64(container.Resources.Limits.Memory().Value())
					policyLimitPercent := float64(policy.Spec.LimitPercentage) * 0.01
					policyLimit := int64(math.Ceil(podLimit * policyLimitPercent))
					if containerMetrics.Usage.Memory().Cmp(*resource.NewQuantity(policyLimit, resource.BinarySI)) > 0 {
						logger.Info("memory limit exceeded", "pod", pod.Name, "namespace", namespace.Name, "container", container.Name)
						isExceeded = true
						break
					}
				}
			}

			if isExceeded {
				if HasAnnotation(pod.ObjectMeta, "policy.sealos.io/memory-limit-exceeded") {
					// if pod has annotation, delete this pod
					logger.Info("Deleting pod", "pod", pod.Name, "namespace", namespace.Name)
					err = r.Delete(ctx, &pod)
					if err != nil {
						logger.Error(err, "unable to delete pod")
					}
				} else {
					logger.Info("Adding annotation", "pod", pod.Name, "namespace", namespace.Name)
					SetMetaDataAnnotation(&pod.ObjectMeta, "policy.sealos.io/memory-limit-exceeded", "true")
					err = r.Update(ctx, &pod)
					if err != nil {
						logger.Error(err, "unable to update pod")
						break
					}
				}
			} else {
				logger.Info("Passed policy", "pod", pod.Name, "namespace", namespace.Name)
			}
		}
	}
	logger.Info("reconcile complete", "requeue in", policy.Spec.RequeueTime.Duration)
	return ctrl.Result{RequeueAfter: policy.Spec.RequeueTime.Duration}, nil
}

func SetMetaDataAnnotation(m *metav1.ObjectMeta, s string, s2 string) {
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	m.Annotations[s] = s2
}

func HasAnnotation(meta metav1.ObjectMeta, s string) bool {
	_, found := meta.Annotations[s]
	return found
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemoryLimitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1.MemoryLimit{}).
		Complete(r)
}
