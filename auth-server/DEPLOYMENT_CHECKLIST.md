# Deployment Checklist

## POC Deployment (Kind Clusters)

### Prerequisites
- [ ] Docker installed and running
- [ ] Kind installed (`kind version`)
- [ ] kubectl installed (`kubectl version`)
- [ ] Helm 3 installed (`helm version`)
- [ ] Go 1.24+ installed (`go version`)

### Hub Cluster Setup
- [ ] Kind hub cluster created
- [ ] cert-manager installed and ready
- [ ] rbac-apiserver image built
- [ ] rbac-apiserver image loaded to kind
- [ ] rbac-apiserver deployed via Helm
- [ ] rbac-apiserver pod running
- [ ] APIService available (`kubectl get apiservices v1alpha1.authorization.open-cluster-management.io`)

### Managed Cluster Setup
- [ ] Kind managed cluster created
- [ ] auth-server image built
- [ ] auth-server image loaded to kind
- [ ] auth-server namespace created
- [ ] Hub kubeconfig secret created
- [ ] TLS certificates secret created
- [ ] auth-server deployment created
- [ ] auth-server pods running (2 replicas)
- [ ] auth-server service created

### Webhook Configuration
- [ ] Webhook config file created at `/etc/kubernetes/authz-webhook/config.yaml`
- [ ] Webhook config copied to managed cluster node
- [ ] kube-apiserver flags include `--authorization-webhook-config-file`
- [ ] kube-apiserver restarted
- [ ] kube-apiserver shows webhook in process args

### Verification
- [ ] Hub rbac-apiserver responding to API calls
- [ ] Managed auth-server health check passing
- [ ] Auth-server can connect to hub
- [ ] Webhook is being called (check logs)
- [ ] PermissionRequest objects created and cleaned up

### Testing
- [ ] PermissionBinding created on hub
- [ ] `kubectl auth can-i` commands work
- [ ] Auth-server logs show authorization requests
- [ ] Correct allow/deny decisions

---

## Production Deployment

### Prerequisites
- [ ] Kubernetes 1.24+ cluster (hub)
- [ ] Kubernetes 1.24+ cluster (managed)
- [ ] Container registry access
- [ ] TLS certificates (production-grade)
- [ ] Network connectivity between clusters

### Hub Cluster (rbac-apiserver)
- [ ] Production cert-manager OR custom certificates
- [ ] rbac-apiserver Helm chart configured
- [ ] Resource limits appropriately sized
- [ ] Persistent storage for SpiceDB (if required)
- [ ] RBAC permissions configured
- [ ] Service account created
- [ ] Network policies configured
- [ ] Monitoring and alerting configured
- [ ] Backup strategy in place

### Managed Cluster (auth-server)
- [ ] auth-server image pushed to registry
- [ ] Namespace created
- [ ] ServiceAccount created
- [ ] Hub kubeconfig secret created (long-lived token or cert)
- [ ] TLS certificates secret created (signed by trusted CA)
- [ ] Deployment manifest applied
- [ ] Service created
- [ ] Pod Security Policies/Standards configured
- [ ] Resource requests/limits tuned for workload
- [ ] High availability (multiple replicas)
- [ ] Pod disruption budgets configured

### Security
- [ ] TLS certificates from trusted CA
- [ ] Service accounts with minimal permissions
- [ ] Network policies restrict traffic
- [ ] Secrets encryption at rest enabled
- [ ] Image pull secrets configured
- [ ] Images scanned for vulnerabilities
- [ ] Pod security policies enforced
- [ ] Audit logging enabled

### Networking
- [ ] Hub rbac-apiserver accessible from managed cluster
- [ ] TLS configured and verified
- [ ] Firewall rules allow traffic
- [ ] DNS resolution working
- [ ] Network latency acceptable (<100ms recommended)
- [ ] Load balancer configured (if needed)

### Authorization Webhook
- [ ] Webhook config file created
- [ ] Webhook config deployed to all control plane nodes
- [ ] kube-apiserver configuration updated
- [ ] kube-apiserver restarted on all control plane nodes
- [ ] Fallback authorization mode configured (RBAC)
- [ ] Cache TTL configured appropriately
- [ ] Test mode verified before production

