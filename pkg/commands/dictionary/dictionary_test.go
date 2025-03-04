package dictionary_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/fastly/go-fastly/v8/fastly"

	"github.com/fastly/cli/pkg/app"
	"github.com/fastly/cli/pkg/mock"
	"github.com/fastly/cli/pkg/testutil"
)

func TestDictionaryDescribe(t *testing.T) {
	args := testutil.Args
	scenarios := []struct {
		args       []string
		api        mock.API
		wantError  string
		wantOutput string
	}{
		{
			args:      args("dictionary describe --version 1 --service-id 123"),
			wantError: "error parsing arguments: required flag --name not provided",
		},
		{
			args: args("dictionary describe --version 1 --service-id 123 --name dict-1"),
			api: mock.API{
				ListVersionsFn:  testutil.ListVersions,
				GetDictionaryFn: describeDictionaryOK,
			},
			wantOutput: describeDictionaryOutput,
		},
		{
			args: args("dictionary describe --version 1 --service-id 123 --name dict-1"),
			api: mock.API{
				ListVersionsFn:  testutil.ListVersions,
				GetDictionaryFn: describeDictionaryOKDeleted,
			},
			wantOutput: describeDictionaryOutputDeleted,
		},
		{
			args: args("dictionary describe --version 1 --service-id 123 --name dict-1 --verbose"),
			api: mock.API{
				ListVersionsFn:        testutil.ListVersions,
				GetDictionaryFn:       describeDictionaryOK,
				GetDictionaryInfoFn:   getDictionaryInfoOK,
				ListDictionaryItemsFn: listDictionaryItemsOK,
			},
			wantOutput: describeDictionaryOutputVerbose,
		},
	}
	for testcaseIdx := range scenarios {
		testcase := &scenarios[testcaseIdx]
		t.Run(strings.Join(testcase.args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			opts := testutil.NewRunOpts(testcase.args, &stdout)
			opts.APIClient = mock.APIClient(testcase.api)
			err := app.Run(opts)
			testutil.AssertErrorContains(t, err, testcase.wantError)
			testutil.AssertString(t, testcase.wantOutput, stdout.String())
		})
	}
}

func TestDictionaryCreate(t *testing.T) {
	args := testutil.Args
	scenarios := []struct {
		args       []string
		api        mock.API
		wantError  string
		wantOutput string
	}{
		{
			args:      args("dictionary create --version 1"),
			wantError: "error reading service: no service ID found",
		},
		{
			args: args("dictionary create --version 1 --service-id 123 --name denylist --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				CreateDictionaryFn: createDictionaryOK,
			},
			wantOutput: createDictionaryOutput,
		},
		{
			args: args("dictionary create --version 1 --service-id 123 --name denylist --write-only --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				CreateDictionaryFn: createDictionaryOK,
			},
			wantOutput: createDictionaryOutputWriteOnly,
		},
		{
			args: args("dictionary create --version 1 --service-id 123 --name denylist --write-only fish --autoclone"),
			api: mock.API{
				ListVersionsFn: testutil.ListVersions,
				CloneVersionFn: testutil.CloneVersionResult(4),
			},
			wantError: "error parsing arguments: unexpected 'fish'",
		},
		{
			args: args("dictionary create --version 1 --service-id 123 --name denylist --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				CreateDictionaryFn: createDictionaryDuplicate,
			},
			wantError: "Duplicate record",
		},
	}
	for testcaseIdx := range scenarios {
		testcase := &scenarios[testcaseIdx]
		t.Run(strings.Join(testcase.args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			opts := testutil.NewRunOpts(testcase.args, &stdout)
			opts.APIClient = mock.APIClient(testcase.api)
			err := app.Run(opts)
			testutil.AssertErrorContains(t, err, testcase.wantError)
			testutil.AssertString(t, testcase.wantOutput, stdout.String())
		})
	}
}

