package gen_test

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	paymentsv1 "github.com/oynozan/bugnet-proto/gen/go/bugnet/payments/v1"
	"github.com/oynozan/bugnet-proto/gen/go/bugnet/payments/v1/paymentsv1connect"
)

func TestSignerFieldNumbers(t *testing.T) {
	for name, want := range map[string]protoreflect.FieldNumber{
		"ReserveAndSignRequest.account_id":            1,
		"ReserveAndSignRequest.placement_id":          2,
		"ReserveAndSignRequest.ref":                   3,
		"ReserveAndSignRequest.service_id":            4,
		"ReserveAndSignRequest.method":                5,
		"ReserveAndSignRequest.host":                  6,
		"ReserveAndSignRequest.path":                  7,
		"ReserveAndSignRequest.body_sha256":           8,
		"ReserveAndSignRequest.payment_required":      9,
		"ReserveAndSignRequest.target_sha256":         10,
		"Signed.payment_id":                           1,
		"Signed.payment_signature":                    2,
		"Signed.amount_micros":                        3,
		"Signed.pay_to":                               4,
		"Signed.keep_until":                           5,
		"Signed.allowance_left_micros":                6,
		"Signed.resend":                               7,
		"Signed.payment_ref":                          8,
		"Refused.reason":                              1,
		"Refused.asked_micros":                        2,
		"Refused.max_micros":                          3,
		"Refused.allowance_left_micros":               4,
		"ReserveAndSignResponse.signed":               1,
		"ReserveAndSignResponse.refused":              2,
		"ReportOutcomeRequest.payment_id":             1,
		"ReportOutcomeRequest.ref":                    2,
		"ReportOutcomeRequest.http_status":            3,
		"ReportOutcomeRequest.hint":                   4,
		"ReportOutcomeRequest.hint_tx":                5,
		"ReportOutcomeRequest.hint_error":             6,
		"ReportOutcomeRequest.stream_cut":             7,
		"ReportOutcomeRequest.resend":                 8,
		"ReportOutcomeResponse.allowance_left_micros": 1,
	} {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName(
			protoreflect.FullName("bugnet.payments.v1." + name))
		if err != nil {
			t.Fatalf("find %s: %v", name, err)
		}
		if got := d.(protoreflect.FieldDescriptor).Number(); got != want {
			t.Errorf("%s is field %d, want %d", name, got, want)
		}
	}
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
