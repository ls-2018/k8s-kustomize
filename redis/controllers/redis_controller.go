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

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"ls.com/helper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	myappv1 "ls.com/api/v1"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	EventRecord record.EventRecorder
}

//+kubebuilder:rbac:groups=myapp.ls.com,resources=redis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=myapp.ls.com,resources=redis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=myapp.ls.com,resources=redis/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Redis object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	redis := &myappv1.Redis{}
	// 删除后会往这发一个请求,只有NamespacedName数据，别的都没有
	if err := r.Get(ctx, req.NamespacedName, redis); err != nil {
		return ctrl.Result{}, nil
	}
	// 正在删除
	if !redis.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.clearPods(context.Background(), redis)
	}
	// TODO pod 个数变动
	podNames := helper.GetRedisPodNames(redis)
	var err error
	if redis.Spec.Num > len(redis.Finalizers) {
		err = r.UpPods(ctx, podNames, redis)
		if err == nil {
			r.EventRecord.Event(redis, corev1.EventTypeNormal, "UpPods", fmt.Sprintf("%d", redis.Spec.Num))
		} else {
			r.EventRecord.Event(redis, corev1.EventTypeWarning, "DownPods", fmt.Sprintf("%d", redis.Spec.Num))
		}
	} else if redis.Spec.Num < len(redis.Finalizers) {
		err = r.DownPods(ctx, podNames, redis)
		if err == nil {
			r.EventRecord.Event(redis, corev1.EventTypeNormal, "DownPods", fmt.Sprintf("%d", redis.Spec.Num))
		} else {
			r.EventRecord.Event(redis, corev1.EventTypeWarning, "DownPods", fmt.Sprintf("%d", redis.Spec.Num))
		}
		redis.Status.RedisNum = len(redis.Finalizers)
	} else {
		for _, podName := range redis.Finalizers {
			if helper.IsExisted(podName, redis, r.Client) {
				continue
			} else {
				//	 重建此pod
				err = r.UpPods(ctx, []string{podName}, redis)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}
	r.Status().Update(ctx, redis)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myappv1.Redis{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, handler.Funcs{
			CreateFunc:  nil,
			UpdateFunc:  nil,
			DeleteFunc:  r.podDeleteHandler,
			GenericFunc: nil,
		}).
		Complete(r)
}

// 对于用户主动 删除的pod 需要重新创建
func (r *RedisReconciler) podDeleteHandler(event event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {
	fmt.Printf(`######################
%s
######################
`, event.Object.GetName())
	for _, ref := range event.Object.GetOwnerReferences() {
		if ref.Kind == r.kind() && ref.APIVersion == r.apiVersion() {
			//触发 Reconcile
			limitingInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: event.Object.GetNamespace(),
					Name:      ref.Name,
				},
			})
		}
	}
}

func (r *RedisReconciler) kind() string {
	return "Redis"
}

func (r *RedisReconciler) apiVersion() string {
	return "myapp.ls.com/v1"
}

func (r *RedisReconciler) clearPods(ctx context.Context, redis *myappv1.Redis) error {
	for _, podName := range redis.Finalizers {
		err := r.Client.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: redis.Namespace,
			},
		})
		//TODO 如果pod已经被删除、
		if err != nil {
			return err
		}
	}
	redis.Finalizers = []string{}
	return r.Client.Update(ctx, redis)
}
func (r *RedisReconciler) DownPods(ctx context.Context, podNames []string, redis *myappv1.Redis) error {
	for i := len(redis.Finalizers) - 1; i >= len(podNames); i-- {
		if !helper.IsExisted(redis.Finalizers[i], redis, r.Client) {
			continue
		}
		err := r.Client.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      redis.Finalizers[i],
				Namespace: redis.Namespace,
			},
		})
		if err != nil {
			return err
		}
	}
	redis.Finalizers = append(redis.Finalizers[:0], redis.Finalizers[:len(podNames)]...)
	return r.Client.Update(ctx, redis)
}
func (r *RedisReconciler) UpPods(ctx context.Context, podNames []string, redis *myappv1.Redis) error {
	for _, podName := range podNames {
		podName, err := helper.CreateRedis(r.Client, redis, podName, r.Scheme)
		if err != nil {
			return err
		}
		if controllerutil.ContainsFinalizer(redis, podName) {
			continue
		}
		redis.Finalizers = append(redis.Finalizers, podName)
	}
	err := r.Client.Update(ctx, redis)
	return err
}
