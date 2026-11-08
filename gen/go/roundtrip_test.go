package gen_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/timestamppb"

	routerv1 "github.com/bugnet-sh/bugnet-proto/gen/go/bugnet/router/v1"
	"github.com/bugnet-sh/bugnet-proto/gen/go/bugnet/router/v1/routerv1connect"
)

func TestExecRequestRoundTrip(t *testing.T) {
	in := &routerv1.ExecRequest{
		Handle:         "h-1",
		Argv:           []string{"echo", "hi"},
		Cwd:            "/work",
		Env:            map[string]string{"A": "1", "B": "2"},
		IdempotencyKey: "key-1",
		WaitMs:         1500,
		Until: &routerv1.Until{
			Exit:          true,
			OutputPattern: "ready",
			IdleMs:        250,
		},
	}
	raw, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := &routerv1.ExecRequest{}
	if err := proto.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !proto.Equal(in, out) {
		t.Fatalf("round trip mismatch: %v != %v", in, out)
	}
}

func TestWatchJobSkipAndTruncatedListingRoundTrip(t *testing.T) {
	msgs := []proto.Message{
		&routerv1.WatchJobResponse{
			Frame: &routerv1.WatchJobResponse_Skip{Skip: &routerv1.JobSkip{
				StdoutOffset: 1 << 40, StderrOffset: 7,
			}},
			Cursor: "1-0",
		},
		&routerv1.ListFilesResponse{
			Entries:   []*routerv1.FileEntry{{Path: "/a", Size: 3}},
			Truncated: true,
		},
	}
	for _, in := range msgs {
		raw, err := proto.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		out := in.ProtoReflect().New().Interface()
		if err := proto.Unmarshal(raw, out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !proto.Equal(in, out) {
			t.Fatalf("round trip mismatch: %v != %v", in, out)
		}
	}
}

func TestCheckExtendPlacementRoundTrip(t *testing.T) {
	deadline := timestamppb.New(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC))
	msgs := []proto.Message{
		&routerv1.CheckExtendPlacementRequest{Handle: "h-1", PackageId: "small-10m-more"},
		&routerv1.CheckExtendPlacementResponse{
			Placement: &routerv1.Placement{
				PlacementId: "p-1", State: routerv1.PlacementState_PLACEMENT_STATE_READY,
			},
			Deadline: deadline,
		},
	}
	for _, in := range msgs {
		raw, err := proto.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		out := in.ProtoReflect().New().Interface()
		if err := proto.Unmarshal(raw, out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !proto.Equal(in, out) {
			t.Fatalf("round trip mismatch: %v != %v", in, out)
		}
	}
}

func TestExtensionPricingRoundTrip(t *testing.T) {
	more := &routerv1.PackageOption{
		PackageId: "small-10m-more", Resources: &routerv1.Resources{Vcpu: 2, MemoryMib: 2048},
		Minutes: 10, PriceMicros: 45000, Available: true,
	}
	msgs := []proto.Message{
		&routerv1.CheckExtendPlacementResponse{
			Placement: &routerv1.Placement{PlacementId: "p-1"}, Package: more,
		},
		&routerv1.ListExtensionsRequest{Handle: "h-1"},
		&routerv1.ListExtensionsResponse{Options: []*routerv1.PackageOption{
			more, {PackageId: "small-10m", Minutes: 10, UnavailableReason: "exceeds_lifetime"},
		}},
	}
	for _, in := range msgs {
		raw, err := proto.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		out := in.ProtoReflect().New().Interface()
		if err := proto.Unmarshal(raw, out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !proto.Equal(in, out) {
			t.Fatalf("round trip mismatch: %v != %v", in, out)
		}
	}
}

func TestPricePackageRoundTrip(t *testing.T) {
	req := &routerv1.PricePackageRequest{
		PackageId: "small-10m", Requirements: &routerv1.Requirements{Ssh: true},
	}
	eq(t, proto.Equal(roundTrip(t, req), req), true)
	resp := &routerv1.PricePackageResponse{Option: &routerv1.PackageOption{
		PackageId: "small-10m", Minutes: 10, PriceMicros: 50000,
		UnavailableReason: "feature_unavailable",
	}}
	eq(t, proto.Equal(roundTrip(t, resp), resp), true)
}

func TestPricePackageFollowsEstimate(t *testing.T) {
	methods := service(t, "SandboxRouterService").Methods()
	eq(t, string(methods.Get(0).Name()), "Estimate")
	eq(t, string(methods.Get(1).Name()), "PricePackage")
}

func TestDownloadChecksumRoundTrips(t *testing.T) {
	eq(t, roundTrip(t, &routerv1.CreateTransferRequest{WantSha256: true}).WantSha256, true)
	sum := make([]byte, 32)
	sum[0], sum[31] = 0xab, 0xcd
	eq(t, roundTrip(t, &routerv1.CreateTransferResponse{Sha256: sum}).Sha256, sum)
	eq(t, len(roundTrip(t, &routerv1.CreateTransferResponse{}).Sha256), 0)
	eq(t, roundTrip(t, &routerv1.OpenTransferResponse{Sha256: sum}).Sha256, sum)
}

func TestTransferSizePresenceSurvives(t *testing.T) {
	out := roundTrip(t, &routerv1.CreateTransferRequest{SizeBytes: proto.Uint64(0)})
	eq(t, out.SizeBytes != nil && *out.SizeBytes == 0, true)
	eq(t, roundTrip(t, &routerv1.CreateTransferRequest{}).SizeBytes == nil, true)
}

func TestExpectedMinutesPresenceSurvives(t *testing.T) {
	out := roundTrip(t, &routerv1.EstimateRequest{ExpectedMinutes: proto.Uint32(0)})
	eq(t, out.ExpectedMinutes != nil && *out.ExpectedMinutes == 0, true)
	eq(t, roundTrip(t, &routerv1.EstimateRequest{}).ExpectedMinutes == nil, true)
}

func TestPlacementEndedAtRoundTripsAndKeepsTwelveForImage(t *testing.T) {
	ended := timestamppb.New(time.Date(2026, 11, 6, 1, 2, 3, 4000, time.UTC))
	eq(t, proto.Equal(roundTrip(t, &routerv1.Placement{EndedAt: ended}).EndedAt, ended), true)
	eq(t, roundTrip(t, &routerv1.Placement{}).EndedAt == nil, true)
	md := (&routerv1.Placement{}).ProtoReflect().Descriptor()
	twelve := md.Fields().ByNumber(12)
	eq(t, twelve == nil || twelve.Name() == "image", true)
	eq(t, md.ReservedRanges().Has(12), false)
}

func placementEnded(reason routerv1.EndReason) *routerv1.EdgeEvent {
	return &routerv1.EdgeEvent{Event: &routerv1.EdgeEvent_PlacementEnded{
		PlacementEnded: &routerv1.PlacementEnded{PlacementId: "p-1", Reason: reason},
	}}
}

func deadlineMoved() *routerv1.EdgeEvent {
	return &routerv1.EdgeEvent{Event: &routerv1.EdgeEvent_DeadlineMoved{
		DeadlineMoved: &routerv1.DeadlineMoved{
			PlacementId:        "p-1",
			Deadline:           timestamppb.New(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)),
			EdgeBytesRemaining: 2 << 30, EgressBytesRemaining: 1 << 30,
		},
	}}
}

func previewRevoked() *routerv1.EdgeEvent {
	return &routerv1.EdgeEvent{Event: &routerv1.EdgeEvent_PreviewRevoked{
		PreviewRevoked: &routerv1.PreviewRevoked{PlacementId: "p-1", PreviewId: "abc"},
	}}
}

func sshKeyRemoved() *routerv1.EdgeEvent {
	return &routerv1.EdgeEvent{Event: &routerv1.EdgeEvent_SshKeyRemoved{
		SshKeyRemoved: &routerv1.SshKeyRemoved{PlacementId: "p-1", Fingerprint: "SHA256:x"},
	}}
}

func budgetSpent(kind routerv1.BudgetKind) *routerv1.EdgeEvent {
	return &routerv1.EdgeEvent{Event: &routerv1.EdgeEvent_BudgetSpent{
		BudgetSpent: &routerv1.BudgetSpent{PlacementId: "p-1", Kind: kind},
	}}
}

func secretsChanged() *routerv1.EdgeEvent {
	return &routerv1.EdgeEvent{Event: &routerv1.EdgeEvent_SecretsChanged{
		SecretsChanged: &routerv1.SecretsChanged{PlacementId: "p-1"},
	}}
}

func resync() *routerv1.EdgeEvent {
	return &routerv1.EdgeEvent{Event: &routerv1.EdgeEvent_Resync{Resync: &routerv1.Resync{}}}
}

func TestEveryEdgeEventRoundTrips(t *testing.T) {
	for _, ev := range []*routerv1.EdgeEvent{placementEnded(8), deadlineMoved(), previewRevoked(),
		sshKeyRemoved(), budgetSpent(2), secretsChanged(), resync()} {
		eq(t, proto.Equal(roundTrip(t, ev), ev), true)
	}
	eq(t, routerv1.EndReason(8), routerv1.EndReason_END_REASON_SANDBOX_EXITED)
	eq(t, routerv1.BudgetKind(2), routerv1.BudgetKind_BUDGET_KIND_EGRESS)
}

func TestOneofsRoundTrip(t *testing.T) {
	since := timestamppb.New(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC))
	msgs := []proto.Message{
		&routerv1.OpenSshUpstreamResponse{
			Transport: &routerv1.OpenSshUpstreamResponse_Websocket{
				Websocket: &routerv1.WebSocketUpstream{
					Url:     "wss://gate.example/__bugnet/ssh",
					Headers: []*routerv1.HeaderValue{{Name: "Bugnet-Gate", Value: "k"}},
				},
			},
			User: "root", HostKeys: []string{"ssh-ed25519 AAAA"},
		},
		&routerv1.OpenSshUpstreamResponse{Transport: &routerv1.OpenSshUpstreamResponse_Tcp{
			Tcp: &routerv1.TcpUpstream{Address: "203.0.113.7:2022"},
		}},
		&routerv1.ResolveTunnelRequest{
			Caller: &routerv1.ResolveTunnelRequest_TokenSha256{TokenSha256: []byte{1, 2, 3}},
		},
		&routerv1.ResolveTunnelRequest{
			Caller: &routerv1.ResolveTunnelRequest_FetchPlacementId{FetchPlacementId: "p-1"},
		},
		&routerv1.FindPlacementRequest{Target: &routerv1.PlacementTarget{
			Target: &routerv1.PlacementTarget_ProviderSandbox{
				ProviderSandbox: &routerv1.ProviderSandbox{Provider: "boxd", SandboxId: "vm-1"},
			},
		}},
		&routerv1.FindPlacementRequest{Target: &routerv1.PlacementTarget{
			Target: &routerv1.PlacementTarget_Transfer{Transfer: "bnx_abc"},
		}},
		&routerv1.ReportStartsRequest{
			Window: &routerv1.ReportStartsRequest_BenchId{BenchId: "b-1"},
		},
		&routerv1.ReportStartsRequest{
			Window: &routerv1.ReportStartsRequest_Since{Since: since},
			Kinds:  []routerv1.SampleKind{routerv1.SampleKind_SAMPLE_KIND_CANARY},
		},
		&routerv1.CheckEdgeLeasesResponse{
			Leases: map[string]*routerv1.EdgeLease{
				"p-1": {Deadline: since, BytesRemaining: 7},
			},
			RevokedPreviewIds: []string{"abc"},
		},
		&routerv1.CheckSshGrantsResponse{
			Deadlines: map[string]*timestamppb.Timestamp{"g1.x": since},
		},
		&routerv1.GetPlacementResponse{Placement: &routerv1.Placement{
			PlacementId: "p-1", SshAlias: "brave-otter", TerminationReason: "sandbox_exited",
			Bootstrap: &routerv1.Bootstrap{
				State: routerv1.BootstrapState_BOOTSTRAP_STATE_DONE, Path: "~/src/app",
				Revision: "0123abc",
			},
		}},
	}
	for _, in := range msgs {
		if out := roundTrip(t, in); !proto.Equal(in, out) {
			t.Fatalf("round trip mismatch: %v != %v", in, out)
		}
	}
}

