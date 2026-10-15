package integrationassessment

import "context"

// Key characteristics a tenant maintains itself: groups, key characteristics
// and their values. SAP delivers its own entries of all three, and SAP's
// Entities API specification documents create (POST), update (PATCH) and
// delete for each set, with the bodies below. A create links a key
// characteristic to its group and a value to its key characteristic as
// {"Id": ...}; the update bodies list no links. The specification's
// KeyCharacteristic update body also lists IsSingle, but a tenant refused it
// (400 V101 "The property IsSingle is not allowed to be updated",
// 2026-10-15), so the update leaves it out.

type keyCharacteristicGroupWrite struct {
	Name        string  `json:"Name"`
	Description *string `json:"Description"`
}

type keyCharacteristicCreate struct {
	Name                   string  `json:"Name"`
	Description            *string `json:"Description,omitempty"`
	IsSingle               bool    `json:"IsSingle"`
	KeyCharacteristicGroup *idRef  `json:"KeyCharacteristicGroup"`
}

type keyCharacteristicUpdate struct {
	Name        string  `json:"Name"`
	Description *string `json:"Description"`
}

type keyCharacteristicValueCreate struct {
	Name              string  `json:"Name"`
	Description       *string `json:"Description,omitempty"`
	KeyCharacteristic *idRef  `json:"KeyCharacteristic"`
}

type keyCharacteristicValueUpdate struct {
	Name        string  `json:"Name"`
	Description *string `json:"Description"`
}

// GetKeyCharacteristicGroup reads a key characteristic group by Id.
func (c *Client) GetKeyCharacteristicGroup(ctx context.Context, id string) (*KeyCharacteristicGroup, error) {
	var g KeyCharacteristicGroup
	return &g, c.get(ctx, keyCharacteristicGroupEntitySet, id, "", &g)
}

// CreateKeyCharacteristicGroup creates a key characteristic group.
func (c *Client) CreateKeyCharacteristicGroup(ctx context.Context, name string, description *string) (*KeyCharacteristicGroup, error) {
	var g KeyCharacteristicGroup
	return &g, c.create(ctx, keyCharacteristicGroupEntitySet, keyCharacteristicGroupWrite{Name: name, Description: description}, &g)
}

// UpdateKeyCharacteristicGroup changes a group's name and description.
func (c *Client) UpdateKeyCharacteristicGroup(ctx context.Context, id, name string, description *string) error {
	return c.patch(ctx, keyCharacteristicGroupEntitySet, id, keyCharacteristicGroupWrite{Name: name, Description: description})
}

// DeleteKeyCharacteristicGroup deletes a key characteristic group.
func (c *Client) DeleteKeyCharacteristicGroup(ctx context.Context, id string) error {
	return c.remove(ctx, keyCharacteristicGroupEntitySet, id)
}

// GetKeyCharacteristic reads a key characteristic with its group link.
func (c *Client) GetKeyCharacteristic(ctx context.Context, id string) (*KeyCharacteristic, error) {
	var k KeyCharacteristic
	return &k, c.get(ctx, keyCharacteristicEntitySet, id, "KeyCharacteristicGroup", &k)
}

// CreateKeyCharacteristic creates a key characteristic in the group
// groupID.
func (c *Client) CreateKeyCharacteristic(ctx context.Context, name string, description *string, isSingle bool, groupID string) (*KeyCharacteristic, error) {
	var k KeyCharacteristic
	return &k, c.create(ctx, keyCharacteristicEntitySet, keyCharacteristicCreate{
		Name: name, Description: description, IsSingle: isSingle, KeyCharacteristicGroup: ref(groupID),
	}, &k)
}

// UpdateKeyCharacteristic changes a key characteristic's name and
// description. Its type cannot change.
func (c *Client) UpdateKeyCharacteristic(ctx context.Context, id, name string, description *string) error {
	return c.patch(ctx, keyCharacteristicEntitySet, id, keyCharacteristicUpdate{Name: name, Description: description})
}

// DeleteKeyCharacteristic deletes a key characteristic.
func (c *Client) DeleteKeyCharacteristic(ctx context.Context, id string) error {
	return c.remove(ctx, keyCharacteristicEntitySet, id)
}

// GetKeyCharacteristicValue reads a key characteristic value with its key
// characteristic link.
func (c *Client) GetKeyCharacteristicValue(ctx context.Context, id string) (*KeyCharacteristicValue, error) {
	var v KeyCharacteristicValue
	return &v, c.get(ctx, keyCharacteristicValueEntitySet, id, "KeyCharacteristic", &v)
}

// CreateKeyCharacteristicValue creates a value of the key characteristic
// keyCharacteristicID.
func (c *Client) CreateKeyCharacteristicValue(ctx context.Context, name string, description *string, keyCharacteristicID string) (*KeyCharacteristicValue, error) {
	var v KeyCharacteristicValue
	return &v, c.create(ctx, keyCharacteristicValueEntitySet, keyCharacteristicValueCreate{
		Name: name, Description: description, KeyCharacteristic: ref(keyCharacteristicID),
	}, &v)
}

// UpdateKeyCharacteristicValue changes a value's name and description.
func (c *Client) UpdateKeyCharacteristicValue(ctx context.Context, id, name string, description *string) error {
	return c.patch(ctx, keyCharacteristicValueEntitySet, id, keyCharacteristicValueUpdate{Name: name, Description: description})
}

// DeleteKeyCharacteristicValue deletes a key characteristic value.
func (c *Client) DeleteKeyCharacteristicValue(ctx context.Context, id string) error {
	return c.remove(ctx, keyCharacteristicValueEntitySet, id)
}
