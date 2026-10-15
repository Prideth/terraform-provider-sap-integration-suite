package integrationassessment

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateKeyCharacteristic_LinksTheGroupByID(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"kc-1","Name":"Retry","IsSingle":true,"KeyCharacteristicGroup":{"__deferred":{"uri":"x"}}}}`)
	k, err := c.CreateKeyCharacteristic(context.Background(), "Retry", nil, true, "group-1")
	if err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if r.method != http.MethodPost || r.path != "/intas/entities/v1/KeyCharacteristic" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
	if linkID(r.body, "KeyCharacteristicGroup") != "group-1" || r.body["IsSingle"] != true || r.body["Name"] != "Retry" {
		t.Errorf("body = %v", r.body)
	}
	if _, present := r.body["Description"]; present {
		t.Errorf("body = %v, want no Description when none is set", r.body)
	}
	if k.ID != "kc-1" || !k.IsSingle {
		t.Errorf("key characteristic = %+v", k)
	}
}

func TestUpdateKeyCharacteristic_LeavesTheTypeOut(t *testing.T) {
	c, got := fakeService(t, "")
	if err := c.UpdateKeyCharacteristic(context.Background(), "kc-1", "Retries", nil); err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if r.method != http.MethodPatch || r.path != "/intas/entities/v1/KeyCharacteristic('kc-1')" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
	if _, present := r.body["IsSingle"]; present || len(r.body) != 2 || r.body["Name"] != "Retries" {
		t.Errorf("body = %v, want Name and Description only: the tenant refuses IsSingle in an update", r.body)
	}
	if v, present := r.body["Description"]; !present || v != nil {
		t.Errorf("body = %v, want Description: null to clear it", r.body)
	}
}

func TestGetKeyCharacteristic_ExpandsItsGroup(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"kc-1","Name":"Retry","Description":"d","IsSingle":false,`+
		`"KeyCharacteristicGroup":{"Id":"group-1","Name":"Connectivity"}}}`)
	k, err := c.GetKeyCharacteristic(context.Background(), "kc-1")
	if err != nil {
		t.Fatal(err)
	}
	if q := (*got)[0].query; q != "$expand=KeyCharacteristicGroup" {
		t.Errorf("query = %q", q)
	}
	if k.KeyCharacteristicGroupID() != "group-1" || k.Description == nil || *k.Description != "d" {
		t.Errorf("key characteristic = %+v", k)
	}
}

func TestCreateKeyCharacteristicValue_LinksTheKeyCharacteristicByID(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"v-1","Name":"Yes"}}`)
	desc := "supported"
	if _, err := c.CreateKeyCharacteristicValue(context.Background(), "Yes", &desc, "kc-1"); err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if r.path != "/intas/entities/v1/KeyCharacteristicValue" || linkID(r.body, "KeyCharacteristic") != "kc-1" || r.body["Description"] != "supported" {
		t.Errorf("request = %s %s %v", r.method, r.path, r.body)
	}
}

func TestUpdateKeyCharacteristicValue_LeavesTheLinkOut(t *testing.T) {
	c, got := fakeService(t, "")
	if err := c.UpdateKeyCharacteristicValue(context.Background(), "v-1", "No", nil); err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if _, present := r.body["KeyCharacteristic"]; present || r.method != http.MethodPatch || r.body["Name"] != "No" {
		t.Errorf("request = %s %v, want a PATCH with Name and Description only", r.method, r.body)
	}
}

func TestKeyCharacteristicGroup_CreateUpdateDelete(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"group-1","Name":"Compliance"}}`)
	ctx := context.Background()
	if _, err := c.CreateKeyCharacteristicGroup(ctx, "Compliance", nil); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateKeyCharacteristicGroup(ctx, "group-1", "Compliance rules", nil); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteKeyCharacteristicGroup(ctx, "group-1"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /intas/entities/v1/KeyCharacteristicGroup",
		"PATCH /intas/entities/v1/KeyCharacteristicGroup('group-1')",
		"DELETE /intas/entities/v1/KeyCharacteristicGroup('group-1')",
	}
	for i, r := range *got {
		if r.method+" "+r.path != want[i] {
			t.Errorf("request %d = %s %s, want %s", i, r.method, r.path, want[i])
		}
	}
}