func TestServiceProcedures(t *testing.T) {
	eq(t, routerv1connect.EdgeServiceResolvePreviewProcedure,
		"/bugnet.router.v1.EdgeService/ResolvePreview")
	eq(t, routerv1connect.SecretsRouteServicePutBridgeSecretProcedure,
		"/bugnet.router.v1.SecretsRouteService/PutBridgeSecret")
	eq(t, routerv1connect.SshGatewayServiceOpenSshUpstreamProcedure,
		"/bugnet.router.v1.SshGatewayService/OpenSshUpstream")
	eq(t, routerv1connect.EgressServiceResolveTunnelProcedure,
		"/bugnet.router.v1.EgressService/ResolveTunnel")
	eq(t, routerv1connect.OperatorServiceRevokePreviewProcedure,
		"/bugnet.router.v1.OperatorService/RevokePreview")
	eq(t, routerv1connect.PaymentsServiceCheckPayerProcedure,
		"/bugnet.router.v1.PaymentsService/CheckPayer")
	eq(t, routerv1connect.PaymentsServiceGetOrderProcedure,
		"/bugnet.router.v1.PaymentsService/GetOrder")
	eq(t, routerv1connect.SandboxRouterServiceFetchSourceProcedure,
		"/bugnet.router.v1.SandboxRouterService/FetchSource")
	eq(t, routerv1connect.SandboxRouterServicePricePackageProcedure,
		"/bugnet.router.v1.SandboxRouterService/PricePackage")
}

