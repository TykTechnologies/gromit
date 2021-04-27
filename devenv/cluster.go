package devenv

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/rs/zerolog/log"
)

// getTaskENI depends on there being just one container per task
func (c *GromitCluster) getTaskENI(taskid string) (string, string, error) {
	input := &ecs.DescribeTasksInput{
		Tasks: []string{
			taskid,
		},
		Cluster: aws.String(c.Name),
	}

	req := c.ecsClient.DescribeTasksRequest(input)
	result, err := req.Send(context.Background())
	if err != nil {
		return "", "", err
	}
	c.log.Trace().Interface("taskdetails", result)

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
	return "", "", fmt.Errorf("no eni found for %s", taskid)
}

func (c *GromitCluster) getPublicIP(eni string) (string, error) {
	input := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{
			eni,
		},
	}

	req := c.ec2Client.DescribeNetworkInterfacesRequest(input)
	result, err := req.Send(context.Background())
	if err != nil {
		return "", err
	}
	c.log.Trace().Interface("netifaces", result)

	if len(result.NetworkInterfaces) > 0 {
		assoc := result.NetworkInterfaces[0].Association
		if assoc != nil && assoc.PublicIp != nil {
			return *assoc.PublicIp, nil
		}
	}
	return "", fmt.Errorf("no public IP")
}

// Will only fetch fargate tasks that want to be or are RUNNING
func (c *GromitCluster) getTasks() ([]string, error) {
	input := &ecs.ListTasksInput{
		Cluster:       aws.String(c.Name),
		DesiredStatus: ecs.DesiredStatus("RUNNING"),
		LaunchType:    ecs.LaunchType("FARGATE"),
	}

	req := c.ecsClient.ListTasksRequest(input)
	result, err := req.Send(context.Background())
	if err != nil {
		return []string{}, err
	}
	c.log.Trace().Interface("taskarns", result).Msg("tasks")
	return result.TaskArns, nil
}

// Populate will look up all tasks in this cluster and fill in the tasks array with IPs
func (c *GromitCluster) Populate() error {
	tasks, err := c.getTasks()
	if err != nil {
		return err
	}
	for _, t := range tasks {
		tname, eni, err := c.getTaskENI(t)
		if err != nil {
			continue
		}
		ip, err := c.getPublicIP(eni)
		if err != nil {
			continue
		}
		c.tasks = append(c.tasks, GromitTask{
			Name: tname,
			IP:   ip,
		})
	}
	return nil
}

// SyncDNS will update the public Route53 records for cluster in zoneid
// The FQDN is constructed by appending domain to the task name
func (c *GromitCluster) SyncDNS(action route53.ChangeAction, zoneid string, domain string) error {
	var changes []route53.Change
	for _, t := range c.tasks {
		fqdn := fmt.Sprintf("%s.%s.%s", t.Name, c.Name, domain)
		c.log.Trace().Str("fqdn", fqdn).Str("A record", t.IP).Msgf("for task %s", t.Name)
		changes = append(changes, route53.Change{
			Action: action,
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name:   aws.String(fqdn),
				Region: route53.ResourceRecordSetRegion(c.Region),
				TTL:    aws.Int64(10),
				Type:   route53.RRType("A"),
				ResourceRecords: []route53.ResourceRecord{
					{
						Value: aws.String(t.IP),
					},
				},
			},
		})
	}
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
			Comment: aws.String("[CI] update from gromit"),
		},
		HostedZoneId: aws.String(zoneid),
	}

	req := c.r53Client.ChangeResourceRecordSetsRequest(input)
	result, err := req.Send(context.Background())
	c.log.Trace().Interface("r53execute", result).Msg("r53 bulk upsert")
	return err
}

// GetGromitCluster returns a fully populated gromit cluster
func GetGromitCluster(name string) (*GromitCluster, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return &GromitCluster{}, err
	}
	region, _, err := external.GetRegion(external.Configs{cfg})
	if err != nil {
		log.Error().Err(err).Msg("unable to find region,")
		return &GromitCluster{}, err
	}
	gc := &GromitCluster{
		Name:      name,
		Region:    region,
		r53Client: route53.New(cfg),
		ecsClient: ecs.New(cfg),
		ec2Client: ec2.New(cfg),
		aws:       cfg,
		log:       log.With().Str("cluster", name).Logger(),
	}
	err = gc.Populate()
	return gc, err
}

// ListClusters will return a list running ECS clusters. Just the names.
func ListClusters(svc ecsiface.ClientAPI) ([]string, error) {
	req := svc.ListClustersRequest(&ecs.ListClustersInput{})
	result, err := req.Send(context.Background())
	if err != nil {
		return []string{}, err
	}

	var clusters []string
	for _, c := range result.ClusterArns {
		clusters = append(clusters, strings.Split(c, "/")[1])
	}
	return clusters, nil
}

// FastFetchClusters will fetch an array of clusters concurrently
func FastFetchClusters(cnames []string) []*GromitCluster {
	var wg sync.WaitGroup
	errChan := make(chan error)

	var clusters []*GromitCluster
	for _, c := range cnames {
		c := c
		wg.Add(1)
		go func() {
			gc, err := GetGromitCluster(c)
			if err != nil {
				errChan <- err
			}
			clusters = append(clusters, gc)
			wg.Done()
		}()
	}

	// Wait to close waitgroup
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		log.Error().Err(err).Msg("fast fetch clusters")
	}
	return clusters
}