### Monitoring and Observability
- [ ] Prometheus metrics exposed
- [ ] Grafana dashboards created
- [ ] Log aggregation configured
- [ ] Alerts configured for:
  - [ ] Auth-server pod restarts
  - [ ] High authorization latency
  - [ ] Hub connectivity failures
  - [ ] High error rates
  - [ ] Certificate expiration
- [ ] Distributed tracing (optional)

### Testing
- [ ] Smoke tests pass
- [ ] Load testing completed
- [ ] Failover testing completed
- [ ] Certificate rotation tested
- [ ] Hub unavailability handling tested
- [ ] Authorization caching verified
- [ ] Multi-user scenarios tested

### Documentation
- [ ] Architecture diagrams updated
- [ ] Runbook created
- [ ] Incident response procedures documented
- [ ] Monitoring guide created
- [ ] Troubleshooting guide created

### Rollout Strategy
- [ ] Rollout plan created
- [ ] Rollback plan documented
- [ ] Gradual rollout to managed clusters
- [ ] Canary deployment tested
- [ ] Communication plan to users
- [ ] Maintenance window scheduled

---

## Post-Deployment

### Immediate (First Hour)
- [ ] All pods running and healthy
- [ ] No crash loops
- [ ] Logs show successful startups
- [ ] Authorization requests being processed
- [ ] No error spikes in logs
- [ ] Latency within acceptable range

### Short Term (First Day)
- [ ] Monitor error rates
- [ ] Check resource usage
- [ ] Verify cache hit rates
- [ ] Review authorization decisions
- [ ] Check for any denied requests
- [ ] Verify PermissionRequest cleanup

### Medium Term (First Week)
- [ ] Performance metrics stable
- [ ] No memory leaks
- [ ] Certificate expiration dates noted
- [ ] User feedback gathered
- [ ] Fine-tune cache TTL if needed
- [ ] Adjust resource limits if needed

### Long Term (Ongoing)
- [ ] Regular security updates
- [ ] Certificate rotation
- [ ] Performance optimization
- [ ] Feature enhancements
- [ ] User training
- [ ] Documentation updates

---

## Troubleshooting Checklist

### Auth-Server Not Starting
- [ ] Check pod logs
- [ ] Verify hub-kubeconfig secret exists
- [ ] Verify TLS certs secret exists
- [ ] Check resource limits
- [ ] Verify image pull successful

### Authorization Not Working
- [ ] Verify webhook config on apiserver
- [ ] Check auth-server logs for requests
- [ ] Verify hub rbac-apiserver is accessible
- [ ] Check PermissionBindings exist on hub
- [ ] Verify network connectivity
- [ ] Check TLS certificates valid

### High Latency
- [ ] Check network latency to hub
- [ ] Review auth-server resource usage
- [ ] Check hub rbac-apiserver performance
- [ ] Verify SpiceDB performance
- [ ] Consider caching optimization

### Hub Connectivity Issues
- [ ] Verify network connectivity
- [ ] Check hub-kubeconfig validity
- [ ] Verify hub rbac-apiserver running
- [ ] Check firewall rules
- [ ] Verify DNS resolution

---

## Rollback Procedure

### Emergency Rollback
1. [ ] Disable authorization webhook on kube-apiserver
2. [ ] Edit kube-apiserver manifest to remove webhook flags
3. [ ] Wait for kube-apiserver restart
4. [ ] Verify cluster operations normal
5. [ ] Investigate issue
6. [ ] Plan remediation

### Planned Rollback
1. [ ] Communicate to users
2. [ ] Disable webhook on managed clusters
3. [ ] Scale down auth-server
4. [ ] Remove hub rbac-apiserver (if needed)
5. [ ] Clean up resources
6. [ ] Document lessons learned

---

## Success Criteria

### POC Success
- [ ] Authorization webhook functional
- [ ] Permissions evaluated correctly
- [ ] Logs show expected behavior
- [ ] No critical errors
- [ ] Demo-able to stakeholders

### Production Success
- [ ] 99.9% uptime
- [ ] <100ms authorization latency (p95)
- [ ] <1% error rate
- [ ] Zero security incidents
- [ ] Positive user feedback
- [ ] Successful load testing

---

## Contact and Escalation

### Support Contacts
- **Primary**: [Your Team]
- **Secondary**: [Backup Team]
- **Emergency**: [On-call]

### Escalation Path
1. Team lead
2. Platform team
3. Engineering manager
4. On-call engineer

---

## Revision History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-10-27 | 1.0 | Initial checklist | Auto-generated |