// service finds a service of the package by its short name
func service(t *testing.T, name string) protoreflect.ServiceDescriptor {
	t.Helper()
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(
		protoreflect.FullName("bugnet.router.v1." + name))
	if err != nil {
		t.Fatalf("find %s: %v", name, err)
	}
	return d.(protoreflect.ServiceDescriptor)
}

func TestEachServiceLivesInItsFile(t *testing.T) {
	for name, file := range map[string]string{
		"SandboxRouterService": "bugnet/router/v1/router.proto",
		"EdgeService":          "bugnet/router/v1/edge.proto",
		"SecretsRouteService":  "bugnet/router/v1/edge.proto",
		"SshGatewayService":    "bugnet/router/v1/edge.proto",
		"EgressService":        "bugnet/router/v1/edge.proto",
		"OperatorService":      "bugnet/router/v1/operator.proto",
		"PaymentsService":      "bugnet/router/v1/payments.proto",
	} {
		eq(t, service(t, name).ParentFile().Path(), file)
	}
	eq(t, service(t, "SandboxRouterService").Methods().ByName("RevokePreview") == nil, true)
	eq(t, service(t, "OperatorService").Methods().ByName("RevokePreview") != nil, true)
	imports := service(t, "SandboxRouterService").ParentFile().Imports()
	for i := range imports.Len() {
		eq(t, imports.Get(i).Path(), "google/protobuf/timestamp.proto")
	}
}

