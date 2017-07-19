# Auto Kuberentes Lets's Encrypt

_Generate TSL/SSL certs for your kubernetes ingress automatically with Let's Encrypt_

## How To Use

#### 1. Generate The Kubernetes Resources

```
./generate-resources $DOMAIN $EMAIL
```

#### 2. Apply Part 1 To Cluster

```
kubectl apply -f ./kubernetes-resources-part-1.yml
```

#### 3. Update DNS

```
watch "kubectl get svc auto-kubernetes-lets-encrypt"
```

Take the IP address from the service and update your DNS settings.

#### 4. Apply Part 2 and Wait for Job and Get Certificates

```
kubectl apply -f ./kubernetes-resources-part-1.yml
watch "kubectl get job auto-kubernetes-lets-encrypt"
kubectl get secret auto-kubernetes-lets-encrypt
```
