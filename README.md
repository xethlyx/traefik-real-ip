# Traefik Real IP

If Traefik is behind a load balancer, it won't be able to get the Real IP from the external client by checking the remote IP address. Traefik can be configured to accept `X-Real-Ip` and `X-Forwarded-For`, but `X-Real-Ip` is only set to the IP of the connecting proxy. This bridges the gap and passes the data downwards to applications behind the proxy.

Evaluation of `X-Forwarded-For` IPs will go from the last to first, breaking on the first untrusted IP. This also removes all untrusted IPs from `X-Forwarded-For`.

#
## Configuration

### Static

```yaml
pilot:
  token: xxxx

experimental:
  plugins:
    traefik-real-ip:
      modulename: github.com/xethlyx/traefik-real-ip
      version: v1.0.4
```

### Dynamic configuration

```yaml
http:
  routers:
    my-router:
      rule: Path(`/whoami`)
      service: service-whoami
      entryPoints:
        - http
      middlewares:
        - traefik-real-ip

  services:
   service-whoami:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000

  middlewares:
    traefik-real-ip:
      plugin:
        traefik-real-ip:
          trustedIPs:
            - "1.1.1.1/24"
```

### Kubernetes configuration

```yaml
kind: Deployment
apiVersion: apps/v1
metadata:
  namespace: default
  name: traefik
  labels:
    app: traefik
spec:
  replicas: 1
  selector:
    matchLabels:
      app: traefik
  template:
    metadata:
      labels:
        app: traefik
    spec:
      terminationGracePeriodSeconds: 60
      serviceAccountName: traefik-ingress-controller
      containers:
        - name: traefik
          image: traefik:v2.4
          args:
            - --api.insecure
            - --accesslog
            - --entrypoints.web.Address=:80
            - --providers.kubernetescrd
            - --pilot.token={YOUR_PILOT_TOKEN}
            - --experimental.plugins.traefik-real-ip.modulename=github.com/xethlyx/traefik-real-ip
            - --experimental.plugins.traefik-real-ip.version=v1.0.4
          ports:
            - name: web
              containerPort: 80
            - name: admin
              containerPort: 8080
          resources:
            requests:
              cpu: 300m
            limits:
              cpu: 500m

---
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: traefik-real-ip
spec:
  plugin:
    traefik-real-ip:
      trustedIPs:
        - "1.1.1.1/24"

---
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: ingress-example
  namespace: default
spec:
  entryPoints:
    - web
  routes:
    - kind: Rule
      match: Host(`domain.ltd`) && PathPrefix(`/`)
      services:
        - name: example-service
          port: 80
      middlewares:
        - name: traefik-real-ip
```

#
## Configuration documentation

Supported configurations per body

| Setting           | Allowed values      | Required    | Description |
| :--               | :--                 | :--         | :--         |
| trustedIPs        | []string            | No          | CIDR IP ranges to trust (use `/32` for a single value) |

#