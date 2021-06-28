package heroku

import (
	"io"

	"github.com/fastly/cli/pkg/cmd"
	"github.com/fastly/cli/pkg/compute/manifest"
	"github.com/fastly/cli/pkg/config"
	"github.com/fastly/cli/pkg/text"
	"github.com/fastly/go-fastly/v3/fastly"
)

// UpdateCommand calls the Fastly API to update a Heroku logging endpoint.
type UpdateCommand struct {
	cmd.Base
	manifest manifest.Data

	// required
	EndpointName   string // Can't shadow cmd.Base method Name().
	serviceVersion cmd.OptionalServiceVersion

	// optional
	autoClone         cmd.OptionalAutoClone
	NewName           cmd.OptionalString
	Format            cmd.OptionalString
	FormatVersion     cmd.OptionalUint
	URL               cmd.OptionalString
	Token             cmd.OptionalString
	ResponseCondition cmd.OptionalString
	Placement         cmd.OptionalString
}

// NewUpdateCommand returns a usable command registered under the parent.
func NewUpdateCommand(parent cmd.Registerer, globals *config.Data) *UpdateCommand {
	var c UpdateCommand
	c.Globals = globals
	c.manifest.File.SetOutput(c.Globals.Output)
	c.manifest.File.Read(manifest.Filename)
	c.CmdClause = parent.Command("update", "Update a Heroku logging endpoint on a Fastly service version")
	c.RegisterServiceVersionFlag(cmd.ServiceVersionFlagOpts{
		Dst: &c.serviceVersion.Value,
	})
	c.RegisterAutoCloneFlag(cmd.AutoCloneFlagOpts{
		Action: c.autoClone.Set,
		Dst:    &c.autoClone.Value,
	})
	c.CmdClause.Flag("name", "The name of the Heroku logging object").Short('n').Required().StringVar(&c.EndpointName)
	c.RegisterServiceIDFlag(&c.manifest.Flag.ServiceID)
	c.CmdClause.Flag("new-name", "New name of the Heroku logging object").Action(c.NewName.Set).StringVar(&c.NewName.Value)
	c.CmdClause.Flag("format", "Apache style log formatting").Action(c.Format.Set).StringVar(&c.Format.Value)
	c.CmdClause.Flag("format-version", "The version of the custom logging format used for the configured endpoint. Can be either 2 (default) or 1").Action(c.FormatVersion.Set).UintVar(&c.FormatVersion.Value)
	c.CmdClause.Flag("url", "The url to stream logs to").Action(c.URL.Set).StringVar(&c.URL.Value)
	c.CmdClause.Flag("auth-token", "The token to use for authentication (https://devcenter.heroku.com/articles/add-on-partner-log-integration)").Action(c.Token.Set).StringVar(&c.Token.Value)
	c.CmdClause.Flag("response-condition", "The name of an existing condition in the configured endpoint, or leave blank to always execute").Action(c.ResponseCondition.Set).StringVar(&c.ResponseCondition.Value)
	c.CmdClause.Flag("placement", "Where in the generated VCL the logging call should be placed, overriding any format_version default. Can be none or waf_debug").Action(c.Placement.Set).StringVar(&c.Placement.Value)
	return &c
}

// constructInput transforms values parsed from CLI flags into an object to be used by the API client library.
func (c *UpdateCommand) constructInput(serviceID string, serviceVersion int) (*fastly.UpdateHerokuInput, error) {
	input := fastly.UpdateHerokuInput{
		ServiceID:      serviceID,
		ServiceVersion: serviceVersion,
		Name:           c.EndpointName,
	}

	if c.NewName.WasSet {
		input.NewName = fastly.String(c.NewName.Value)
	}

	if c.Format.WasSet {
		input.Format = fastly.String(c.Format.Value)
	}

	if c.FormatVersion.WasSet {
		input.FormatVersion = fastly.Uint(c.FormatVersion.Value)
	}

	if c.Token.WasSet {
		input.Token = fastly.String(c.Token.Value)
	}

	if c.URL.WasSet {
		input.URL = fastly.String(c.URL.Value)
	}

	if c.ResponseCondition.WasSet {
		input.ResponseCondition = fastly.String(c.ResponseCondition.Value)
	}

	if c.Placement.WasSet {
		input.Placement = fastly.String(c.Placement.Value)
	}

	return &input, nil
}

// Exec invokes the application logic for the command.
func (c *UpdateCommand) Exec(in io.Reader, out io.Writer) error {
	serviceID, serviceVersion, err := cmd.ServiceDetails(cmd.ServiceDetailsOpts{
		AutoCloneFlag:      c.autoClone,
		Client:             c.Globals.Client,
		Manifest:           c.manifest,
		Out:                out,
		ServiceVersionFlag: c.serviceVersion,
		VerboseMode:        c.Globals.Flag.Verbose,
	})
	if err != nil {
		return err
	}

	input, err := c.constructInput(serviceID, serviceVersion.Number)
	if err != nil {
		return err
	}

	heroku, err := c.Globals.Client.UpdateHeroku(input)
	if err != nil {
		return err
	}

	text.Success(out, "Updated Heroku logging endpoint %s (service %s version %d)", heroku.Name, heroku.ServiceID, heroku.ServiceVersion)
	return nil
}
