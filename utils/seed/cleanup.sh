#!/bin/bash

source ./config.sh

# Clean up Master
for ((i = 0; i < ${#MASTER_HOSTNAMES[@]}; i++)); do
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo kubeadm reset || true"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo rm -rf /etc/kubernetes || true"
done

# Clean up ETCD
for ((i = 0; i < ${#ETCD_HOSTNAMES[@]}; i++)); do
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo systemctl stop etcd || true"
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo systemctl disable etcd || true"
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo rm -rf /var/lib/etcd || true"
done

# Clean up Worker
for ((i = 0; i < ${#WORKER_PUBLIC_IPS[@]}; i++)); do
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo kubeadm reset || true"
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo rm -rf /etc/kubernetes || true"
done
