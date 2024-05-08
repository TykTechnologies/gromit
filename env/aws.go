package env

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
)

type Client struct {
	cfg aws.Config
}

// NewClientFromProfile returns an object that can be used to control the
// environments running on AWS
func NewClientFromProfile(profile string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		return nil, err
	}
	stsc := sts.NewFromConfig(cfg)
	identity, err := stsc.GetCallerIdentity(
		ctx,
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed client self-test: %v", err)
	}
	log.Debug().Str("acct", *identity.Account).Str("arn", *identity.Arn).Str("user", *identity.UserId).Msg("identity")
	return &Client{
		cfg: cfg,
	}, nil
}

// clusterMap contains taskname.clustername -> ENI
type clusterMap map[string]string

// ExposeCluster will upsert A records with the public IPs for all
// tasks in the cluster that have public IPs. The records will have
// the form taskname.clustername.domain.
func (c *Client) Expose(cluster, zone string) error {
	log.Logger = log.With().Str("cluster", cluster).Logger()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	r53c := route53.NewFromConfig(c.cfg)
	lhzi := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(zone),
	}
	lhzo, err := r53c.ListHostedZonesByName(ctx, lhzi)
	if err != nil {
		return fmt.Errorf("could not find zone for %s: %w", zone, err)
	}
	var zid string
	if len(lhzo.HostedZones) > 0 {
		zid = *lhzo.HostedZones[0].Id
	}

	ecsc := ecs.NewFromConfig(c.cfg)
	clusterMap, err := getClusterMap(ctx, ecsc, cluster)
	if err != nil {
		return err
	}
	ec2c := ec2.NewFromConfig(c.cfg)
	var changes []r53types.Change
	for name, eni := range clusterMap {
		ip, err := getPublicIP(ctx, ec2c, eni)
		if err != nil {
			log.Warn().Err(err).Msgf("could not get ip for %s", name)
		}
		changes = append(changes, r53types.Change{
			Action: r53types.ChangeActionUpsert,
			ResourceRecordSet: &r53types.ResourceRecordSet{
				Name:          aws.String(fmt.Sprintf("%s.%s", name, zone)),
				Region:        r53types.ResourceRecordSetRegion(c.cfg.Region),
				TTL:           aws.Int64(10),
				Type:          r53types.RRType("A"),
				SetIdentifier: aws.String(cluster),
				ResourceRecords: []r53types.ResourceRecord{
					{
						Value: aws.String(ip),
					},
				},
			},
		})
		log.Info().Msgf("adding change %s → %s", name, ip)
	}
	r53i := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &r53types.ChangeBatch{
			Changes: changes,
			Comment: aws.String("[CI] update from gromit"),
		},
		HostedZoneId: aws.String(zid),
	}
	r53o, err := r53c.ChangeResourceRecordSets(ctx, r53i)
	log.Trace().Interface("r53o", r53o).Msg("route53 upsert")
	if err != nil {
		return err
	}
	return nil
}

// getClusterMap assumes that there is only one container per task. If
// there are more than one container in a task, the first one returned
// by the API will be used. Only Fargate™ tasks are supported as the
// rest of the processing only works for this case.
func getClusterMap(ctx context.Context, svc *ecs.Client, cluster string) (clusterMap, error) {
	cm := make(clusterMap)
	lti := &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		DesiredStatus: ecstypes.DesiredStatus("RUNNING"),
		LaunchType:    ecstypes.LaunchType("FARGATE"),
	}

	lto, err := svc.ListTasks(ctx, lti)
	if err != nil {
		return cm, err
	}
	log.Trace().Interface("lto", lto).Msg("found tasks")

	dti := &ecs.DescribeTasksInput{
		Tasks:   lto.TaskArns,
		Cluster: aws.String(cluster),
	}

	dto, err := svc.DescribeTasks(ctx, dti)
	if err != nil {
		return cm, err
	}
	log.Trace().Interface("dto", dto)
	if len(dto.Tasks) < 1 {
		return cm, fmt.Errorf("no tasks in cluster %s", cluster)
	}
	for _, task := range dto.Tasks {
		if *task.LastStatus == "RUNNING" {
			for _, d := range task.Attachments[0].Details {
				if *d.Name == "networkInterfaceId" {
					cm[fmt.Sprintf("%s.%s", *task.Containers[0].Name, cluster)] = *d.Value
				}
			}
		}
	}
	return cm, nil
}

// getPublicIP will return the IP associated with the network
// interface that is returned first
func getPublicIP(ctx context.Context, svc *ec2.Client, eni string) (string, error) {
	dnii := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{
			eni,
		},
	}
	dnio, err := svc.DescribeNetworkInterfaces(ctx, dnii)
	if err != nil {
		return "", err
	}
	log.Trace().Interface("netifaces", dnio)

	if len(dnio.NetworkInterfaces) > 0 {
		assoc := dnio.NetworkInterfaces[0].Association
		if assoc != nil && assoc.PublicIp != nil {
			return *assoc.PublicIp, nil
		}
	}
	return "", fmt.Errorf("no public IP")
}
