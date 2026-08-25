package test

import (
	"errors"
	"testing"

	"github.com/expressjs/cors/internal/vary"
)

type recordingResponse struct {
	headers map[string]string
}

func newRecordingResponse() *recordingResponse {
	return &recordingResponse{headers: map[string]string{}}
}

func (r *recordingResponse) GetHeader(name string) string { return r.headers[name] }

func (r *recordingResponse) SetHeader(name, value string) { r.headers[name] = value }

func TestVaryAppendSetsTheFirstFieldName(t *testing.T) {
	got, err := vary.Append("", "Origin")

	if err != nil {
		t.Fatalf("Append err = %v, want nil", err)
	}
	if got != "Origin" {
		t.Errorf("Append = %q, want %q", got, "Origin")
	}
}

func TestVaryAppendPreservesCaseAndSkipsDuplicates(t *testing.T) {
	got, err := vary.Append("Origin", "origin")

	if err != nil {
		t.Fatalf("Append err = %v, want nil", err)
	}
	if got != "Origin" {
		t.Errorf("Append = %q, want %q", got, "Origin")
	}
}

func TestVaryAppendSeparatesFieldNamesWithACommaAndSpace(t *testing.T) {
	got, err := vary.Append("Accept,  Accept-Encoding", "Origin")

	if err != nil {
		t.Fatalf("Append err = %v, want nil", err)
	}
	if got != "Accept,  Accept-Encoding, Origin" {
		t.Errorf("Append = %q, want %q", got, "Accept,  Accept-Encoding, Origin")
	}
}

func TestVaryAppendKeepsAnExistingWildcard(t *testing.T) {
	got, err := vary.Append("*", "Origin")

	if err != nil {
		t.Fatalf("Append err = %v, want nil", err)
	}
	if got != "*" {
		t.Errorf("Append = %q, want %q", got, "*")
	}
}

func TestVaryAppendCollapsesToAWildcard(t *testing.T) {
	fromField, err := vary.Append("Accept", "*")
	if err != nil {
		t.Fatalf("Append err = %v, want nil", err)
	}
	if fromField != "*" {
		t.Errorf("Append = %q, want %q", fromField, "*")
	}

	fromHeader, err := vary.Append("Accept, *", "Origin")
	if err != nil {
		t.Fatalf("Append err = %v, want nil", err)
	}
	if fromHeader != "*" {
		t.Errorf("Append = %q, want %q", fromHeader, "*")
	}
}

func TestVaryAppendRejectsAnEmptyField(t *testing.T) {
	_, err := vary.Append("Accept", "")

	if !errors.Is(err, vary.ErrFieldRequired) {
		t.Errorf("Append err = %v, want %v", err, vary.ErrFieldRequired)
	}
}

func TestVaryAppendRejectsAnInvalidFieldName(t *testing.T) {
	_, err := vary.Append("", "Origin\nInjected")

	if !errors.Is(err, vary.ErrInvalidFieldName) {
		t.Errorf("Append err = %v, want %v", err, vary.ErrInvalidFieldName)
	}
}

func TestVaryAppendFieldsAppendsEveryFieldName(t *testing.T) {
	got, err := vary.AppendFields("", []string{"Accept", "Origin", "Accept"})

	if err != nil {
		t.Fatalf("AppendFields err = %v, want nil", err)
	}
	if got != "Accept, Origin" {
		t.Errorf("AppendFields = %q, want %q", got, "Accept, Origin")
	}
}

func TestVaryAppendFieldsRejectsANilSlice(t *testing.T) {
	_, err := vary.AppendFields("Accept", nil)

	if !errors.Is(err, vary.ErrFieldRequired) {
		t.Errorf("AppendFields err = %v, want %v", err, vary.ErrFieldRequired)
	}
}

func TestVaryAppendFieldsAcceptsAnEmptySlice(t *testing.T) {
	got, err := vary.AppendFields("Accept", []string{})

	if err != nil {
		t.Fatalf("AppendFields err = %v, want nil", err)
	}
	if got != "Accept" {
		t.Errorf("AppendFields = %q, want %q", got, "Accept")
	}
}

func TestVaryWritesTheResponseHeader(t *testing.T) {
	res := newRecordingResponse()

	if err := vary.Vary(res, "Origin"); err != nil {
		t.Fatalf("Vary err = %v, want nil", err)
	}
	if got := res.GetHeader("Vary"); got != "Origin" {
		t.Errorf("Vary header = %q, want %q", got, "Origin")
	}
}

func TestVaryRequiresAResponse(t *testing.T) {
	err := vary.Vary(nil, "Origin")

	if !errors.Is(err, vary.ErrResponseRequired) {
		t.Errorf("Vary err = %v, want %v", err, vary.ErrResponseRequired)
	}
}

func TestVaryPropagatesAFieldError(t *testing.T) {
	res := newRecordingResponse()

	err := vary.Vary(res, "")

	if !errors.Is(err, vary.ErrFieldRequired) {
		t.Errorf("Vary err = %v, want %v", err, vary.ErrFieldRequired)
	}
	if got := res.GetHeader("Vary"); got != "" {
		t.Errorf("Vary header = %q, want it unset", got)
	}
}

func TestVaryFieldsWritesTheResponseHeader(t *testing.T) {
	res := newRecordingResponse()

	if err := vary.VaryFields(res, []string{"Accept", "Origin"}); err != nil {
		t.Fatalf("VaryFields err = %v, want nil", err)
	}
	if got := res.GetHeader("Vary"); got != "Accept, Origin" {
		t.Errorf("Vary header = %q, want %q", got, "Accept, Origin")
	}
}

func TestVaryFieldsRequiresAResponse(t *testing.T) {
	err := vary.VaryFields(nil, []string{"Origin"})

	if !errors.Is(err, vary.ErrResponseRequired) {
		t.Errorf("VaryFields err = %v, want %v", err, vary.ErrResponseRequired)
	}
}

func TestVaryFieldsLeavesTheHeaderUnsetForNoFields(t *testing.T) {
	res := newRecordingResponse()

	if err := vary.VaryFields(res, []string{}); err != nil {
		t.Fatalf("VaryFields err = %v, want nil", err)
	}
	if got := res.GetHeader("Vary"); got != "" {
		t.Errorf("Vary header = %q, want it unset", got)
	}
}

func TestVaryFieldsPropagatesAFieldError(t *testing.T) {
	res := newRecordingResponse()

	err := vary.VaryFields(res, nil)

	if !errors.Is(err, vary.ErrFieldRequired) {
		t.Errorf("VaryFields err = %v, want %v", err, vary.ErrFieldRequired)
	}
	if got := res.GetHeader("Vary"); got != "" {
		t.Errorf("Vary header = %q, want it unset", got)
	}
}
