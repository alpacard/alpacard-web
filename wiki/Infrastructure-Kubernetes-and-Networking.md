# ☸️ Infrastructure, Kubernetes & Networking

Alpacard runs on a production-grade, highly optimized **Dedicated Kubernetes Cluster** with software-defined networking, automated TLS certificate provisioning, and isolated zero-trust VPN staging access.

---

## 🏗️ Cluster Topology

```
+-----------------------------------------------------------------------------------+
|  DEDICATED KUBERNETES INFRASTRUCTURE NODE (Ubuntu Linux Cluster Node)             |
|                                                                                   |
|  +-----------------------------------------------------------------------------+  |
|  |  Kubernetes Control Plane & Worker Node Runtime                             |  |
|  |                                                                             |  |
|  |  +-----------------------------------------------------------------------+  |  |
|  |  |  MetalLB Layer 2 Load Balancer -> Virtual IP: 192.168.0.50             |  |  |
|  |  +-----------------------------------------------------------------------+  |  |
|  |                                      |                                      |  |
|  |                                      v                                      |  |
|  |  +-----------------------------------------------------------------------+  |  |
|  |  |  Traefik Ingress Controller (Ports 80 & 443 TLS Termination)           |  |  |
|  |  +-----------------------------------------------------------------------+  |  |
|  |                    |                                   |                    |  |
|  |                    v (Host: alpacard.app)               v (Host: dev...)     |  |
|  |  +------------------------------------+  +-------------------------------+  |  |
|  |  |  Namespace: alpacard (PROD)        |  |  Namespace: alpacard-dev (DEV)|  |  |
|  |  |  • alpacard-web (Nginx/React)      |  |  • alpacard-web (Nginx/React) |  |  |
|  |  |  • alpacard-wallet (PWA)           |  |  • alpacard-wallet (PWA)      |  |  |
|  |  |  • alpacard-api (Go Server)        |  |  • alpacard-api (Go Server)   |  |  |
|  |  |  • alpacard-worker (Go Consumer)   |  |  • alpacard-worker (Consumer) |  |  |
|  |  |  • PostgreSQL 16 (StatefulSet)     |  |  • PostgreSQL 16 (Isolated)   |  |  |
|  |  |  • RabbitMQ 3.13 (Message Broker)  |  |  • RabbitMQ 3.13 (Isolated)   |  |  |
|  |  +------------------------------------+  +-------------------------------+  |  |
|  +-----------------------------------------------------------------------------+  |
+-----------------------------------------------------------------------------------+
```

---

## 🔒 Multi-Environment Isolation

| Layer | Production (`alpacard`) | Development (`alpacard-dev`) |
| :--- | :--- | :--- |
| **Domain** | [https://alpacard.app](https://alpacard.app) | [https://dev.alpacard.app](https://dev.alpacard.app) |
| **Network Reachability** | Public Internet (Proxied by Cloudflare Edge CDN) | **Private Only**: Routed via Tailscale VPN subnet (`192.168.0.0/24`) |
| **Kubernetes Namespace** | `alpacard` | `alpacard-dev` |
| **Database Instance** | Dedicated PostgreSQL 16 with automated daily backups | Fully isolated PostgreSQL instance (`alpacard_dev`) |
| **Message Broker** | Production RabbitMQ cluster | Isolated RabbitMQ instance |
| **TLS Certificates** | Automated Let's Encrypt multi-SAN SSL | Automated Let's Encrypt TLS via Cloudflare DNS-01 |

---

## 📜 Automated TLS & Cert-Manager Architecture

TLS certificates are issued and renewed automatically using `cert-manager` and the **Cloudflare DNS-01 challenge solver**:

### ClusterIssuer Configuration (`letsencrypt-prod`)
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    email: Cavallo.v.gabriel@gmail.com
    server: https://acme-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: letsencrypt-prod-account-key
    # Anchors certificate issuance directly to universal ISRG Root X1 for 100% iOS/Android compatibility
    preferredChain: "ISRG Root X1"
    solvers:
    - dns01:
        cloudflare:
          apiTokenSecretRef:
            key: api-token
            name: cloudflare-api-token-secret
```

* **Why DNS-01?**: Because `dev.alpacard.app` resolves to an RFC 1918 private IP address (`192.168.0.50`), HTTP-01 challenges are impossible. The DNS-01 solver creates temporary TXT records in Cloudflare DNS, allowing Let's Encrypt to validate domain ownership even for private intranet environments.
* **Why `preferredChain: "ISRG Root X1"`?**: Bypasses intermediate `Root YR` hierarchies that can trigger `NET::ERR_CERT_AUTHORITY_INVALID` warnings on older mobile trust stores, providing instant zero-warning SSL validation on mobile Chrome and iOS Safari.

---

## 🛡️ Zero-Trust Staging VPN via Tailscale

To allow QA testers, developers, and stakeholders to test the non-production environment securely from anywhere in the world on mobile data (4G/5G) or remote networks without exposing internal staging ports to the public internet:

1. **Enterprise Subnet Gateway**:
   - A dedicated secure subnet router node advertises the internal staging subnet:
     ```bash
     tailscale up --advertise-routes=192.168.0.0/24 --accept-routes
     ```
2. **Accessing the Dev Environment**:
   - Authorized team members connect through the enterprise Tailscale VPN mesh.
   - When connected, traffic to the internal staging subnet (`192.168.0.0/24`) routes securely through the private gateway.
   - Opening `https://dev.alpacard.app` connects directly to the internal Kubernetes ingress load balancer (`192.168.0.50`), presenting the isolated staging environment with valid Let's Encrypt TLS.
3. **Database Reachability & QA Automation**:
   - The PostgreSQL databases are exposed through dedicated MetalLB Layer 2 Virtual IPs and NodePorts on the internal subnet (`192.168.0.0/24`), reachable by developers and QA automation tools directly when connected to Tailscale VPN or on the local network:
     - **Development DB (`alpacard-dev`)**:
       - Host: `dev-db.alpacard.app` (or direct IP `192.168.0.54`)
       - NodePort Fallback: `192.168.0.103:30542`
       - Port: `5432` | DB: `alpacard_dev` | User: `alpacard_dev` | Password: `alpacard_dev_secret_2026`
     - **Production DB (`alpacard`)**:
       - Host: `prod-db.alpacard.app` (or direct IP `192.168.0.55`)
       - NodePort Fallback: `192.168.0.103:30541`
       - Port: `5432` | DB: `alpacard` | User: `alpacard`
   - **QA Automated Testing Suite**:
     - `pnpm test:qa:all` (executes Newman API test journeys followed by relational database integrity checks)
     - `pnpm test:qa:db` (direct relational schema, referential integrity, and anti-replay audit verification)
