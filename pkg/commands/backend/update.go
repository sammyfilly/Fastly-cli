package backend

import (
	"io"

	"github.com/fastly/cli/pkg/cmd"
	"github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/cli/pkg/text"
	"github.com/fastly/go-fastly/v8/fastly"
)

// UpdateCommand calls the Fastly API to update backends.
type UpdateCommand struct {
	cmd.Base
	manifest       manifest.Data
	serviceName    cmd.OptionalServiceNameID
	serviceVersion cmd.OptionalServiceVersion
	autoClone      cmd.OptionalAutoClone

	name                string
	NewName             cmd.OptionalString
	Comment             cmd.OptionalString
	Address             cmd.OptionalString
	Port                cmd.OptionalInt
	OverrideHost        cmd.OptionalString
	ConnectTimeout      cmd.OptionalInt
	MaxConn             cmd.OptionalInt
	FirstByteTimeout    cmd.OptionalInt
	BetweenBytesTimeout cmd.OptionalInt
	AutoLoadbalance     cmd.OptionalBool
	Weight              cmd.OptionalInt
	RequestCondition    cmd.OptionalString
	HealthCheck         cmd.OptionalString
	Hostname            cmd.OptionalString
	Shield              cmd.OptionalString
	UseSSL              cmd.OptionalBool
	SSLCheckCert        cmd.OptionalBool
	SSLCACert           cmd.OptionalString
	SSLClientCert       cmd.OptionalString
	SSLClientKey        cmd.OptionalString
	SSLCertHostname     cmd.OptionalString
	SSLSNIHostname      cmd.OptionalString
	MinTLSVersion       cmd.OptionalString
	MaxTLSVersion       cmd.OptionalString
	SSLCiphers          cmd.OptionalString
}

// NewUpdateCommand returns a usable command registered under the parent.
func NewUpdateCommand(parent cmd.Registerer, g *global.Data, m manifest.Data) *UpdateCommand {
	c := UpdateCommand{
		Base: cmd.Base{
			Globals: g,
		},
		manifest: m,
	}
	c.CmdClause = parent.Command("update", "Update a backend on a Fastly service version")

	// Required.
	c.RegisterFlag(cmd.StringFlagOpts{
		Name:        cmd.FlagVersionName,
		Description: cmd.FlagVersionDesc,
		Dst:         &c.serviceVersion.Value,
		Required:    true,
	})
	c.CmdClause.Flag("name", "backend name").Short('n').Required().StringVar(&c.name)

	// Optional.
	c.CmdClause.Flag("address", "A hostname, IPv4, or IPv6 address for the backend").Action(c.Address.Set).StringVar(&c.Address.Value)
	c.RegisterAutoCloneFlag(cmd.AutoCloneFlagOpts{
		Action: c.autoClone.Set,
		Dst:    &c.autoClone.Value,
	})
	c.CmdClause.Flag("auto-loadbalance", "Whether or not this backend should be automatically load balanced").Action(c.AutoLoadbalance.Set).BoolVar(&c.AutoLoadbalance.Value)
	c.CmdClause.Flag("between-bytes-timeout", "How long to wait between bytes in milliseconds").Action(c.BetweenBytesTimeout.Set).IntVar(&c.BetweenBytesTimeout.Value)
	c.CmdClause.Flag("comment", "A descriptive note").Action(c.Comment.Set).StringVar(&c.Comment.Value)
	c.CmdClause.Flag("connect-timeout", "How long to wait for a timeout in milliseconds").Action(c.ConnectTimeout.Set).IntVar(&c.ConnectTimeout.Value)
	c.CmdClause.Flag("first-byte-timeout", "How long to wait for the first bytes in milliseconds").Action(c.FirstByteTimeout.Set).IntVar(&c.FirstByteTimeout.Value)
	c.CmdClause.Flag("healthcheck", "The name of the healthcheck to use with this backend").Action(c.HealthCheck.Set).StringVar(&c.HealthCheck.Value)
	c.CmdClause.Flag("max-conn", "Maximum number of connections").Action(c.MaxConn.Set).IntVar(&c.MaxConn.Value)
	c.CmdClause.Flag("max-tls-version", "Maximum allowed TLS version on SSL connections to this backend").Action(c.MaxTLSVersion.Set).StringVar(&c.MaxTLSVersion.Value)
	c.CmdClause.Flag("min-tls-version", "Minimum allowed TLS version on SSL connections to this backend").Action(c.MinTLSVersion.Set).StringVar(&c.MinTLSVersion.Value)
	c.CmdClause.Flag("new-name", "New backend name").Action(c.NewName.Set).StringVar(&c.NewName.Value)
	c.CmdClause.Flag("override-host", "The hostname to override the Host header").Action(c.OverrideHost.Set).StringVar(&c.OverrideHost.Value)
	c.CmdClause.Flag("port", "Port number of the address").Action(c.Port.Set).IntVar(&c.Port.Value)
	c.CmdClause.Flag("request-condition", "condition, which if met, will select this backend during a request").Action(c.RequestCondition.Set).StringVar(&c.RequestCondition.Value)
	c.RegisterFlag(cmd.StringFlagOpts{
		Name:        cmd.FlagServiceIDName,
		Description: cmd.FlagServiceIDDesc,
		Dst:         &c.manifest.Flag.ServiceID,
		Short:       's',
	})
	c.RegisterFlag(cmd.StringFlagOpts{
		Action:      c.serviceName.Set,
		Name:        cmd.FlagServiceName,
		Description: cmd.FlagServiceDesc,
		Dst:         &c.serviceName.Value,
	})
	c.CmdClause.Flag("shield", "The shield POP designated to reduce inbound load on this origin by serving the cached data to the rest of the network").Action(c.Shield.Set).StringVar(&c.Shield.Value)
	c.CmdClause.Flag("ssl-ca-cert", "CA certificate attached to origin").Action(c.SSLCACert.Set).StringVar(&c.SSLCACert.Value)
	c.CmdClause.Flag("ssl-cert-hostname", "Overrides ssl_hostname, but only for cert verification. Does not affect SNI at all.").Action(c.SSLCertHostname.Set).StringVar(&c.SSLCertHostname.Value)
	c.CmdClause.Flag("ssl-check-cert", "Be strict on checking SSL certs").Action(c.SSLCheckCert.Set).BoolVar(&c.SSLCheckCert.Value)
	c.CmdClause.Flag("ssl-ciphers", "List of OpenSSL ciphers (https://www.openssl.org/docs/man1.0.2/man1/ciphers)").Action(c.SSLCiphers.Set).StringVar(&c.SSLCiphers.Value)
	c.CmdClause.Flag("ssl-client-cert", "Client certificate attached to origin").Action(c.SSLClientCert.Set).StringVar(&c.SSLClientCert.Value)
	c.CmdClause.Flag("ssl-client-key", "Client key attached to origin").Action(c.SSLClientKey.Set).StringVar(&c.SSLClientKey.Value)
	c.CmdClause.Flag("ssl-sni-hostname", "Overrides ssl_hostname, but only for SNI in the handshake. Does not affect cert validation at all.").Action(c.SSLSNIHostname.Set).StringVar(&c.SSLSNIHostname.Value)
	c.CmdClause.Flag("use-ssl", "Whether or not to use SSL to reach the backend").Action(c.UseSSL.Set).BoolVar(&c.UseSSL.Value)
	c.CmdClause.Flag("weight", "Weight used to load balance this backend against others").Action(c.Weight.Set).IntVar(&c.Weight.Value)
	return &c
}

