package logshuttle_test

import (
	"bytes"
	"testing"

	"github.com/fastly/cli/pkg/cmd"
	"github.com/fastly/cli/pkg/commands/logging/logshuttle"
	"github.com/fastly/cli/pkg/config"
	"github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/cli/pkg/mock"
	"github.com/fastly/cli/pkg/testutil"
	"github.com/fastly/go-fastly/v8/fastly"
)

func TestCreateLogshuttleInput(t *testing.T) {
	for _, testcase := range []struct {
		name      string
		cmd       *logshuttle.CreateCommand
		want      *fastly.CreateLogshuttleInput
		wantError string
	}{
		{
			name: "required values set flag serviceID",
			cmd:  createCommandRequired(),
			want: &fastly.CreateLogshuttleInput{
				ServiceID:      "123",
				ServiceVersion: 4,
				Name:           fastly.String("log"),
				Token:          fastly.String("tkn"),
				URL:            fastly.String("example.com"),
			},
		},
		{
			name: "all values set flag serviceID",
			cmd:  createCommandAll(),
			want: &fastly.CreateLogshuttleInput{
				ServiceID:         "123",
				ServiceVersion:    4,
				Name:              fastly.String("log"),
				Format:            fastly.String(`%h %l %u %t "%r" %>s %b`),
				FormatVersion:     fastly.Int(2),
				URL:               fastly.String("example.com"),
				Token:             fastly.String("tkn"),
				ResponseCondition: fastly.String("Prevent default logging"),
				Placement:         fastly.String("none"),
			},
		},
		{
			name:      "error missing serviceID",
			cmd:       createCommandMissingServiceID(),
			want:      nil,
			wantError: errors.ErrNoServiceID.Error(),
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var bs []byte
			out := bytes.NewBuffer(bs)
			verboseMode := true

			serviceID, serviceVersion, err := cmd.ServiceDetails(cmd.ServiceDetailsOpts{
				AutoCloneFlag:      testcase.cmd.AutoClone,
				APIClient:          testcase.cmd.Globals.APIClient,
				Manifest:           testcase.cmd.Manifest,
				Out:                out,
				ServiceVersionFlag: testcase.cmd.ServiceVersion,
				VerboseMode:        verboseMode,
			})

			switch {
			case err != nil && testcase.wantError == "":
				t.Fatalf("unexpected error getting service details: %v", err)
				return
			case err != nil && testcase.wantError != "":
				testutil.AssertErrorContains(t, err, testcase.wantError)
				return
			case err == nil && testcase.wantError != "":
				t.Fatalf("expected error, have nil (service details: %s, %d)", serviceID, serviceVersion.Number)
			case err == nil && testcase.wantError == "":
				have, err := testcase.cmd.ConstructInput(serviceID, serviceVersion.Number)
				testutil.AssertErrorContains(t, err, testcase.wantError)
				testutil.AssertEqual(t, testcase.want, have)
			}
		})
	}
}

func TestUpdateLogshuttleInput(t *testing.T) {
	scenarios := []struct {
		name      string
		cmd       *logshuttle.UpdateCommand
		api       mock.API
		want      *fastly.UpdateLogshuttleInput
		wantError string
	}{
		{
			name: "no update",
			cmd:  updateCommandNoUpdate(),
			api: mock.API{
				ListVersionsFn:  testutil.ListVersions,
				CloneVersionFn:  testutil.CloneVersionResult(4),
				GetLogshuttleFn: getLogshuttleOK,
			},
			want: &fastly.UpdateLogshuttleInput{
				ServiceID:      "123",
				ServiceVersion: 4,
				Name:           "log",
			},
		},
		{
			name: "all values set flag serviceID",
			cmd:  updateCommandAll(),
			api: mock.API{
				ListVersionsFn:  testutil.ListVersions,
				CloneVersionFn:  testutil.CloneVersionResult(4),
				GetLogshuttleFn: getLogshuttleOK,
			},
			want: &fastly.UpdateLogshuttleInput{
				ServiceID:         "123",
				ServiceVersion:    4,
				Name:              "log",
				NewName:           fastly.String("new1"),
				Format:            fastly.String("new2"),
				FormatVersion:     fastly.Int(3),
				Token:             fastly.String("new3"),
				URL:               fastly.String("new4"),
				ResponseCondition: fastly.String("new5"),
				Placement:         fastly.String("new6"),
			},
		},
		{
			name:      "error missing serviceID",
			cmd:       updateCommandMissingServiceID(),
			want:      nil,
			wantError: errors.ErrNoServiceID.Error(),
		},
	}
	for testcaseIdx := range scenarios {
		testcase := &scenarios[testcaseIdx]
		t.Run(testcase.name, func(t *testing.T) {
			testcase.cmd.Globals.APIClient = testcase.api

			var bs []byte
			out := bytes.NewBuffer(bs)
			verboseMode := true

			serviceID, serviceVersion, err := cmd.ServiceDetails(cmd.ServiceDetailsOpts{
				AutoCloneFlag:      testcase.cmd.AutoClone,
				APIClient:          testcase.api,
				Manifest:           testcase.cmd.Manifest,
				Out:                out,
				ServiceVersionFlag: testcase.cmd.ServiceVersion,
				VerboseMode:        verboseMode,
			})

			switch {
			case err != nil && testcase.wantError == "":
				t.Fatalf("unexpected error getting service details: %v", err)
				return
			case err != nil && testcase.wantError != "":
				testutil.AssertErrorContains(t, err, testcase.wantError)
				return
			case err == nil && testcase.wantError != "":
				t.Fatalf("expected error, have nil (service details: %s, %d)", serviceID, serviceVersion.Number)
			case err == nil && testcase.wantError == "":
				have, err := testcase.cmd.ConstructInput(serviceID, serviceVersion.Number)
				testutil.AssertErrorContains(t, err, testcase.wantError)
				testutil.AssertEqual(t, testcase.want, have)
			}
		})
	}
}

