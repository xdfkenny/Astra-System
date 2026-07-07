package domain

import (
	"testing"

	paymentv1 "github.com/astra-systems/astra-service/proto/gen/go/payment"
)

func TestPayment_Transitions(t *testing.T) {
	cases := []struct {
		from   PaymentStatus
		to     PaymentStatus
		wantOK bool
	}{
		{StatusPending, StatusAuthorizing, true},
		{StatusPending, StatusFailed, true},
		{StatusPending, StatusCaptured, false},
		{StatusPending, StatusSettled, false},
		{StatusAuthorizing, StatusCaptured, true},
		{StatusAuthorizing, StatusFailed, true},
		{StatusAuthorizing, StatusSettled, false},
		{StatusCaptured, StatusSettled, true},
		{StatusCaptured, StatusFailed, true},
		{StatusSettled, StatusFailed, false},
		{StatusSettled, StatusPending, false},
		{StatusFailed, StatusPending, false},
		{StatusFailed, StatusCaptured, false},
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"_"+string(tc.to), func(t *testing.T) {
			p := &Payment{Status: tc.from}
			err := p.Transition(tc.to)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("expected transition to succeed, got %v", err)
				}
				if p.Status != tc.to {
					t.Fatalf("expected status %s, got %s", tc.to, p.Status)
				}
				if p.UpdatedAt.IsZero() {
					t.Fatal("expected UpdatedAt to be set")
				}
			} else {
				if err != ErrInvalidTransition {
					t.Fatalf("expected ErrInvalidTransition, got %v", err)
				}
			}
		})
	}
}

func TestPayment_IsTerminal(t *testing.T) {
	terminal := []PaymentStatus{StatusSettled, StatusFailed}
	for _, s := range terminal {
		p := &Payment{Status: s}
		if !p.IsTerminal() {
			t.Fatalf("status %s should be terminal", s)
		}
	}
	nonTerminal := []PaymentStatus{StatusPending, StatusAuthorizing, StatusCaptured}
	for _, s := range nonTerminal {
		p := &Payment{Status: s}
		if p.IsTerminal() {
			t.Fatalf("status %s should not be terminal", s)
		}
	}
}

func TestPaymentStatusFromProto(t *testing.T) {
	cases := []struct {
		proto paymentv1.PaymentStatus
		want  PaymentStatus
	}{
		{paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING, StatusPending},
		{paymentv1.PaymentStatus_PAYMENT_STATUS_AUTHORIZED, StatusAuthorizing},
		{paymentv1.PaymentStatus_PAYMENT_STATUS_CAPTURED, StatusCaptured},
		{paymentv1.PaymentStatus_PAYMENT_STATUS_DECLINED, StatusFailed},
	}
	for _, tc := range cases {
		got := PaymentStatusFromProto(tc.proto)
		if got != tc.want {
			t.Fatalf("proto %v: want %s, got %s", tc.proto, tc.want, got)
		}
	}
}