func TestSpecFieldNumbers(t *testing.T) {
	for name, want := range map[string]protoreflect.FieldNumber{
		"Placement.ssh_alias":                      9,
		"Placement.termination_reason":             10,
		"Placement.bootstrap":                      11,
		"Placement.ended_at":                       13,
		"Requirements.secrets":                     4,
		"CreatePlacementRequest.estimate_id":       3,
		"EstimateRequest.expected_minutes":         5,
		"EstimateRequest.needs":                    6,
		"EstimateResponse.estimate_id":             2,
		"EstimateResponse.partial":                 12,
		"PackageOption.fits":                       7,
		"PackageOption.total_price_micros":         11,
		"JobExit.oom_killed":                       4,
		"JobExit.peak_memory_mib":                  5,
		"ExecResponse.oom_killed":                  8,
		"CreateTransferRequest.size_bytes":         4,
		"CreateTransferRequest.idempotency_key":    7,
		"CreateTransferRequest.want_sha256":        8,
		"CreateTransferResponse.sha256":            6,
		"PricePackageRequest.requirements":         2,
		"PricePackageResponse.option":              1,
		"ResolvePreviewResponse.started_at":        9,
		"OpenTransferResponse.budget":              10,
		"OpenTransferResponse.sha256":              11,
		"ResolveTunnelResponse.requests_in_flight": 9,
		"GetEstimateResponse.catalog":              7,
		"ProviderStatus.foreign_sandboxes":         13,
		"CheckPayerRequest.kind":                   4,
		"GetOrderRequest.order_id":                 1,
		"GetOrderResponse.placement_state":         3,
		"GetOrderResponse.ended_at":                7,
		"OpenSshUpstreamResponse.deadline":         5,
		"EdgeEvent.resync":                         7,
		"StartReport.errors":                       18,
		"SubmitSecretLinkRequest.client_ip":        5,
		"OperatorPlacement.termination_reason":     11,
		"CheckEdgeLeasesRequest.previews":          2,
		"DeliverHookRequest.client_ip":             4,
		"WatchEgressEventsResponse.event":          2,
		"AcquireSshConnectionResponse.budget":      3,
		"CreateSecretLinkRequest.request_id":       4,
		"ListSignalsRequest.held_only":             5,
		"BlockWalletRequest.actor":                 5,
		"EgressSecret.custom_hosts":                4,
		"GateUpstream.blocked_path_prefixes":       4,
		"EdgeBudget.requests_burst":                6,
		"AddSshKeyResponse.expires_at":             8,
		"ResolvedSource.unread_reason":             6,
		"ResourceFloor.context_tokens":             5,
	} {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName(
			protoreflect.FullName("bugnet.router.v1." + name))
		if err != nil {
			t.Fatalf("find %s: %v", name, err)
		}
		if got := d.(protoreflect.FieldDescriptor).Number(); got != want {
			t.Errorf("%s is field %d, want %d", name, got, want)
		}
	}
	for value, want := range map[routerv1.EndReason]string{
		6: "END_REASON_CONTROL_LOST", 7: "END_REASON_OPERATOR", 8: "END_REASON_SANDBOX_EXITED",
	} {
		eq(t, value.String(), want)
	}
	eq(t, routerv1.WarningCode(13).String(), "WARNING_CODE_RAN_OUT_OF_MEMORY")
	eq(t, routerv1.BootstrapState(6).String(), "BOOTSTRAP_STATE_NEEDS_SECRET")
	eq(t, routerv1.PayerVerdict(4).String(), "PAYER_VERDICT_RATE_LIMITED")
}

