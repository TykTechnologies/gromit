package devenv

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/rs/zerolog/log"
)

// Will only fetch fargate tasks that want to be or are RUNNING
func getClusterTasks(svc ecsiface.ClientAPI, cluster string) ([]string, error) {
	input := &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		DesiredStatus: ecs.DesiredStatus("RUNNING"),
		LaunchType:    ecs.LaunchType("FARGATE"),
	}

	req := svc.ListTasksRequest(input)
	result, err := req.Send(context.Background())
	if err != nil {
		return []string{}, err
	}
	log.Trace().Interface("taskarns", result)

	return result.TaskArns, nil
}

// getTaskENI depends on there being just one container per task
func getTaskENI(svc ecsiface.ClientAPI, cluster string, taskid string) (string, string, error) {
	input := &ecs.DescribeTasksInput{
		Tasks: []string{
			taskid,
		},
		Cluster: aws.String(cluster),
	}

	req := svc.DescribeTasksRequest(input)
	result, err := req.Send(context.Background())
	if err != nil {
		return "", "", err
	}
	log.Trace().Interface("taskdetails", result)

	if len(result.Tasks) > 1 {
		log.Warn().
			Interface("tasks", result.Tasks).
			Msg("Should have got just one task, using first.")
	}
	task := result.Tasks[0].Containers[0].Name
	for _, d := range result.Tasks[0].Attachments[0].Details {
		if *d.Name == "networkInterfaceId" {
			return *task, *d.Value, nil
		}
	}
	return "", "", errors.New("no eni found for " + taskid)
}

func getPublicIP(svc ec2iface.ClientAPI, eni string) (string, error) {
	input := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{
			eni,
		},
	}

	req := svc.DescribeNetworkInterfacesRequest(input)
	result, err := req.Send(context.Background())
	if err != nil {
		return "", err
	}
	log.Trace().Interface("netifaces", result)

	if len(result.NetworkInterfaces) > 0 {
		assoc := result.NetworkInterfaces[0].Association
		if assoc != nil && assoc.PublicIp != nil {
			return *assoc.PublicIp, nil
		}
	}
	return "", errors.New("No public IP")
}

// UpdateClusterIPs is the entrypoint for CLI
func UpdateClusterIPs(cluster string, zoneid string, domain string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Error().Err(err).Msg("unable to load SDK config,")
		return err
	}
	region, flag, err := external.GetRegion(external.Configs{cfg})
	log.Trace().Msgf("got region flag: %v, not sure what this is supposed to indicate", flag)
	if err != nil {
		log.Error().Err(err).Msg("unable to find region,")
		return err
	}

	fargate := ecs.New(cfg)
	tasks, err := getClusterTasks(fargate, cluster)
	for _, task := range tasks {
		taskName, eni, err := getTaskENI(fargate, cluster, task)
		if err != nil {
			log.Warn().Err(err).Msgf("could not get eni for %s.%s", cluster, task)
			continue
		}
		log.Debug().Msgf("Found eni %s for task %s.%s (%s)", eni, cluster, taskName, task)
		ip, err := getPublicIP(ec2.New(cfg), eni)
		if err != nil {
			log.Warn().Err(err).Msgf("could not get ip for %s.%s", cluster, taskName)
			continue
		}
		log.Debug().Msgf("Found ip %s for task %s.%s", ip, cluster, taskName)
		fqdn := fmt.Sprintf("%s.%s", taskName, domain)

		err = UpsertTaskDNS(route53.New(cfg), region, zoneid, fqdn, ip)
		if err != nil {
			log.Warn().Err(err).Msgf("could not bind %s", ip)
			continue
		}
		log.Info().Msgf("Bound %s to %s", ip, fqdn)
	}
	return nil
}
