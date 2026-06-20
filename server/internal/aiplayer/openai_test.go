package aiplayer

import (
	"encoding/json"
	"testing"
)

func TestChooseActionArgumentsAcceptsFlatNoteFields(t *testing.T) {
	var args chooseActionArguments
	if err := json.Unmarshal([]byte(`{"actionId":"vote:seat_2","notePlayerId":"seat_2","noteText":"发言摇摆"}`), &args); err != nil {
		t.Fatalf("unmarshal args: %v", err)
	}
	if args.ActionID != "vote:seat_2" || args.Notes["seat_2"] != "发言摇摆" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestChooseActionArgumentsAcceptsObjectNotes(t *testing.T) {
	var args chooseActionArguments
	if err := json.Unmarshal([]byte(`{"actionId":"vote:p1","notes":{"p1":"可疑"}}`), &args); err != nil {
		t.Fatalf("unmarshal args: %v", err)
	}
	if args.ActionID != "vote:p1" || args.Notes["p1"] != "可疑" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestChooseActionArgumentsAcceptsStringifiedNotes(t *testing.T) {
	var args chooseActionArguments
	if err := json.Unmarshal([]byte(`{"actionId":"vote:p1","notes":"{\"p1\":\"可疑\"}"}`), &args); err != nil {
		t.Fatalf("unmarshal args: %v", err)
	}
	if args.ActionID != "vote:p1" || args.Notes["p1"] != "可疑" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestChooseActionArgumentsIgnoresInvalidNotes(t *testing.T) {
	var args chooseActionArguments
	if err := json.Unmarshal([]byte(`{"actionId":"vote:p1","notes":"not-json"}`), &args); err != nil {
		t.Fatalf("invalid notes should not reject action: %v", err)
	}
	if args.ActionID != "vote:p1" || args.Notes != nil {
		t.Fatalf("unexpected args: %+v", args)
	}
}