func TestX402FieldNumbers(t *testing.T) {
	checkWire(t, "bugnet.router.v1", map[string]wire{
		"ResolveTunnelResponse.x402":                 {11, "X402Config"},
		"ReportEgressUsageRequest.x402_unpaid":       {3, "map<string, X402UnpaidByHost>"},
		"AppendAllowanceEventRequest.order_id":       {1, "string"},
		"AppendAllowanceEventRequest.event":          {2, "AllowanceEvent"},
		"AppendAllowanceEventRequest.start_order_id": {3, "string"},
		"X402Event.service_id":                       {22, "string"},
		"X402Event.payment_ref":                      {23, "string"},
		"EnableX402Request.wait_active":              {5, "bool"},
		"AllowanceEvent.phase":                       {1, "string"},
		"AllowanceEvent.amount_micros":               {2, "int64"},
		"AllowanceEvent.asset":                       {3, "string"},
		"AllowanceEvent.network":                     {4, "string"},
		"AllowanceEvent.tx":                          {5, "string"},
		"AllowanceEvent.kept_micros":                 {6, "int64"},
		"X402ServiceSpec.host":                       {1, "string"},
		"X402ServiceSpec.prefix":                     {2, "string"},
		"X402ServiceSpec.method":                     {3, "string"},
		"X402ServiceSpec.max_price_micros":           {4, "int64"},
		"X402ServiceSpec.pay_to":                     {5, "string"},
		"X402ServiceSpec.indexed":                    {6, "bool"},
		"X402ServiceSpec.paid_path":                  {7, "string"},
		"X402ServiceInfo.service_id":                 {1, "string"},
		"X402ServiceInfo.spec":                       {2, "X402ServiceSpec"},
		"X402ServiceInfo.state":                      {3, "X402ServiceState"},
		"X402ServiceInfo.failure_reason":             {4, "string"},
		"X402ServiceInfo.requests":                   {5, "uint32"},
		"X402ServiceInfo.paid":                       {6, "uint32"},
		"X402ServiceInfo.uncertain":                  {7, "uint32"},
		"X402Refusal.reason":                         {1, "string"},
		"X402Refusal.count":                          {2, "uint32"},
		"X402Refusal.last_at":                        {3, "google.protobuf.Timestamp"},
		"CheckX402Request.handle":                    {1, "string"},
		"CheckX402Request.add":                       {2, "repeated X402ServiceSpec"},
		"CheckX402Request.disable":                   {3, "repeated string"},
		"EnableX402Request.handle":                   {1, "string"},
		"EnableX402Request.add":                      {2, "repeated X402ServiceSpec"},
		"EnableX402Request.disable":                  {3, "repeated string"},
		"EnableX402Request.request_id":               {4, "string"},
		"EnableX402Response.services":                {1, "repeated X402ServiceInfo"},
		"ListX402Request.handle":                     {1, "string"},
		"ListX402Response.enabled":                   {1, "bool"},
		"ListX402Response.services":                  {2, "repeated X402ServiceInfo"},
		"ListX402Response.refusals":                  {3, "repeated X402Refusal"},
		"X402Service.service_id":                     {1, "string"},
		"X402Service.host":                           {2, "string"},
		"X402Service.prefix":                         {3, "string"},
		"X402Service.method":                         {4, "string"},
		"X402Service.max_price_micros":               {5, "int64"},
		"X402Service.pay_to":                         {6, "string"},
		"X402Service.paid_path":                      {7, "string"},
		"X402Unpaid.requests":                        {1, "uint32"},
		"X402Unpaid.bytes":                           {2, "uint64"},
		"X402Config.enabled":                         {1, "bool"},
		"X402Config.account_id":                      {2, "string"},
		"X402Config.services":                        {3, "repeated X402Service"},
		"X402Config.unpaid_requests_per_host":        {4, "uint32"},
		"X402Config.unpaid_bytes_per_host":           {5, "uint64"},
		"X402Config.unpaid_used":                     {6, "map<string, X402Unpaid>"},
		"X402UnpaidByHost.hosts":                     {1, "map<string, X402Unpaid>"},
		"X402Event.at":                               {1, "google.protobuf.Timestamp"},
		"X402Event.ref":                              {2, "string"},
		"X402Event.host":                             {3, "string"},
		"X402Event.method":                           {4, "string"},
		"X402Event.path":                             {5, "string"},
		"X402Event.status":                           {6, "string"},
		"X402Event.reason":                           {7, "string"},
		"X402Event.asked_micros":                     {8, "int64"},
		"X402Event.amount_micros":                    {9, "int64"},
		"X402Event.asset":                            {10, "string"},
		"X402Event.network":                          {11, "string"},
		"X402Event.scheme":                           {12, "string"},
		"X402Event.x402_version":                     {13, "uint32"},
		"X402Event.pay_to":                           {14, "string"},
		"X402Event.http_status":                      {15, "uint32"},
		"X402Event.duration_ms":                      {16, "uint32"},
		"X402Event.bytes_out":                        {17, "uint64"},
		"X402Event.bytes_in":                         {18, "uint64"},
		"X402Event.stream_cut":                       {19, "bool"},
		"X402Event.allowance_left_micros":            {20, "int64"},
		"X402Event.count":                            {21, "uint32"},
		"AppendX402EventsRequest.placement_id":       {1, "string"},
		"AppendX402EventsRequest.events":             {2, "repeated X402Event"},
		"AppendX402EventsResponse.rejected":          {1, "repeated uint32"},
		"LookupPlacementRequest.handle_sha256":       {1, "string"},
		"LookupPlacementRequest.start_order_id":      {2, "string"},
		"LookupPlacementResponse.placement_id":       {1, "string"},
		"LookupPlacementResponse.start_order_id":     {2, "string"},
		"LookupPlacementResponse.state":              {3, "string"},
		"LookupPlacementResponse.deadline":           {4, "google.protobuf.Timestamp"},
		"LookupPlacementResponse.ended_at":           {5, "google.protobuf.Timestamp"},
		"LookupPlacementResponse.x402_capable":       {6, "bool"},
		"LookupPlacementResponse.package_id":         {7, "string"},
		"AppendX402ResolvedRequest.start_order_id":   {1, "string"},
		"AppendX402ResolvedRequest.ref":              {2, "string"},
		"AppendX402ResolvedRequest.status":           {3, "string"},
		"AppendX402ResolvedRequest.amount_micros":    {4, "int64"},
		"AppendX402ResolvedRequest.tx":               {5, "string"},
		"AppendX402ResolvedRequest.at":               {6, "google.protobuf.Timestamp"},
		"AppendX402ResolvedRequest.host":             {7, "string"},
		"AppendX402ResolvedRequest.method":           {8, "string"},
		"AppendX402ResolvedRequest.path":             {9, "string"},
		"AppendX402ResolvedRequest.pay_to":           {10, "string"},
	})
	for msg, n := range map[proto.Message]int{
		&routerv1.X402ServiceSpec{}: 7, &routerv1.X402ServiceInfo{}: 7,
		&routerv1.X402Refusal{}: 3, &routerv1.CheckX402Request{}: 3,
		&routerv1.CheckX402Response{}: 0, &routerv1.EnableX402Request{}: 5,
		&routerv1.EnableX402Response{}: 1, &routerv1.ListX402Request{}: 1,
		&routerv1.ListX402Response{}: 3, &routerv1.X402Service{}: 7, &routerv1.X402Unpaid{}: 2,
		&routerv1.X402Config{}: 6, &routerv1.X402UnpaidByHost{}: 1, &routerv1.X402Event{}: 23,
		&routerv1.AppendX402EventsRequest{}: 2, &routerv1.AppendX402EventsResponse{}: 1,
		&routerv1.LookupPlacementRequest{}: 2, &routerv1.LookupPlacementResponse{}: 7,
		&routerv1.AllowanceEvent{}: 6, &routerv1.AppendAllowanceEventRequest{}: 3,
		&routerv1.AppendAllowanceEventResponse{}: 0, &routerv1.AppendX402ResolvedRequest{}: 10,
		&routerv1.AppendX402ResolvedResponse{}: 0,
	} {
		if md := msg.ProtoReflect().Descriptor(); md.Fields().Len() != n {
			t.Errorf("%s has %d fields, want %d", md.Name(), md.Fields().Len(), n)
		}
	}
}