func TestDeleteDictionary(t *testing.T) {
	args := testutil.Args
	scenarios := []struct {
		args       []string
		api        mock.API
		wantError  string
		wantOutput string
	}{
		{
			args:      args("dictionary delete --service-id 123 --version 1"),
			wantError: "error parsing arguments: required flag --name not provided",
		},
		{
			args: args("dictionary delete --service-id 123 --version 1 --name allowlist --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				DeleteDictionaryFn: deleteDictionaryOK,
			},
			wantOutput: deleteDictionaryOutput,
		},
		{
			args: args("dictionary delete --service-id 123 --version 1 --name allowlist --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				DeleteDictionaryFn: deleteDictionaryError,
			},
			wantError: errTest.Error(),
		},
	}
	for testcaseIdx := range scenarios {
		testcase := &scenarios[testcaseIdx]
		t.Run(strings.Join(testcase.args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			opts := testutil.NewRunOpts(testcase.args, &stdout)
			opts.APIClient = mock.APIClient(testcase.api)
			err := app.Run(opts)
			testutil.AssertErrorContains(t, err, testcase.wantError)
			testutil.AssertString(t, testcase.wantOutput, stdout.String())
		})
	}
}

func TestListDictionary(t *testing.T) {
	args := testutil.Args
	scenarios := []struct {
		args       []string
		api        mock.API
		wantError  string
		wantOutput string
	}{
		{
			args: args("dictionary list --version 1"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				ListDictionariesFn: listDictionariesOk,
			},
			wantError: "error reading service: no service ID found",
		},
		{
			args:      args("dictionary list --service-id 123"),
			wantError: "error parsing arguments: required flag --version not provided",
		},
		{
			args: args("dictionary list --version 1 --service-id 123"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				ListDictionariesFn: listDictionariesOk,
			},
			wantOutput: listDictionariesOutput,
		},
	}
	for testcaseIdx := range scenarios {
		testcase := &scenarios[testcaseIdx]
		t.Run(strings.Join(testcase.args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			opts := testutil.NewRunOpts(testcase.args, &stdout)
			opts.APIClient = mock.APIClient(testcase.api)
			err := app.Run(opts)
			testutil.AssertErrorContains(t, err, testcase.wantError)
			testutil.AssertString(t, testcase.wantOutput, stdout.String())
		})
	}
}

func TestUpdateDictionary(t *testing.T) {
	args := testutil.Args
	scenarios := []struct {
		args       []string
		api        mock.API
		wantError  string
		wantOutput string
	}{
		{
			args:      args("dictionary update --version 1 --name oldname --new-name newname"),
			wantError: "error reading service: no service ID found",
		},
		{
			args:      args("dictionary update --service-id 123 --name oldname --new-name newname"),
			wantError: "error parsing arguments: required flag --version not provided",
		},
		{
			args:      args("dictionary update --service-id 123 --version 1 --new-name newname"),
			wantError: "error parsing arguments: required flag --name not provided",
		},
		{
			args: args("dictionary update --service-id 123 --version 1 --name oldname --autoclone"),
			api: mock.API{
				ListVersionsFn: testutil.ListVersions,
				CloneVersionFn: testutil.CloneVersionResult(4),
			},
			wantError: "error parsing arguments: required flag --new-name or --write-only not provided",
		},
		{
			args: args("dictionary update --service-id 123 --version 1 --name oldname --new-name dict-1 --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				UpdateDictionaryFn: updateDictionaryNameOK,
			},
			wantOutput: updateDictionaryNameOutput,
		},
		{
			args: args("dictionary update --service-id 123 --version 1 --name oldname --new-name dict-1 --write-only true --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				UpdateDictionaryFn: updateDictionaryNameOK,
			},
			wantOutput: updateDictionaryNameOutput,
		},
		{
			args: args("dictionary update --service-id 123 --version 1 --name oldname --write-only true --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				UpdateDictionaryFn: updateDictionaryWriteOnlyOK,
			},
			wantOutput: updateDictionaryOutput,
		},
		{
			args: args("dictionary update -v --service-id 123 --version 1 --name oldname --new-name dict-1 --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				UpdateDictionaryFn: updateDictionaryNameOK,
			},
			wantOutput: updateDictionaryOutputVerbose,
		},
		{
			args: args("dictionary update --service-id 123 --version 1 --name oldname --new-name dict-1 --autoclone"),
			api: mock.API{
				ListVersionsFn:     testutil.ListVersions,
				CloneVersionFn:     testutil.CloneVersionResult(4),
				UpdateDictionaryFn: updateDictionaryError,
			},
			wantError: errTest.Error(),
		},
	}
	for testcaseIdx := range scenarios {
		testcase := &scenarios[testcaseIdx]
		t.Run(strings.Join(testcase.args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			opts := testutil.NewRunOpts(testcase.args, &stdout)
			opts.APIClient = mock.APIClient(testcase.api)
			err := app.Run(opts)
			t.Log(stdout.String())
			testutil.AssertErrorContains(t, err, testcase.wantError)
			testutil.AssertString(t, testcase.wantOutput, stdout.String())
		})
	}
}

