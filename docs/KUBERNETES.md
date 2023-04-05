# Kubernetes

```console
$ kubectl apply -f test/full-setup/kubernetes/k8s.yml
service/go-proxy-cache created
deployment.apps/go-proxy-cache created
service/nginx created
deployment.apps/nginx created
service/node created
deployment.apps/node created
service/redis created
deployment.apps/redis created

$ kubectl get services
NAME             TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                                                           AGE
go-proxy-cache   LoadBalancer   10.96.255.92     localhost     50080:30836/TCP,50443:32728/TCP                                   1s
kubernetes       ClusterIP      10.96.0.1        <none>        443/TCP                                                           1s
nginx            LoadBalancer   10.108.101.190   localhost     40080:30935/TCP,40081:32703/TCP,40082:31082/TCP,40443:31329/TCP   1s
node             ClusterIP      10.105.106.132   <none>        9001/TCP,9002/TCP                                                 1s
redis            LoadBalancer   10.98.252.182    localhost     6379:32000/TCP                                                    1s

$ kubectl get deployments
NAME             READY   UP-TO-DATE   AVAILABLE   AGE
go-proxy-cache   1/1     1            1           1s
nginx            1/1     1            1           1s
node             1/1     1            1           1s
redis            1/1     1            1           1s

$ kubectl get pods
NAME                              READY   STATUS    RESTARTS   AGE
go-proxy-cache-76cccc45db-jgphw   1/1     Running   0          1s
nginx-77df469c6f-4dnnx            1/1     Running   0          1s
node-8659d8958f-gtpkh             1/1     Running   0          1s
redis-b46545bbd-65tn9             1/1     Running   0          1s

$ export NGINX_HOST_80=localhost:40080
$ export NGINX_HOST_443=localhost:40443
$ export REDIS_HOSTS=localhost:6379
$ make test
```