func TestResolveTunnelKeepsTenAndHundredForTheirTasks(t *testing.T) {
	fields := (&routerv1.ResolveTunnelResponse{}).ProtoReflect().Descriptor().Fields()
	ten, hundred := fields.ByNumber(10), fields.ByNumber(100)
	eq(t, ten == nil || ten.Name() == "source_check", true)
	eq(t, hundred == nil || hundred.Name() == "order_traceparent", true)
}

// x402At is a fixed time the x402 round trips carry
var x402At = timestamppb.New(time.Date(2026, 11, 7, 18, 2, 11, 204e6, time.UTC))

func TestX402MessagesRoundTrip(t *testing.T) {
	spec := &routerv1.X402ServiceSpec{
		Host: "api.example.com", Prefix: "/search/", Method: "GET", MaxPriceMicros: 10000,
		PayTo: "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin", PaidPath: "/search",
	}
	cfg := &routerv1.X402Config{
		Enabled: true, AccountId: "order-1",
		Services: []*routerv1.X402Service{{
			ServiceId: "xs_vky43ltmauaj3w53", Host: "api.example.com", Prefix: "/search/",
			Method: "GET", MaxPriceMicros: 10000, PayTo: spec.GetPayTo(), PaidPath: "/search",
		}},
		UnpaidRequestsPerHost: 300, UnpaidBytesPerHost: 64 << 20,
		UnpaidUsed: map[string]*routerv1.X402Unpaid{
			"api.example.com": {Requests: 3, Bytes: 4096},
		},
	}
	msgs := []proto.Message{
		cfg,
		&routerv1.X402Event{
			At: x402At, Ref: "k3m7q2x7a4b6c5d2", Host: "agent402.tools", Method: "POST",
			Path: "/v1/auto/chat/completions", Status: "paid", Reason: "",
			AskedMicros: 10000, AmountMicros: 10000,
			Asset:   "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
			Network: "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp", Scheme: "exact",
			X402Version: 2, PayTo: spec.GetPayTo(), HttpStatus: 200, DurationMs: 1840,
			BytesOut: 512, BytesIn: 2048, StreamCut: true, AllowanceLeftMicros: 990000,
			Count: 1, ServiceId: "xs_w5a4htopmpzmmhos", PaymentRef: "k3m7q2x7a4b6c5d2",
		},
		&routerv1.X402ServiceInfo{
			ServiceId: "xs_vky43ltmauaj3w53", Spec: spec,
			State:         routerv1.X402ServiceState_X402_SERVICE_STATE_FAILED,
			FailureReason: "HOST_BLOCKED", Requests: 7, Paid: 3, Uncertain: 1,
		},
		&routerv1.LookupPlacementResponse{
			PlacementId: "p-1", StartOrderId: "order-1", State: "ready", Deadline: x402At,
			EndedAt: x402At, X402Capable: true, PackageId: "small-10m",
		},
		&routerv1.AppendX402ResolvedRequest{
			StartOrderId: "order-1", Ref: "k3m7q2x7a4b6c5d2", Status: "paid",
			AmountMicros: 10000, Tx: "5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnbJLgp8uirBgmQpj" +
				"KhoR4tjF3ZpRzrFmBV6UjKdiSZkQUW",
			At: x402At, Host: "agent402.tools", Method: "POST", Path: "/v1/auto/chat/completions",
			PayTo: spec.GetPayTo(),
		},
		&routerv1.ResolveTunnelResponse{PlacementId: "p-1", X402: cfg},
		&routerv1.ReportEgressUsageRequest{
			ReporterId: "egress-1",
			X402Unpaid: map[string]*routerv1.X402UnpaidByHost{"p-1": {
				Hosts: map[string]*routerv1.X402Unpaid{"api.example.com": {Requests: 1, Bytes: 9}},
			}},
		},
		&routerv1.CheckX402Request{
			Handle: "sbx_a", Add: []*routerv1.X402ServiceSpec{spec}, Disable: []string{"xs_a"},
		},
		&routerv1.EnableX402Request{
			Handle: "sbx_a", Add: []*routerv1.X402ServiceSpec{spec}, Disable: []string{"xs_a"},
			RequestId: "req-1", WaitActive: true,
		},
		&routerv1.ListX402Response{
			Enabled: true,
			Services: []*routerv1.X402ServiceInfo{{ServiceId: "xs_a", Spec: spec,
				State: routerv1.X402ServiceState_X402_SERVICE_STATE_ACTIVE}},
			Refusals: []*routerv1.X402Refusal{{Reason: "refused_host", Count: 4, LastAt: x402At}},
		},
		&routerv1.AppendX402EventsRequest{
			PlacementId: "p-1", Events: []*routerv1.X402Event{{Ref: "a", Count: 2}},
		},
		&routerv1.AppendX402EventsResponse{Rejected: []uint32{0, 3}},
		&routerv1.AppendAllowanceEventRequest{
			OrderId: "order-1", StartOrderId: "order-1",
			Event: &routerv1.AllowanceEvent{Phase: "kept", KeptMicros: 4000},
		},
	}
	for _, in := range msgs {
		if out := roundTrip(t, in); !proto.Equal(in, out) {
			t.Fatalf("round trip mismatch: %v != %v", in, out)
		}
	}
}

