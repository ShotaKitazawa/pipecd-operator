# pipecd-operator

Kubernetes Operator of https://github.com/pipe-cd/pipe

---

### checked environment

* Kubernetes v1.19.1 (kind cluster)
* PipeCD v0.9.0

---

### Install

* Install CRD & Custom Controller

```
kubectl apply -f https://raw.githubusercontent.com/ShotaKitazawa/pipecd-operator/master/deploy/deploy.yaml
```

### Sample Usage

* Run PipeCD (`ControlPlane` & `Piped`)
    * `Project`, `Environment`, and `Piped` already set up

```
kubectl apply -f https://raw.githubusercontent.com/ShotaKitazawa/pipecd-operator/master/config/samples/secret.yaml
kubectl apply -f https://raw.githubusercontent.com/ShotaKitazawa/pipecd-operator/master/config/samples/pipecd_v1alpha1_minio.yaml
kubectl apply -f https://raw.githubusercontent.com/ShotaKitazawa/pipecd-operator/master/config/samples/pipecd_v1alpha1_mongo.yaml
kubectl apply -f https://raw.githubusercontent.com/ShotaKitazawa/pipecd-operator/master/config/samples/pipecd_v1alpha1_controlplane.yaml
kubectl apply -f https://raw.githubusercontent.com/ShotaKitazawa/pipecd-operator/master/config/samples/pipecd_v1alpha1_environment.yaml
kubectl apply -f https://raw.githubusercontent.com/ShotaKitazawa/pipecd-operator/master/config/samples/pipecd_v1alpha1_piped.yaml
```

* using kubectl port-forward to expose the installed control-plane & point your web browser to http://localhost:8080

```
kubectl -n pipecd port-forward svc/pipecd 8080
```
