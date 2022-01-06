go mod init ls.com
rm -rf ./redis
mkdir redis
cd ./redis
kubebuilder init --domain ls.com
echo 'y\ny\n' | kubebuilder create api --group myapp --version v1 --kind Redis
#        modified:   PROJECT
#        modified:   go.mod
#        modified:   main.go
#        new file:   api/v1/groupversion_info.go
#        new file:   api/v1/redis_types.go
#        new file:   api/v1/zz_generated.deepcopy.go
#        new file:   config/crd/kustomization.yaml
#        new file:   config/crd/kustomizeconfig.yaml
#        new file:   config/crd/patches/cainjection_in_redis.yaml
#        new file:   config/crd/patches/webhook_in_redis.yaml
#        new file:   config/rbac/redis_editor_role.yaml
#        new file:   config/rbac/redis_viewer_role.yaml
#        new file:   config/samples/myapp_v1_redis.yaml
#        new file:   controllers/redis_controller.go
#        new file:   controllers/suite_test.go
kubebuilder create webhook --group myapp --version v1 --kind Redis --defaulting --programmatic-validation
# defaulting  修改
# programmatic-validation 验证
#        modified:   redis/PROJECT
#        new file:   redis/api/v1/redis_webhook.go
#        new file:   redis/api/v1/webhook_suite_test.go
#        modified:   redis/api/v1/zz_generated.deepcopy.go
#        new file:   redis/config/certmanager/certificate.yaml
#        new file:   redis/config/certmanager/kustomization.yaml
#        new file:   redis/config/certmanager/kustomizeconfig.yaml
#        new file:   redis/config/default/manager_webhook_patch.yaml
#        new file:   redis/config/default/webhookcainjection_patch.yaml
#        new file:   redis/config/webhook/kustomization.yaml
#        new file:   redis/config/webhook/kustomizeconfig.yaml
#        new file:   redis/config/webhook/service.yaml
#        modified:   redis/main.go
# github.com/jetstack/cert-manager 证书管理, 签发免费证书、自动续期
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.6.1/cert-manager.yaml
# 验证
kubectl get pods -n cert-manager

#make manifests
### controller-gen crd:trivialVersions=true object:headerFile=./hack/boilerplate.go.txt crd:crdVersions=v1 paths=./api/... output:crd:artifacts:config=config/crd/bases
###        new file:   config/crd/bases/myapp.ls.com_redis.yaml
#controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
#        new file:   config/crd/bases/myapp.ls.com_redis.yaml
#        new file:   config/rbac/role.yaml

#------test-----
# https://book.kubebuilder.io/reference/markers/crd-validation.html
make install
make run

make docker-build docker-push IMG=acejilam/xxx:latest
make deploy IMG=acejilam/xxx:latest

#docker pull exploitht/operator-static
#docker tag exploitht/operator-static gcr.io/distroless/static:nonroot
#docker rmi exploitht/operator-static
#docker pull kubesphere/kube-rbac-proxy:v0.8.0
#docker tag  kubesphere/kube-rbac-proxy:v0.8.0 gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0
#docker rmi  kubesphere/kube-rbac-proxy:v0.8.0

# kubectl patch configmap/mymap --type json --patch='[{"op":"remove","path":"/metadata/finalizers"}]'