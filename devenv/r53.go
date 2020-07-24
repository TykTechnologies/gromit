package devenv

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/route53iface"
	"github.com/rs/zerolog/log"
)

// UpsertTaskDNS will create or update the given record
func UpsertTaskDNS(r53 route53iface.ClientAPI, region string, name string, ip string) (string, error) {
	fqdn := fmt.Sprintf("%s.%s", name, e.Domain)
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []route53.Change{
				{
					Action: route53.ChangeActionUpsert,
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:   aws.String(fqdn),
						Region: route53.ResourceRecordSetRegion(region),
						TTL:    func() *int64 { i := int64(300); return &i }(),
						Type:   route53.RRType("A"),
						ResourceRecords: []route53.ResourceRecord{
							route53.ResourceRecord{
								Value: aws.String(ip),
							},
						},
					},
				},
			},
			Comment: aws.String("[CI] update from gromit"),
		},
		HostedZoneId: aws.String(e.ZoneID),
	}

	req := r53.ChangeResourceRecordSetsRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("r53upsert", result)
	if err != nil {
		return "", err
	}
	return fqdn, nil
}
