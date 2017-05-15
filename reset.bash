git add -A
git commit -m "$1"
docker build -t quay.io/hiphipjorge/auto-kubernetes-lets-encrypt:$1 server/.
docker push  quay.io/hiphipjorge/auto-kubernetes-lets-encrypt:$1
sed -i "s/v0.0.[[:digit:]]*/$1/g" kubernetes-resources.yml
git add -A
git commit -m "Update kubernetes resources $1"
kubectl --namespace=auto-kubernetes-lets-encrypt delete job auto-kubernetes-lets-encrypt
kubectl --namespace=auto-kubernetes-lets-encrypt apply -f kubernetes-resources.yml
sleep 5
POD=$(kubectl --namespace=auto-kubernetes-lets-encrypt get pods | grep auto-kubernetes-lets-encrypt | awk '{print $1}')
echo $POD
kubectl --namespace=auto-kubernetes-lets-encrypt logs -f $POD
