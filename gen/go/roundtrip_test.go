package gen_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/timestamppb"

	routerv1 "github.com/oynozan/bugnet-proto/gen/go/bugnet/router/v1"
	"github.com/oynozan/bugnet-proto/gen/go/bugnet/router/v1/routerv1connect"
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

func TestPlacementEndedAtRoundTripsAndLeavesTwelveFree(t *testing.T) {
	ended := timestamppb.New(time.Date(2026, 11, 6, 1, 2, 3, 4000, time.UTC))
	eq(t, proto.Equal(roundTrip(t, &routerv1.Placement{EndedAt: ended}).EndedAt, ended), true)
	eq(t, roundTrip(t, &routerv1.Placement{}).EndedAt == nil, true)
	md := (&routerv1.Placement{}).ProtoReflect().Descriptor()
	eq(t, md.Fields().ByNumber(12) == nil, true)
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
		"ResolveTunnelResponse.requests_in_flight": 9,
		"GetEstimateResponse.catalog":              7,
		"ProviderStatus.foreign_sandboxes":         13,
		"CheckPayerRequest.kind":                   4,
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