func describeDictionaryOK(i *fastly.GetDictionaryInput) (*fastly.Dictionary, error) {
	return &fastly.Dictionary{
		ServiceID:      i.ServiceID,
		ServiceVersion: i.ServiceVersion,
		Name:           i.Name,
		CreatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
		WriteOnly:      false,
		ID:             "456",
		UpdatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
	}, nil
}

func describeDictionaryOKDeleted(i *fastly.GetDictionaryInput) (*fastly.Dictionary, error) {
	return &fastly.Dictionary{
		ServiceID:      i.ServiceID,
		ServiceVersion: i.ServiceVersion,
		Name:           i.Name,
		CreatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
		WriteOnly:      false,
		ID:             "456",
		UpdatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
		DeletedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:08Z"),
	}, nil
}

func createDictionaryOK(i *fastly.CreateDictionaryInput) (*fastly.Dictionary, error) {
	if i.WriteOnly == nil {
		i.WriteOnly = fastly.CBool(false)
	}
	return &fastly.Dictionary{
		ServiceID:      i.ServiceID,
		ServiceVersion: i.ServiceVersion,
		Name:           *i.Name,
		CreatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
		WriteOnly:      *i.WriteOnly == true,
		ID:             "456",
		UpdatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
	}, nil
}

// getDictionaryInfoOK mocks the response from fastly.GetDictionaryInfo, which is not otherwise used
// in the fastly-cli and will need to be updated here if that call changes.
// This function requires i.ID to equal "456" to enforce the input to this call matches the
// response to GetDictionaryInfo in describeDictionaryOK
func getDictionaryInfoOK(i *fastly.GetDictionaryInfoInput) (*fastly.DictionaryInfo, error) {
	if i.ID == "456" {
		return &fastly.DictionaryInfo{
			ItemCount:   2,
			LastUpdated: testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
			Digest:      "digest_hash",
		}, nil
	} else {
		return nil, errFail
	}
}

// listDictionaryItemsOK mocks the response from fastly.ListDictionaryItems which is primarily used
// in the fastly-cli.dictionaryitem package and will need to be updated here if that call changes
func listDictionaryItemsOK(i *fastly.ListDictionaryItemsInput) ([]*fastly.DictionaryItem, error) {
	return []*fastly.DictionaryItem{
		{
			ServiceID:    i.ServiceID,
			DictionaryID: i.DictionaryID,
			ItemKey:      "foo",
			ItemValue:    "bar",
			CreatedAt:    testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
			UpdatedAt:    testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
		},
		{
			ServiceID:    i.ServiceID,
			DictionaryID: i.DictionaryID,
			ItemKey:      "baz",
			ItemValue:    "bear",
			CreatedAt:    testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
			UpdatedAt:    testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
			DeletedAt:    testutil.MustParseTimeRFC3339("2001-02-03T04:06:08Z"),
		},
	}, nil
}

func createDictionaryDuplicate(*fastly.CreateDictionaryInput) (*fastly.Dictionary, error) {
	return nil, errors.New("Duplicate record")
}

func deleteDictionaryOK(*fastly.DeleteDictionaryInput) error {
	return nil
}

func deleteDictionaryError(*fastly.DeleteDictionaryInput) error {
	return errTest
}

func listDictionariesOk(i *fastly.ListDictionariesInput) ([]*fastly.Dictionary, error) {
	return []*fastly.Dictionary{
		{
			ServiceID:      i.ServiceID,
			ServiceVersion: i.ServiceVersion,
			Name:           "dict-1",
			CreatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
			WriteOnly:      false,
			ID:             "456",
			UpdatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
		},
		{
			ServiceID:      i.ServiceID,
			ServiceVersion: i.ServiceVersion,
			Name:           "dict-2",
			CreatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
			WriteOnly:      false,
			ID:             "456",
			UpdatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
		},
	}, nil
}