// Exec invokes the application logic for the command.
func (c *UpdateCommand) Exec(_ io.Reader, out io.Writer) error {
	serviceID, serviceVersion, err := cmd.ServiceDetails(cmd.ServiceDetailsOpts{
		AutoCloneFlag:      c.autoClone,
		APIClient:          c.Globals.APIClient,
		Manifest:           c.manifest,
		Out:                out,
		ServiceNameFlag:    c.serviceName,
		ServiceVersionFlag: c.serviceVersion,
		VerboseMode:        c.Globals.Flags.Verbose,
	})
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]any{
			"Service ID":      serviceID,
			"Service Version": errors.ServiceVersion(serviceVersion),
		})
		return err
	}

	input := &fastly.UpdateBackendInput{
		ServiceID:      serviceID,
		ServiceVersion: serviceVersion.Number,
		Name:           c.name,
	}

	if c.NewName.WasSet {
		input.NewName = &c.NewName.Value
	}

	if c.Comment.WasSet {
		input.Comment = &c.Comment.Value
	}

	if c.Address.WasSet {
		input.Address = &c.Address.Value
	}

	if c.Port.WasSet {
		input.Port = &c.Port.Value
	}

	if c.OverrideHost.WasSet {
		input.OverrideHost = &c.OverrideHost.Value
	}

	if c.ConnectTimeout.WasSet {
		input.ConnectTimeout = &c.ConnectTimeout.Value
	}

	if c.MaxConn.WasSet {
		input.MaxConn = &c.MaxConn.Value
	}

	if c.FirstByteTimeout.WasSet {
		input.FirstByteTimeout = &c.FirstByteTimeout.Value
	}

	if c.BetweenBytesTimeout.WasSet {
		input.BetweenBytesTimeout = &c.BetweenBytesTimeout.Value
	}

	if c.AutoLoadbalance.WasSet {
		input.AutoLoadbalance = fastly.CBool(c.AutoLoadbalance.Value)
	}

	if c.Weight.WasSet {
		input.Weight = &c.Weight.Value
	}

	if c.RequestCondition.WasSet {
		input.RequestCondition = &c.RequestCondition.Value
	}

	if c.HealthCheck.WasSet {
		input.HealthCheck = &c.HealthCheck.Value
	}

	if c.Shield.WasSet {
		input.Shield = &c.Shield.Value
	}

	if c.UseSSL.WasSet {
		input.UseSSL = fastly.CBool(c.UseSSL.Value)
	}

	if c.SSLCheckCert.WasSet {
		input.SSLCheckCert = fastly.CBool(c.SSLCheckCert.Value)
	}

	if c.SSLCACert.WasSet {
		input.SSLCACert = &c.SSLCACert.Value
	}

	if c.SSLClientCert.WasSet {
		input.SSLClientCert = &c.SSLClientCert.Value
	}

	if c.SSLClientKey.WasSet {
		input.SSLClientKey = &c.SSLClientKey.Value
	}

	if c.SSLCertHostname.WasSet {
		input.SSLCertHostname = &c.SSLCertHostname.Value
	}

	if c.SSLSNIHostname.WasSet {
		input.SSLSNIHostname = &c.SSLSNIHostname.Value
	}

	if c.MinTLSVersion.WasSet {
		input.MinTLSVersion = &c.MinTLSVersion.Value
	}

	if c.MaxTLSVersion.WasSet {
		input.MaxTLSVersion = &c.MaxTLSVersion.Value
	}

	if c.SSLCiphers.WasSet {
		input.SSLCiphers = &c.SSLCiphers.Value
	}

	b, err := c.Globals.APIClient.UpdateBackend(input)
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]any{
			"Service ID":      serviceID,
			"Service Version": serviceVersion.Number,
		})
		return err
	}

	text.Success(out, "Updated backend %s (service %s version %d)", b.Name, b.ServiceID, b.ServiceVersion)
	return nil
}
