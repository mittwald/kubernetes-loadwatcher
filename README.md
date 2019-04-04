# Kubernetes Load Watcher

[![Build Status](https://travis-ci.com/mittwald/kubernetes-loadwatcher.svg?branch=master)](https://travis-ci.com/mittwald/kubernetes-loadwatcher)
[![Docker Repository on Quay](https://quay.io/repository/mittwald/kubernetes-loadwatcher/status "Docker Repository on Quay")](https://quay.io/repository/mittwald/kubernetes-loadwatcher)

Automatically taint and evict nodes with high CPU load.

## Synopsis

By default, Kubernetes will not evict Pods from a node based on CPU usage, since CPU is considered a compressible resource. However, in some cases it might be desirable to actually evict some pods from a node with high CPU load (or at least to prevent Kubernetes from scheduling even more pods on a node that is already overloaded).

This project contains a small Kubernetes controller that watches each node's CPU load; when a certain threshold is exceeded, the node will be tainted (so that no additional workloads are scheduled on an already-overloaded node) and finally the controller will start to evict Pods from the node.

## Installation

This repository contains a Helm chart that can be used to install the controller; it needs to be run as a DaemonSet on every node.

```console
> git clone github.com/mittwald/kubernetes-loadwatcher
> helm upgrade \
    --install \
    --namespace kube-system \
    loadwatcher \
    ./kubernetes-loadwatcher/chart
``` 

## How it works

This controller can be started with two threshold flags: `-taint-threshold` and `-evict-threshold`. The controller will continuously monitor a node's CPU load.

- If the CPU load (5min average) exceeds the _taint threshold_, the node will be tainted with a `loadwatcher.mittwald.systems/load-exceeded` taint with the `PreferNoSchedule` effect. This will instruct Kubernetes to not schedule any additional workloads on this node if at all possible. 
- If the CPU load (both 5min and 15min average) falls back below the _taint threshold_, the taint will be removed again.
- If the CPU load (15 min average) exceeds the _eviction threshold_, the controller will pick a suitable Pod running on the node and evict it. However, the following types of Pods will _not_ be evicted:

    - Pods with the `Guaranteed` QoS class
    - Pods belonging to Stateful Sets
    - Pods belonging to Daemon Sets
    - Standalone pods not managed by any kind of controller
    - Pods running in the `kube-system` namespace or with a critical `priorityClassName`
    
  Among the remaining pods, pods with the `BestEfford` QoS class will be preferred for eviction.
  
After a Pod was evicted, the next Pod will be evicted after a configurable _eviction backoff_ (controllable using the `evict-backoff` argument) if the load15 is still above the _eviction threshold_.
