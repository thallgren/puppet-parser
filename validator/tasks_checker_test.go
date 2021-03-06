package validator

import "testing"

func TestTasksResourceValidation(t *testing.T) {
	PuppetTasks = true
	defer func() { PuppetTasks = false }()

	expectIssues(t, `class { my: message => 'syntax ok' }`, VALIDATE_CATALOG_OPERATION_NOT_SUPPORTED)

	expectIssues(t, `foo { my: message => 'syntax ok' }`, VALIDATE_CATALOG_OPERATION_NOT_SUPPORTED)

	expectIssues(t, `@foo { my: message => 'syntax ok' }`, VALIDATE_CATALOG_OPERATION_NOT_SUPPORTED)

	expectIssues(t, `@@foo { my: message => 'syntax ok' }`, VALIDATE_CATALOG_OPERATION_NOT_SUPPORTED)

	expectIssues(t, `@class { my: message => 'syntax ok' }`, VALIDATE_CATALOG_OPERATION_NOT_SUPPORTED)

	expectIssues(t, `@@class { my: message => 'syntax ok' }`, VALIDATE_CATALOG_OPERATION_NOT_SUPPORTED)
}
