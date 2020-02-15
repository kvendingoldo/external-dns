package provider

import (
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestBlueCatCreateRecords(t *testing.T) {
	customTTL := endpoint.TTL(60)
	provider, _ := NewBlueCatProvider(t, NewDomainFilter([]string{"ext-dns-test-2.teapot.zalan.do."}), NewZoneIDFilter([]string{}), NewZoneTypeFilter(""), defaultEvaluateTargetHealth, false, []*endpoint.Endpoint{})

	records := []*endpoint.Endpoint{
		endpoint.NewEndpoint("create-test.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, "1.2.3.4"),
		endpoint.NewEndpoint("create-test.zone-2.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, "8.8.8.8"),
		endpoint.NewEndpointWithTTL("create-test-cname-custom-ttl.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, customTTL, "172.17.0.1"),
		endpoint.NewEndpoint("create-test-cname.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeCNAME, "foo.elb.amazonaws.com"),
		endpoint.NewEndpoint("create-test-multiple.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, "8.8.8.8", "8.8.4.4"),
	}

	require.NoError(t, provider.CreateRecords(context.Background(), records))

	records, err := provider.Records(context.Background())
	require.NoError(t, err)

	validateEndpoints(t, records, []*endpoint.Endpoint{
		endpoint.NewEndpointWithTTL("create-test.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, endpoint.TTL(recordTTL), "1.2.3.4"),
		endpoint.NewEndpointWithTTL("create-test.zone-2.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, endpoint.TTL(recordTTL), "8.8.8.8"),
		endpoint.NewEndpointWithTTL("create-test-cname-custom-ttl.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, customTTL, "172.17.0.1"),
		endpoint.NewEndpointWithTTL("create-test-cname.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeCNAME, endpoint.TTL(recordTTL), "foo.elb.amazonaws.com"),
		endpoint.NewEndpointWithTTL("create-test-multiple.zone-1.ext-dns-test-2.teapot.zalan.do", endpoint.RecordTypeA, endpoint.TTL(recordTTL), "8.8.8.8", "8.8.4.4"),
	})
}