func createCommandRequired() *logshuttle.CreateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}
	g.APIClient, _ = mock.APIClient(mock.API{
		ListVersionsFn: testutil.ListVersions,
		CloneVersionFn: testutil.CloneVersionResult(4),
	})("token", "endpoint")

	return &logshuttle.CreateCommand{
		Base: cmd.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		EndpointName: cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "log"},
		Token:        cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "tkn"},
		URL:          cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "example.com"},
		ServiceVersion: cmd.OptionalServiceVersion{
			OptionalString: cmd.OptionalString{Value: "1"},
		},
		AutoClone: cmd.OptionalAutoClone{
			OptionalBool: cmd.OptionalBool{
				Optional: cmd.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
	}
}

func createCommandAll() *logshuttle.CreateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}
	g.APIClient, _ = mock.APIClient(mock.API{
		ListVersionsFn: testutil.ListVersions,
		CloneVersionFn: testutil.CloneVersionResult(4),
	})("token", "endpoint")

	return &logshuttle.CreateCommand{
		Base: cmd.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		EndpointName: cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "log"},
		Token:        cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "tkn"},
		URL:          cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "example.com"},
		ServiceVersion: cmd.OptionalServiceVersion{
			OptionalString: cmd.OptionalString{Value: "1"},
		},
		AutoClone: cmd.OptionalAutoClone{
			OptionalBool: cmd.OptionalBool{
				Optional: cmd.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
		Format:            cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: `%h %l %u %t "%r" %>s %b`},
		FormatVersion:     cmd.OptionalInt{Optional: cmd.Optional{WasSet: true}, Value: 2},
		ResponseCondition: cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "Prevent default logging"},
		Placement:         cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "none"},
	}
}

func createCommandMissingServiceID() *logshuttle.CreateCommand {
	res := createCommandAll()
	res.Manifest = manifest.Data{}
	return res
}

func updateCommandNoUpdate() *logshuttle.UpdateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}

	return &logshuttle.UpdateCommand{
		Base: cmd.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		EndpointName: "log",
		ServiceVersion: cmd.OptionalServiceVersion{
			OptionalString: cmd.OptionalString{Value: "1"},
		},
		AutoClone: cmd.OptionalAutoClone{
			OptionalBool: cmd.OptionalBool{
				Optional: cmd.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
	}
}

func updateCommandAll() *logshuttle.UpdateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}

	return &logshuttle.UpdateCommand{
		Base: cmd.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		EndpointName: "log",
		ServiceVersion: cmd.OptionalServiceVersion{
			OptionalString: cmd.OptionalString{Value: "1"},
		},
		AutoClone: cmd.OptionalAutoClone{
			OptionalBool: cmd.OptionalBool{
				Optional: cmd.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
		NewName:           cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "new1"},
		Format:            cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "new2"},
		FormatVersion:     cmd.OptionalInt{Optional: cmd.Optional{WasSet: true}, Value: 3},
		Token:             cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "new3"},
		URL:               cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "new4"},
		ResponseCondition: cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "new5"},
		Placement:         cmd.OptionalString{Optional: cmd.Optional{WasSet: true}, Value: "new6"},
	}
}

func updateCommandMissingServiceID() *logshuttle.UpdateCommand {
	res := updateCommandAll()
	res.Manifest = manifest.Data{}
	return res
}
