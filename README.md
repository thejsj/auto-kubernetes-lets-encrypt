# Auto Kuberentes Lets's Encrypt

_Generate TSL/SSL certs for your kubernetes ingress automatically with Let's Encrypt_

## How To Use

#### 1. Generate The Kubernetes Resources

```
./generate-resources $EMAIL $DOMAIN
```

#### 2. Apply To Cluster

```
kubectl apply -f ./kubernetes-resources.yml
```

#### 3. Update DNS

```
watch "kubectl get svc auto-kubernetes-lets-encrypt"
```

Take the IP address from the service and update your DNS settings.

#### 4. Wait for Job and Get Certificates

```
watch "kubectl get job auto-kubernetes-lets-encrypt"
```
