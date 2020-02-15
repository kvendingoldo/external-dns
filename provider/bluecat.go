package provider

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/route53"
	log "github.com/sirupsen/logrus"
	"regexp"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"

	"github.com/go-resty/resty"
)

const (
	// 10 minutes default timeout if not configured using flags
	blueCatDefaultTTL    = 600
	blueCatRestAPIPrefix = "Services/REST/v1"
)

type BlueCatProvider struct {
	Server       string
	AuthToken    string


	serverId     string
	parentId     string
	viewId       string

	InsecureFlag bool
	DryRun       bool
}

type BlueCatConfig struct {
	Server       string
	Username     string
	Password     string
	
	InsecureFlag bool
	DryRun       bool
}

type changes struct {
	Action   string
	Endpoint *endpoint.Endpoint
}

func NewBlueCatProvider(blueCatConfig BlueCatConfig) (Provider, error) {

	token, err := getAuthToken(blueCatConfig.Server, blueCatConfig.Username, blueCatConfig.Password)

	if err != nil {
		return nil, fmt.Errorf("%s - NewSession initialization", err)
	}

	provider := &BlueCatProvider{
		Server:       blueCatConfig.Server,
		AuthToken:    token,
		serverId:     "todo",
		parentId:     "todo",
		viewId:       "todo",
		InsecureFlag: blueCatConfig.InsecureFlag,
		DryRun:       blueCatConfig.DryRun,
	}

	return provider, nil
}

// getAuthToken returns the BlueCat session authentication token which is used to authenticate
// all of the API calls to the BlueCat server.
func getAuthToken(server, user, pass string) (string, error) {
	sessionToken := regexp.MustCompile(`^.*(BAMAuthToken:\s+[\w=]+)\s+.*$`)
	connErr := regexp.MustCompile(`Get https.*:\s+(.*)`)

	loginReq := fmt.Sprintf("https://%s/%s/login?username=%s&password=%s", server, restAPIPrefix, user, pass)

	resp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		Get(loginReq)

	if err != nil {
		formatted := connErr.ReplaceAllString(fmt.Sprintf("%s", err), "${1}")
		return "", fmt.Errorf("%s - getAuthToken login", formatted)
		//return "", fmt.Errorf("%s - getAuthToken login", err)
	}

	token := sessionToken.FindStringSubmatch(resp.String())
	if len(token) <= 0 {
		return "", fmt.Errorf("%s - getAuthToken token parse", string(resp.Body()))
	}

	return token[1], nil
}

// KEK TODO

// Records returns the endpoints this provider knows about
func (p *BlueCatProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.zonesFiltered()
	if err != nil {
		return nil, err
	}

	var endpoints []*endpoint.Endpoint

	for _, zone := range zones {

		// TODO handle Header Codes
		zoneData, _, err := p.client.GetZone(zone.String())
		if err != nil {
			return nil, err
		}

		for _, record := range zoneData.Records {
			if supportedRecordType(record.Type) {
				endpoints = append(endpoints, endpoint.NewEndpointWithTTL(
					record.Domain,
					record.Type,
					endpoint.TTL(record.TTL),
					record.ShortAns...,
				),
				)
			}
		}
	}

	return endpoints, nil
}

func (p *BlueCatProvider) submitChanges(changes []*changes) error {
	// return early if there is nothing to change
	if len(changes) == 0 {
		return nil
	}

	zones, err := p.zonesFiltered()
	if err != nil {
		return err
	}

	// separate into per-zone change sets to be passed to the API.
	changesByZone := ns1ChangesByZone(zones, changes)
	for zoneName, changes := range changesByZone {
		for _, change := range changes {
			record := ns1BuildRecord(zoneName, change)
			logFields := log.Fields{
				"record": record.Domain,
				"type":   record.Type,
				"ttl":    record.TTL,
				"action": change.Action,
				"zone":   zoneName,
			}

			log.WithFields(logFields).Info("Changing record.")

			if p.dryRun {
				continue
			}

			switch change.Action {
			case ns1Create:
				_, err := p.client.CreateRecord(record)
				if err != nil {
					return err
				}
			case ns1Delete:
				_, err := p.client.DeleteRecord(zoneName, record.Domain, record.Type)
				if err != nil {
					return err
				}
			case ns1Update:
				_, err := p.client.UpdateRecord(record)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ApplyChanges applies a given set of changes in a given zone.
func (p *BlueCatProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {

	return changes
}

// CreateRecords creates a given set of DNS records in the given hosted zone.
func (p *BlueCatProvider) CreateRecords(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	return p.doRecords(ctx, route53.ChangeActionCreate, endpoints)
}

// UpdateRecords updates a given set of old records to a new set of records in a given hosted zone.
func (p *BlueCatProvider) UpdateRecords(ctx context.Context, endpoints, _ []*endpoint.Endpoint) error {
	return p.doRecords(ctx, route53.ChangeActionUpsert, endpoints)
}

// DeleteRecords deletes a given set of DNS records in a given zone.
func (p *BlueCatProvider) DeleteRecords(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	return p.doRecords(ctx, route53.ChangeActionDelete, endpoints)
}
