package helper

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	v1 "ls.com/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateRedis(client client.Client, redisConfig *v1.Redis, podName string, schema *runtime.Scheme) (string, error) {
	if IsExisted(podName, redisConfig, client) {
		return podName, nil
	}
	newPod := &corev1.Pod{}
	newPod.Name = podName
	newPod.Namespace = redisConfig.Namespace
	newPod.Spec.Containers = append(newPod.Spec.Containers, corev1.Container{
		Name:  "redis",
		Image: "redis",
		Ports: []corev1.ContainerPort{
			{
				Name:          redisConfig.Name,
				ContainerPort: int32(redisConfig.Spec.Port),
			},
		},
	})
	err := controllerutil.SetControllerReference(redisConfig, newPod, schema)
	if err != nil {
		return "", err
	}

	return podName, client.Create(context.Background(), newPod)
}
func GetRedisPodNames(redisConfig *v1.Redis) []string {
	names := make([]string, 0, redisConfig.Spec.Num)
	for i := 0; i < redisConfig.Spec.Num; i++ {
		names = append(names, fmt.Sprintf("%s-%d", redisConfig.Name, i))
	}
	return names
}
func IsExisted(podName string, redis *v1.Redis, client client.Client) bool {
	err := client.Get(context.Background(), types.NamespacedName{
		Namespace: redis.Namespace,
		Name:      podName,
	}, &corev1.Pod{})
	if err != nil {
		return false
	}
	return true
}
