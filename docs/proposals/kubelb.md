# KubeLB

**Author**: Matthias Osthues

**Status**: Draft proposal; prototype in progress.

## Goals

KubeLB is a provider for multi cluster load balancing and takes the advantages of Kubernetes to do the load balancing. 

## Non-Goals

* Implementation of service type LoadBalancer like [MetalLB](https://metallb.universe.tf/)

* ECMP and BGP?

## Motivation and Background

Kubernetes does not offer an out of the box implementation of load balancers for clusters. The implementations of Network LB & Ingress LB that Kubernetes does ship with are all calls out to various IaaS platforms (GCP, AWS, Azure…). If you’re not running on a supported IaaS platform (GCP, AWS, Azure…), LoadBalancers & Ingress will not get provisioned.
Solutions which are available e.g. MetalLB focus on a single cluster. KubeLB aims to provide load balancing instances for multiple clusters and takes the advantages of kubernetes itself. 

 
#### Possible features: 

* Implementation of service type LoadBalancer where it's not available

* Possible cost reduction as you can decide between KubeLB and IaaS LoadBalancer instances

* Multi cluster load balancing

* Internal/External load balancing
 
* In a multi cluster environment it can be useful to have a single entrypoint inside the network to do some security or monitoring

* ...

## Implementation

The overall implementation contains three different parts: 

**Agent**: Controller which is deployed in every user cluster. It watches for Services and creates a CRD inside the load balancer cluster.

**Manager**: Controller which is responsible for deploying and configuring the actual resources inside the load balancer cluster.

**Controller**: Deploys the Manager inside the load balancer cluster for each user cluster. The Manager deployment is probably namespaced.

Requires a LoadBalancer implementation for the LB Cluster.

#### Implementation L4

The agent watches for Services of type LoadBalancer/NodePort in the user cluster. In the user cluster itself for each service with type LoadBalancer a NodePort is allocated by default. 
The agent informs the Manager which creates a Service and Endpoint in the LB cluster and adds the node IP addresses of the user cluster to the Endpoint IP addresses there. The agent watches for node changes like "remove", "add" and failures and will update the IP list in the Endpoint accordingly.
Evaluation is needed for failing node detection, so it is fast enough and meet our requirements. If not we need to do some active health checks.
 
For IaaS type load balancers: The controller will use the provisioned load balancers endpoint as its own endpoint.
For non implemented type load balancers: The controller will update the Status and IP of the Service in the User cluster, when the LB is provisioned or changed.

Kubernetes will forward the traffic from the Service on the LB cluster to one of the endpoint IPs and the port.

Example Configuration LB Cluster:

    kind: Service
    apiVersion: v1
    metadata:
      name: hello-svc
    spec:
      type: LoadBalancer
      ports:
      - protocol: TCP
        port: 80
        targetPort: 80
        name: http
    ---
    apiVersion: v1
    kind: Endpoints
    metadata:
      name: hello-svc
    subsets:
    - addresses:
      - ip: 10.0.0.1 #Node-1 address 
      - ip: 10.0.0.2 #Node-2 address
      ports:
      - name: http
        port: 80
        protocol: TCP

#### Implementation L7

Follows the same concept as the L4 implementation plus an Ingress resource in the user cluster. The agent will watch for the Ingress resource and inform the manager in the LB Cluster.
The manager will create the Service as described in the L4 Implementation and a HTTPProxy resource. Which takes the advantage of contour to manage different domains and take care of envoy's configuration.


On the User cluster:

    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: hello-svc
      annotations:
        kubelb.expose: extern / intern
    spec:
      rules:
      - http:
          paths:
          - path: /testpath
            backend:
              serviceName: test
              servicePort: 80

For the LB cluster:

    apiVersion: projectcontour.io/v1
    kind: HTTPProxy
    metadata:
      name: hello-svc
      namespace: clustername
    spec:
      virtualhost:
        fqdn: INGRESS.CLUSTERNAME.lb.example.net
      routes:
        - conditions:
          - prefix: /
          services:
            - name: hello-svc
              port: 80
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: hello-svc
      namespace: clustername
    spec:
      type: LoadBalancer
      ports:
      - protocol: TCP
        port: 80
        targetPort: 80
        name: http
    ---
    apiVersion: v1
    kind: Endpoints
    metadata:
      name: hello-svc
      namespace: clustername
    subsets:
    - addresses:
      - ip: 10.0.0.1 #Node-1 address 
      - ip: 10.0.0.2 #Node-2 address
      ports:
      - name: http
        port: 80
        protocol: TCP

#### Envoy & Contour

On the LB cluster Envoy & Contour are installed. When the controller creates a new HTTPProxy, Contour will configure Envoy automatically to process the traffic of the domain.
The LB cluster will have a domain assigned e.g. lb.example.com each cluster will have a dedicated subdomain CLUSTERNAME.lb.example.net. For an Ingress on the User cluster a subdomain will be created based on the pattern INGRESS.CLUSTERNAME.lb.example.net The user can reference this URL in his DNS as a CNAME for a customer URL e.g. example.com -> CNAME INGRESS.CLUSTERNAME.lb.example.net 
To enable envoy to forward the customer URL, in the Ingress both URLs must set.
Envoy will forward the traffic based on the HTTPProxy to the service and Kubernetes will forward the traffic from the service on the LB cluster to the Endpoints of the User cluster. 

#### TLS and Certificates

- Follows soon

## Alternatives considered

Solved by GKE with Anthos - Cloud only Solution 

## Task & effort:

Sir do a lot