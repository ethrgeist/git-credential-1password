package internal

import "testing"

// mockRunner implements OpRunner for testing.
type mockRunner struct {
	listItemsResult  *OpItemListResult
	listItemsErr     error
	getItemResult    OpItem
	getItemErr       error
	createItemResult []byte
	createItemErr    error
	editItemResult   []byte
	editItemErr      error
	deleteItemErr    error

	getItemCalledWith    []string
	createItemCalledWith [][]string
	editItemCalledWith   [][]string
	deleteItemCalledWith []string
}

func (m *mockRunner) ListItems() (*OpItemListResult, error) {
	return m.listItemsResult, m.listItemsErr
}

func (m *mockRunner) GetItem(id string) (OpItem, error) {
	m.getItemCalledWith = append(m.getItemCalledWith, id)
	return m.getItemResult, m.getItemErr
}

func (m *mockRunner) CreateItem(args ...string) ([]byte, error) {
	m.createItemCalledWith = append(m.createItemCalledWith, args)
	return m.createItemResult, m.createItemErr
}

func (m *mockRunner) EditItem(args ...string) ([]byte, error) {
	m.editItemCalledWith = append(m.editItemCalledWith, args)
	return m.editItemResult, m.editItemErr
}

func (m *mockRunner) DeleteItem(id string) error {
	m.deleteItemCalledWith = append(m.deleteItemCalledWith, id)
	return m.deleteItemErr
}

// setupTest configures global state for testing and restores it on cleanup.
func setupTest(t *testing.T, m *mockRunner) {
	t.Helper()

	origRunner := Runner
	origAccount := Account
	origVault := Vault
	origCategory := Category
	origPrefix := Prefix
	origUsernameField := UsernameField
	origPasswordField := PasswordField
	origAllowErase := AllowErase
	origReadOnly := ReadOnly
	origOpPath := OpPath
	origItemID := ItemID
	origOpItemFlags := OpItemFlags

	Runner = m
	Account = ""
	Vault = ""
	Category = "Login"
	Prefix = ""
	UsernameField = "username"
	PasswordField = "password"
	AllowErase = false
	ReadOnly = false
	OpPath = ""
	ItemID = ""
	OpItemFlags = nil

	t.Cleanup(func() {
		Runner = origRunner
		Account = origAccount
		Vault = origVault
		Category = origCategory
		Prefix = origPrefix
		UsernameField = origUsernameField
		PasswordField = origPasswordField
		AllowErase = origAllowErase
		ReadOnly = origReadOnly
		OpPath = origOpPath
		ItemID = origItemID
		OpItemFlags = origOpItemFlags
	})
}
