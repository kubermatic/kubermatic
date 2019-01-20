package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

var (
	awsAccessKeyID     string
	awsSecretAccessKey string
)

func main() {
	flag.StringVar(&awsAccessKeyID, "aws-access-key-id", "", "AWS AccessKeyID")
	flag.StringVar(&awsSecretAccessKey, "aws-secret-access-key", "", "AWS SecretAccessKey")
	flag.Parse()

	// We need to have 1 region initially, afterwards we dynamically fetch the other regions
	sess, err := getSession("eu-central-1")
	if err != nil {
		log.Fatalf("failed to get EC2 client: %v", err)
	}
	mainEC2Client := ec2.New(sess)

	regionsOut, err := mainEC2Client.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		log.Fatalf("failed to get EC2 regions: %v", err)
	}

	for _, region := range regionsOut.Regions {
		sess, err = getSession(*region.RegionName)
		if err != nil {
			log.Fatalf("failed to create AWS session for region '%s': %v", *region.RegionName, err)
		}

		log.Printf("Cleaning up resources in region %s", *region.RegionName)
		if err := cleanup(ec2.New(sess), elb.New(sess)); err != nil {
			log.Fatalf("failed to cleanup resources in region '%s': %v", *region.RegionName, err)
		}
	}
}

func cleanup(ec2Client *ec2.EC2, elbClient *elb.ELB) error {
	instancesOut, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		return fmt.Errorf("failed to list instances: %v", err)
	}

	var instanceIDs []*string
	for _, reservation := range instancesOut.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State != nil && instance.State.Name != nil && *instance.State.Name == ec2.InstanceStateNameRunning {
				if ec2CanBeDeleted(instance.Tags) {
					instanceIDs = append(instanceIDs, instance.InstanceId)
				}
			}
		}
	}

	if _, err := ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: instanceIDs}); err != nil {
		log.Printf("failed to terminate instances: %v", err)
	}

	var lbNames []*string
	lbOut, err := elbClient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		return fmt.Errorf("failed to list load balancers: %v", err)
	}

	for _, lb := range lbOut.LoadBalancerDescriptions {
		lbNames = append(lbNames, lb.LoadBalancerName)
	}

	// Get tags for all LB's
	lbTagOut, err := elbClient.DescribeTags(&elb.DescribeTagsInput{LoadBalancerNames: lbNames})
	if err != nil {
		return fmt.Errorf("failed to get load balancer tags: %v", err)
	}
	for _, lbDescription := range lbTagOut.TagDescriptions {
		if elbCanBeDeleted(lbDescription.Tags) {
			if _, err := elbClient.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{LoadBalancerName: lbDescription.LoadBalancerName}); err != nil {
				log.Printf("failed to delete load balancer '%s': %v", *lbDescription.LoadBalancerName, err)
			}
		}
	}

	volumeOut, err := ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		return fmt.Errorf("failed to list volumes: %v", err)
	}

	for _, volume := range volumeOut.Volumes {
		if ec2CanBeDeleted(volume.Tags) {
			if _, err := ec2Client.DeleteVolume(&ec2.DeleteVolumeInput{VolumeId: volume.VolumeId}); err != nil {
				log.Printf("failed to delete volume '%s': %v", *volume.VolumeId, err)
			}
		}
	}

	return nil
}

func getSession(region string) (*session.Session, error) {
	config := aws.NewConfig()
	if region != "" {
		config = config.WithRegion(region)
	}
	config = config.WithCredentials(credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""))
	config = config.WithMaxRetries(3)
	return session.NewSession(config)
}

func ec2CanBeDeleted(tags []*ec2.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == "keep" && *tag.Value == "true" {
			return false
		}
	}

	return true
}

func elbCanBeDeleted(tags []*elb.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == "keep" && *tag.Value == "true" {
			return false
		}
	}

	return true
}
