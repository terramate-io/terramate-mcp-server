package terramate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMembershipsList_ParsesArray(t *testing.T) {
	payload := `[{"member_id":123,"org_uuid":"org-uuid","org_name":"acme","org_display_name":"Acme Inc","org_domain":"acme.example","role":"admin","status":"active"}]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/memberships" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	members, _, err := c.Memberships.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(members) != 1 || members[0].OrgName != "acme" || members[0].Role != "admin" {
		t.Fatalf("unexpected memberships: %+v", members)
	}
}
