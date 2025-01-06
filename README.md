# tiny-scheduler

a simple and tiny scheduler for learning scheduling algorithm.

## How to start
first you need a k8s cluster, with its kubeconfig placed at ~/.kube/config. I use minikube for testing.
```bash
go run tiny-scheduler.go -alsologtostderr=true
```
## How to test

```bash
kubectl apply -f test/tiny-pod.yaml
```
then you will see something like this:

```bash
I0106 23:07:24.375661   71650 scheduler.go:105] Scheduled podInfo tiny-pod to node minikube
```

