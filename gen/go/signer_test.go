package gen_test

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	paymentsv1 "github.com/bugnet-sh/bugnet-proto/gen/go/bugnet/payments/v1"
	"github.com/bugnet-sh/bugnet-proto/gen/go/bugnet/payments/v1/paymentsv1connect"
)

func TestSignerFieldNumbers(t *testing.T) {
	checkWire(t, "bugnet.payments.v1", map[string]wire{
		"ReserveAndSignRequest.account_id":            {1, "string"},
		"ReserveAndSignRequest.placement_id":          {2, "string"},
		"ReserveAndSignRequest.ref":                   {3, "string"},
		"ReserveAndSignRequest.service_id":            {4, "string"},
		"ReserveAndSignRequest.method":                {5, "string"},
		"ReserveAndSignRequest.host":                  {6, "string"},
		"ReserveAndSignRequest.path":                  {7, "string"},
		"ReserveAndSignRequest.body_sha256":           {8, "bytes"},
		"ReserveAndSignRequest.payment_required":      {9, "string"},
		"ReserveAndSignRequest.target_sha256":         {10, "bytes"},
		"Signed.payment_id":                           {1, "string"},
		"Signed.payment_signature":                    {2, "string"},
		"Signed.amount_micros":                        {3, "int64"},
		"Signed.pay_to":                               {4, "string"},
		"Signed.keep_until":                           {5, "google.protobuf.Timestamp"},
		"Signed.allowance_left_micros":                {6, "int64"},
		"Signed.resend":                               {7, "bool"},
		"Signed.payment_ref":                          {8, "string"},
		"Refused.reason":                              {1, "string"},
		"Refused.asked_micros":                        {2, "int64"},
		"Refused.max_micros":                          {3, "int64"},
		"Refused.allowance_left_micros":               {4, "int64"},
		"ReserveAndSignResponse.signed":               {1, "Signed"},
		"ReserveAndSignResponse.refused":              {2, "Refused"},
		"ReportOutcomeRequest.payment_id":             {1, "string"},
		"ReportOutcomeRequest.ref":                    {2, "string"},
		"ReportOutcomeRequest.http_status":            {3, "uint32"},
		"ReportOutcomeRequest.hint":                   {4, "SellerHint"},
		"ReportOutcomeRequest.hint_tx":                {5, "string"},
		"ReportOutcomeRequest.hint_error":             {6, "string"},
		"ReportOutcomeRequest.stream_cut":             {7, "bool"},
		"ReportOutcomeRequest.resend":                 {8, "bool"},
		"ReportOutcomeResponse.allowance_left_micros": {1, "int64"},
	})
	for msg, n := range map[proto.Message]int{
		&paymentsv1.ReserveAndSignRequest{}: 10, &paymentsv1.Signed{}: 8,
		&paymentsv1.Refused{}: 4, &paymentsv1.ReserveAndSignResponse{}: 2,
		&paymentsv1.ReportOutcomeRequest{}: 8, &paymentsv1.ReportOutcomeResponse{}: 1,
	} {
		if md := msg.ProtoReflect().Descriptor(); md.Fields().Len() != n {
			t.Errorf("%s has %d fields, want %d", md.Name(), md.Fields().Len(), n)
		}
	}
}

func TestSignerHintValues(t *testing.T) {
	for value, want := range map[paymentsv1.SellerHint]string{
		0: "SELLER_HINT_UNSPECIFIED", 1: "SELLER_HINT_SUCCESS", 2: "SELLER_HINT_FAILURE",
		3: "SELLER_HINT_ABSENT", 4: "SELLER_HINT_REJECTED",
	} {
		eq(t, value.String(), want)
	}
	eq(t, paymentsv1.SellerHint_SELLER_HINT_REJECTED.Descriptor().Values().Len(), 5)
}

func TestSignerMessagesRoundTrip(t *testing.T) {
	signed := &paymentsv1.Signed{
		PaymentId: "pay-1", PaymentSignature: "eyJ4NDAyVmVyc2lvbiI6Mn0=", AmountMicros: 10000,
		PayTo: "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin", KeepUntil: x402At,
		AllowanceLeftMicros: 990000, Resend: true, PaymentRef: "k3m7q2x7a4b6c5d2",
	}
	msgs := []proto.Message{
		&paymentsv1.ReserveAndSignRequest{
			AccountId: "order-1", PlacementId: "p-1", Ref: "a4b6c5d2k3m7q2x7",
			ServiceId: "xs_w5a4htopmpzmmhos", Method: "POST", Host: "agent402.tools",
			Path: "/v1/auto/chat/completions", BodySha256: make([]byte, 32),
			PaymentRequired: "eyJ4NDAyVmVyc2lvbiI6Mn0=", TargetSha256: []byte{1, 2, 3},
		},
		&paymentsv1.ReserveAndSignResponse{
			Result: &paymentsv1.ReserveAndSignResponse_Signed{Signed: signed},
		},
		&paymentsv1.ReserveAndSignResponse{
			Result: &paymentsv1.ReserveAndSignResponse_Refused{Refused: &paymentsv1.Refused{
				Reason: "price_above_max", AskedMicros: 600000, MaxMicros: 500000,
				AllowanceLeftMicros: 990000,
			}},
		},
		&paymentsv1.ReportOutcomeRequest{
			PaymentId: "pay-1", Ref: "k3m7q2x7a4b6c5d2", HttpStatus: 200,
			Hint: paymentsv1.SellerHint_SELLER_HINT_SUCCESS, HintTx: "5VERv8NMvzbJMEkV8xnr",
			HintError: "insufficient_funds", StreamCut: true, Resend: true,
		},
		&paymentsv1.ReportOutcomeResponse{AllowanceLeftMicros: 980000},
	}
	for _, in := range msgs {
		if out := roundTrip(t, in); !proto.Equal(in, out) {
			t.Fatalf("round trip mismatch: %v != %v", in, out)
		}
	}
}

func TestSignerProceduresAndFile(t *testing.T) {
	eq(t, paymentsv1connect.X402SignerServiceReserveAndSignProcedure,
		"/bugnet.payments.v1.X402SignerService/ReserveAndSign")
	eq(t, paymentsv1connect.X402SignerServiceReportOutcomeProcedure,
		"/bugnet.payments.v1.X402SignerService/ReportOutcome")
	d, err := protoregistry.GlobalFiles.FindDescriptorByName("bugnet.payments.v1.X402SignerService")
	if err != nil {
		t.Fatal(err)
	}
	svc := d.(protoreflect.ServiceDescriptor)
	eq(t, svc.ParentFile().Path(), "bugnet/payments/v1/x402.proto")
	eq(t, svc.Methods().Len(), 2)
	imports := svc.ParentFile().Imports()
	for i := range imports.Len() {
		eq(t, imports.Get(i).Path(), "google/protobuf/timestamp.proto")
	}
}
