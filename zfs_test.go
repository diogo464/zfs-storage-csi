package main

import "testing"

func TestParseQuota(t *testing.T) {
	var v *uint64
	var err error

	v, err = parseQuota("20G")
	if err != nil || *v != 20*1024*1024*1024 {
		t.Errorf("failed to parse 20G %v", err)
	}

	v, err = parseQuota("1.5T")
	if err != nil || *v != uint64(1.5*1024*1024*1024*1024) {
		t.Errorf("failed to parse 1.5T %v", err)
	}

	v, err = parseQuota("none")
	if err != nil || v != nil {
		t.Errorf("failed to parse none %v", err)
	}

	v, err = parseQuota("-")
	if err != nil || v != nil {
		t.Errorf("failed to parse - %v", err)
	}
}
