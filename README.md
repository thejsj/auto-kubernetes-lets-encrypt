# Auto Kuberentes Lets's Encrypt

_Generate TSL/SSL certs through your kubernetes cluter automatically with Let's Encrypt_

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
kubectl describe secret auto-kubernetes-lets-encrypt
```

**You're Done!**

You will now find the certificates for the domain in your `auto-kubernetes-lets-encrypt` secret:

```
$ kubectl describe secret auto-kubernetes-lets-encrypt
Name:		auto-kubernetes-lets-encrypt
Namespace:	default
Labels:		<none>
Annotations:
Type:		Opaque

Data
====
$DOMAIN.issuer.crt:	1647 bytes
$DOMAIN.json:		228 bytes
$DOMAIN.key:		1679 bytes
$DOMAIN.pem:		3480 bytes
private_key:			1675 bytes
registration:			668 bytes
$DOMAIN.crt:		1801 bytes
```
