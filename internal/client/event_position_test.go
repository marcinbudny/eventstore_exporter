package client

import "testing"

func Test_Parse_Event_Position(t *testing.T) {
	commit, prepare, err := EventPosition("C:1234/P:5678").ParseCommitPreparePosition()
	if err != nil {
		t.Error(err)
	}

	if commit != 1234 {
		t.Errorf("Expected commit position to be 1234, got %d", commit)
	}
	if prepare != 5678 {
		t.Errorf("Expected prepare position to be 5678, got %d", prepare)
	}
}

func Test_Parse_Invalid_Prepare_Position(t *testing.T) {
	_, _, err := EventPosition("1234").ParseCommitPreparePosition()
	if err == nil {
		t.Error("Expected error")
	}
}

func Test_Parse_Empty_Prepare_Position(t *testing.T) {
	_, _, err := EventPosition("").ParseCommitPreparePosition()
	if err == nil {
		t.Error("Expected error")
	}
}