func TestX402Procedures(t *testing.T) {
	eq(t, routerv1connect.SandboxRouterServiceCheckX402Procedure,
		"/bugnet.router.v1.SandboxRouterService/CheckX402")
	eq(t, routerv1connect.SandboxRouterServiceEnableX402Procedure,
		"/bugnet.router.v1.SandboxRouterService/EnableX402")
	eq(t, routerv1connect.SandboxRouterServiceListX402Procedure,
		"/bugnet.router.v1.SandboxRouterService/ListX402")
	eq(t, routerv1connect.EgressServiceAppendX402EventsProcedure,
		"/bugnet.router.v1.EgressService/AppendX402Events")
	eq(t, routerv1connect.PaymentsServiceLookupPlacementProcedure,
		"/bugnet.router.v1.PaymentsService/LookupPlacement")
	eq(t, routerv1connect.PaymentsServiceAppendAllowanceEventProcedure,
		"/bugnet.router.v1.PaymentsService/AppendAllowanceEvent")
	eq(t, routerv1connect.PaymentsServiceAppendX402ResolvedProcedure,
		"/bugnet.router.v1.PaymentsService/AppendX402Resolved")
}

func TestX402ServiceStateValues(t *testing.T) {
	for value, want := range map[routerv1.X402ServiceState]string{
		0: "X402_SERVICE_STATE_UNSPECIFIED", 1: "X402_SERVICE_STATE_APPLYING",
		2: "X402_SERVICE_STATE_ACTIVE", 3: "X402_SERVICE_STATE_FAILED",
		4: "X402_SERVICE_STATE_DISABLED",
	} {
		eq(t, value.String(), want)
	}
	eq(t, routerv1.X402ServiceState_X402_SERVICE_STATE_DISABLED.Descriptor().Values().Len(), 5)
}