func updateDictionaryNameOK(i *fastly.UpdateDictionaryInput) (*fastly.Dictionary, error) {
	return &fastly.Dictionary{
		ServiceID:      i.ServiceID,
		ServiceVersion: i.ServiceVersion,
		Name:           *i.NewName,
		CreatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
		WriteOnly:      cbPtrIsTrue(i.WriteOnly),
		ID:             "456",
		UpdatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
	}, nil
}

func updateDictionaryWriteOnlyOK(i *fastly.UpdateDictionaryInput) (*fastly.Dictionary, error) {
	return &fastly.Dictionary{
		ServiceID:      i.ServiceID,
		ServiceVersion: i.ServiceVersion,
		Name:           i.Name,
		CreatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:06Z"),
		WriteOnly:      cbPtrIsTrue(i.WriteOnly),
		ID:             "456",
		UpdatedAt:      testutil.MustParseTimeRFC3339("2001-02-03T04:05:07Z"),
	}, nil
}

func cbPtrIsTrue(cb *fastly.Compatibool) bool {
	if cb != nil {
		return *cb == true
	}
	return false
}

func updateDictionaryError(i *fastly.UpdateDictionaryInput) (*fastly.Dictionary, error) {
	return nil, errTest
}

var (
	errTest = errors.New("an expected error occurred")
	errFail = errors.New("this error should not be returned and indicates a failure in the code")
)

var (
	createDictionaryOutput          = "\nSUCCESS: Created dictionary denylist (id 456, service 123, version 4)\n"
	createDictionaryOutputWriteOnly = "\nSUCCESS: Created dictionary denylist as write-only (id 456, service 123, version 4)\n"
	deleteDictionaryOutput          = "\nSUCCESS: Deleted dictionary allowlist (service 123 version 4)\n"
	updateDictionaryOutput          = "\nSUCCESS: Updated dictionary oldname (service 123 version 4)\n"
	updateDictionaryNameOutput      = "\nSUCCESS: Updated dictionary dict-1 (service 123 version 4)\n"
)

var updateDictionaryOutputVerbose = strings.Join(
	[]string{
		"Fastly API token not provided",
		"Fastly API endpoint: https://api.fastly.com",
		"",
		"Service ID (via --service-id): 123",
		"",
		"Service version 1 is not editable, so it was automatically cloned because --autoclone is enabled. Now operating on",
		"version 4.",
		"",
		strings.TrimSpace(updateDictionaryNameOutput),
		updateDictionaryOutputVersionCloned,
	},
	"\n")

var updateDictionaryOutputVersionCloned = strings.TrimSpace(`
Version: 4
ID: 456
Name: dict-1
Write Only: false
Created (UTC): 2001-02-03 04:05
Last edited (UTC): 2001-02-03 04:05
`) + "\n"

var describeDictionaryOutput = strings.TrimSpace(`
Service ID: 123
Version: 1
ID: 456
Name: dict-1
Write Only: false
Created (UTC): 2001-02-03 04:05
Last edited (UTC): 2001-02-03 04:05
`) + "\n"

var describeDictionaryOutputDeleted = strings.TrimSpace(`
Service ID: 123
Version: 1
ID: 456
Name: dict-1
Write Only: false
Created (UTC): 2001-02-03 04:05
Last edited (UTC): 2001-02-03 04:05
Deleted (UTC): 2001-02-03 04:05
`) + "\n"

var describeDictionaryOutputVerbose = strings.TrimSpace(`
Fastly API token not provided
Fastly API endpoint: https://api.fastly.com

Service ID (via --service-id): 123

Version: 1
ID: 456
Name: dict-1
Write Only: false
Created (UTC): 2001-02-03 04:05
Last edited (UTC): 2001-02-03 04:05
Digest: digest_hash
Item Count: 2
Item 1/2:
	Item Key: foo
	Item Value: bar
Item 2/2:
	Item Key: baz
	Item Value: bear
`) + "\n"

var listDictionariesOutput = "\n" + strings.TrimSpace(`
Service ID: 123
Version: 1
ID: 456
Name: dict-1
Write Only: false
Created (UTC): 2001-02-03 04:05
Last edited (UTC): 2001-02-03 04:05
ID: 456
Name: dict-2
Write Only: false
Created (UTC): 2001-02-03 04:05
Last edited (UTC): 2001-02-03 04:05
`) + "\n"
