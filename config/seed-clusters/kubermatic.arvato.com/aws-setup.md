## AWS Install

Clone bootkube https://github.com/kubernetes-incubator/bootkube.git

and checkout version v0.3.1

### Choose a cluster prefix

This can be changed to identify separate clusters.

```
export CLUSTER_PREFIX=loodse
```

### Configure Security Groups

Make note of the `GroupId` output of this command, as it will be referenced later

```
$ aws ec2 create-security-group --region eu-central-1 --group-name ${CLUSTER_PREFIX}-sg --description "Security group for ${CLUSTER_PREFIX} cluster"
GroupID: "sg-abcdefg"
```

Next, create the security group rules.

```
$ aws ec2 authorize-security-group-ingress --region eu-central-1 --group-name ${CLUSTER_PREFIX}-sg --protocol tcp --port 22 --cidr 0.0.0.0/0
$ aws ec2 authorize-security-group-ingress --region eu-central-1 --group-name ${CLUSTER_PREFIX}-sg --protocol tcp --port 443 --cidr 0.0.0.0/0
$ aws ec2 authorize-security-group-ingress --region eu-central-1 --group-name ${CLUSTER_PREFIX}-sg --protocol tcp --port 0-65535 --source-group sg-db7573b3
```

### Create a key-pair

```
$ aws ec2 create-key-pair --key-name ${CLUSTER_PREFIX}-key --query 'KeyMaterial' --output text > ${CLUSTER_PREFIX}-key.pem
$ chmod 400 ${CLUSTER_PREFIX}-key.pem
```

### Launch Nodes

To find the latest CoreOS alpha/beta/stable images, please see the [CoreOS AWS Documentation](https://coreos.com/os/docs/latest/booting-on-ec2.html). Then replace the `--image-id` in the command below.

In the command below, replace `<K8S_SG_ID>` with the security-group-id noted earlier.

```
$ aws ec2 run-instances --region eu-central-1 --image-id ami-f603c599 --security-group-ids sg-db7573b3 --count 1 --instance-type m3.medium --key-name ${CLUSTER_PREFIX}-key --query 'Instances[0].InstanceId' --iam-instance-profile Name=kubernetes-master-role
"i-0d336aa6585ac64d5"
```

Next we will use the output of the above command (instance-id) in place of <INSTANCE_ID> in the command below:

```
$ aws ec2 describe-instances --region eu-central-1 --instance-ids i-0d336aa6585ac64d5 --query 'Reservations[0].Instances[0].PublicIpAddress'
```

### Bootstrap Master

We can then use the public-ip to initialize a master node:

```
$ IDENT=loodse-key.pem ./init-master.sh 35.156.246.178
```

After the master bootstrap is complete, you can continue to add worker nodes. Or cluster state can be inspected via kubectl:

```
$ kubectl --kubeconfig=cluster/auth/kubeconfig get nodes
```

### Add Workers

Run the `Launch Nodes` step for each additional node you wish to add, then using the public-ip, run:

```
$ aws ec2 run-instances --region eu-central-1 --image-id ami-f603c599 --security-group-ids sg-db7573b3 --count 1 --instance-type m3.medium --key-name ${CLUSTER_PREFIX}-key --query 'Instances[0].InstanceId' --iam-instance-profile Name=kubernetes-minion-role
"i-08633d543ee9480ff"

$ aws ec2 describe-instances --region eu-central-1 --instance-ids i-08633d543ee9480ff --query 'Reservations[0].Instances[0].PublicIpAddress'

IDENT=loodse-key.pem ./init-worker.sh 35.157.19.109 cluster/auth/kubeconfig

$ aws ec2 run-instances --region eu-central-1 --image-id ami-f603c599 --security-group-ids sg-db7573b3 --count 1 --instance-type m3.medium --key-name ${CLUSTER_PREFIX}-key --query 'Instances[0].InstanceId' --iam-instance-profile Name=kubernetes-minion-role
"i-074642450a1d84db9"

$ aws ec2 describe-instances --region eu-central-1 --instance-ids i-074642450a1d84db9 --query 'Reservations[0].Instances[0].PublicIpAddress'

IDENT=loodse-key.pem ./init-worker.sh 35.156.196.107 cluster/auth/kubeconfig
```

**NOTE:** It can take a few minutes for each node to download all of the required assets / containers.
 They may not be immediately available, but the state can be inspected with:

```
$ kubectl --kubeconfig=cluster/auth/kubeconfig get nodes
```