func TestLookupPlacementKeyIsOneOf(t *testing.T) {
	m := (&routerv1.LookupPlacementRequest{}).ProtoReflect()
	fields := m.Descriptor().Fields()
	byHandle, byOrder := fields.ByName("handle_sha256"), fields.ByName("start_order_id")
	eq(t, byHandle.ContainingOneof().Name(), protoreflect.Name("key"))
	eq(t, byOrder.ContainingOneof() == byHandle.ContainingOneof(), true)
	m.Set(byHandle, protoreflect.ValueOfString("ab"))
	m.Set(byOrder, protoreflect.ValueOfString("order-1"))
	eq(t, m.Has(byHandle), false)
	req := m.Interface().(*routerv1.LookupPlacementRequest)
	eq(t, req.GetStartOrderId(), "order-1")
	eq(t, req.GetHandleSha256(), "")
	eq(t, roundTrip(t, req).GetStartOrderId(), "order-1")
}

func TestX402RPCsLiveInTheirServices(t *testing.T) {
	for svc, names := range map[string][]string{
		"SandboxRouterService": {"CheckX402", "EnableX402", "ListX402"},
		"EgressService":        {"AppendX402Events"},
		"PaymentsService":      {"LookupPlacement", "AppendAllowanceEvent", "AppendX402Resolved"},
	} {
		for _, name := range names {
			eq(t, service(t, svc).Methods().ByName(protoreflect.Name(name)) != nil, true)
		}
	}
}
