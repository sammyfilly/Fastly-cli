package kvstoreentry

import (
	"io"

	"github.com/fastly/go-fastly/v8/fastly"

	"github.com/fastly/cli/pkg/cmd"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/cli/pkg/text"
)

// ListCommand calls the Fastly API to list the keys for a given kv store.
type ListCommand struct {
	cmd.Base
	cmd.JSONOutput

	manifest manifest.Data
	Input    fastly.ListKVStoreKeysInput
}

// NewListCommand returns a usable command registered under the parent.
func NewListCommand(parent cmd.Registerer, g *global.Data, m manifest.Data) *ListCommand {
	c := ListCommand{
		Base: cmd.Base{
			Globals: g,
		},
		manifest: m,
	}

	c.CmdClause = parent.Command("list", "List keys")

	// Required.
	c.CmdClause.Flag("store-id", "Store ID").Short('s').Required().StringVar(&c.Input.ID)

	// Optional.
	c.RegisterFlagBool(c.JSONFlag()) // --json
	return &c
}

// Exec invokes the application logic for the command.
func (c *ListCommand) Exec(_ io.Reader, out io.Writer) error {
	if c.Globals.Verbose() && c.JSONOutput.Enabled {
		return fsterr.ErrInvalidVerboseJSONCombo
	}

	var (
		cursor string
		keys   []string
		ok     bool
	)

	c.Input.Cursor = cursor

	spinner, err := text.NewSpinner(out)
	if err != nil {
		return err
	}
	msg := "Getting data"

	// A spinner produces output and is incompatible with JSON expected output.
	if !c.JSONOutput.Enabled {
		err := spinner.Start()
		if err != nil {
			return err
		}
		spinner.Message(msg + "... (this can take a few minutes depending on the number of entries)")
	}

	for {
		o, err := c.Globals.APIClient.ListKVStoreKeys(&c.Input)
		if err != nil {
			c.Globals.ErrLog.Add(err)
			if !c.JSONOutput.Enabled {
				spinner.StopFailMessage(msg)
				spinErr := spinner.StopFail()
				if spinErr != nil {
					return spinErr
				}
			}
			return err
		}

		keys = append(keys, o.Data...)

		c.Input.Cursor, ok = o.Meta["next_cursor"]
		if !ok {
			break
		}
	}

	if !c.JSONOutput.Enabled {
		spinner.StopMessage(msg)
		err := spinner.Stop()
		if err != nil {
			return err
		}
	}

	if keys == nil {
		if ok, err := c.WriteJSON(out, []string{}); ok {
			return err
		}
		text.Break(out)
		text.Output(out, "no keys")
		return nil
	}

	if ok, err := c.WriteJSON(out, keys); ok {
		return err
	}

	if c.Globals.Flags.Verbose {
		text.PrintKVStoreKeys(out, "", keys)
		return nil
	}

	for _, k := range keys {
		text.Output(out, k)
	}
	return nil
}
