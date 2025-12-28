package makecloud

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	bcc "github.com/basis-cloud/bcc-go/bcc"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
)

const (
	DriverName        = "makecloud"
	defaultAPIBaseURL = "https://cp.iteco.cloud"
)

type Driver struct {
	*drivers.BaseDriver

	APIToken      string `json:"apiToken,omitempty"`
	APIBaseURL    string `json:"apiBaseUrl,omitempty"`
	APICACert     string `json:"apiCaCert,omitempty"`
	APIClientCert string `json:"apiClientCert,omitempty"`
	APIClientKey  string `json:"apiClientKey,omitempty"`
	APIInsecure   bool   `json:"apiInsecure,omitempty"`

	VDCID  string `json:"vdcId,omitempty"`
	VMID   string `json:"vmId,omitempty"`
	PortID string `json:"portId,omitempty"`

	TemplateID   string `json:"templateId,omitempty"`
	TemplateName string `json:"templateName,omitempty"`

	NetworkID           string   `json:"networkId,omitempty"`
	FirewallTemplateIDs []string `json:"firewallTemplateIds,omitempty"`
	FloatingIP          string   `json:"floatingIp,omitempty"`
	AllocateFloatingIP  bool     `json:"allocateFloatingIp,omitempty"`
	AllocatedFloatingID string   `json:"allocatedFloatingId,omitempty"`

	StorageProfileID string  `json:"storageProfileId,omitempty"`
	DiskSizeGB       int     `json:"diskSizeGb,omitempty"`
	CPU              int     `json:"cpu,omitempty"`
	RAMGB            float64 `json:"ramGb,omitempty"`

	Tags            []string `json:"tags,omitempty"`
	TemplateField   []string `json:"templateField,omitempty"`
	UserData        string   `json:"userData,omitempty"`
	NoInjectSSHKey  bool     `json:"noInjectSshKey,omitempty"`
	WaitTimeoutSecs int      `json:"waitTimeoutSecs,omitempty"`
}

func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
			SSHUser:     drivers.DefaultSSHUser,
			SSHPort:     drivers.DefaultSSHPort,
		},
		APIBaseURL:      defaultAPIBaseURL,
		CPU:             2,
		RAMGB:           4,
		DiskSizeGB:      40,
		WaitTimeoutSecs: 600,
	}
}

func (d *Driver) DriverName() string {
	return DriverName
}

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			Name:   "makecloud-token",
			Usage:  "MakeCloud API token",
			EnvVar: "MAKECLOUD_TOKEN",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-base-url",
			Usage:  "MakeCloud API base URL",
			EnvVar: "MAKECLOUD_BASE_URL",
			Value:  defaultAPIBaseURL,
		},
		mcnflag.BoolFlag{
			Name:   "makecloud-insecure",
			Usage:  "Skip TLS verification (not recommended)",
			EnvVar: "MAKECLOUD_INSECURE",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-ca-cert",
			Usage:  "Path to custom CA certificate (PEM)",
			EnvVar: "MAKECLOUD_CA_CERT",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-client-cert",
			Usage:  "Path to client certificate (PEM)",
			EnvVar: "MAKECLOUD_CLIENT_CERT",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-client-key",
			Usage:  "Path to client certificate key (PEM)",
			EnvVar: "MAKECLOUD_CLIENT_KEY",
		},

		mcnflag.StringFlag{
			Name:   "makecloud-vdc-id",
			Usage:  "VDC ID",
			EnvVar: "MAKECLOUD_VDC_ID",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-template-id",
			Usage:  "Template ID",
			EnvVar: "MAKECLOUD_TEMPLATE_ID",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-template-name",
			Usage:  "Template name (used if template-id is not set)",
			EnvVar: "MAKECLOUD_TEMPLATE_NAME",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-network-id",
			Usage:  "Network ID (defaults to VDC default network)",
			EnvVar: "MAKECLOUD_NETWORK_ID",
		},
		mcnflag.StringSliceFlag{
			Name:   "makecloud-firewall-template-id",
			Usage:  "Firewall template IDs to attach to VM port (repeatable)",
			EnvVar: "MAKECLOUD_FIREWALL_TEMPLATE_ID",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-floating-ip",
			Usage:  "Floating IP (address or ID) to attach to VM",
			EnvVar: "MAKECLOUD_FLOATING_IP",
		},
		mcnflag.BoolFlag{
			Name:   "makecloud-allocate-floating-ip",
			Usage:  "Allocate a new public floating IP (and delete it on remove). Ignored if makecloud-floating-ip is set",
			EnvVar: "MAKECLOUD_ALLOCATE_FLOATING_IP",
		},

		mcnflag.StringFlag{
			Name:   "makecloud-storage-profile-id",
			Usage:  "Storage profile ID (if not set and VDC has a single profile, it will be used)",
			EnvVar: "MAKECLOUD_STORAGE_PROFILE_ID",
		},
		mcnflag.IntFlag{
			Name:   "makecloud-disk-size",
			Usage:  "Root disk size in GiB",
			EnvVar: "MAKECLOUD_DISK_SIZE",
			Value:  40,
		},
		mcnflag.IntFlag{
			Name:   "makecloud-cpu",
			Usage:  "CPU cores",
			EnvVar: "MAKECLOUD_CPU",
			Value:  2,
		},
		mcnflag.StringFlag{
			Name:   "makecloud-ram",
			Usage:  "RAM in GiB (float allowed, e.g. 3.5)",
			EnvVar: "MAKECLOUD_RAM",
			Value:  "4",
		},

		mcnflag.StringSliceFlag{
			Name:   "makecloud-tags",
			Usage:  "Tags to set on VM/port (repeatable)",
			EnvVar: "MAKECLOUD_TAGS",
		},
		mcnflag.StringSliceFlag{
			Name:   "makecloud-template-field",
			Usage:  "Template fields as key=value (key can be field ID, system_alias or name; repeatable)",
			EnvVar: "MAKECLOUD_TEMPLATE_FIELD",
		},
		mcnflag.StringFlag{
			Name:   "makecloud-user-data",
			Usage:  "Cloud-init user-data (inline, @path, or path to file)",
			EnvVar: "MAKECLOUD_USER_DATA",
		},
		mcnflag.BoolFlag{
			Name:   "makecloud-no-inject-ssh-key",
			Usage:  "Disable automatic SSH public key injection via cloud-init",
			EnvVar: "MAKECLOUD_NO_INJECT_SSH_KEY",
		},
		mcnflag.IntFlag{
			Name:   "makecloud-wait-timeout",
			Usage:  "Timeout (seconds) for create/start operations",
			EnvVar: "MAKECLOUD_WAIT_TIMEOUT",
			Value:  600,
		},

		mcnflag.StringFlag{
			Name:   "makecloud-ssh-user",
			Usage:  "SSH username",
			EnvVar: "MAKECLOUD_SSH_USER",
			Value:  drivers.DefaultSSHUser,
		},
		mcnflag.IntFlag{
			Name:   "makecloud-ssh-port",
			Usage:  "SSH port",
			EnvVar: "MAKECLOUD_SSH_PORT",
			Value:  drivers.DefaultSSHPort,
		},
		mcnflag.StringFlag{
			Name:   "makecloud-ssh-keypath",
			Usage:  "SSH private key path (defaults to store path)",
			EnvVar: "MAKECLOUD_SSH_KEYPATH",
		},
	}
}

func (d *Driver) SetConfigFromFlags(opts drivers.DriverOptions) error {
	d.APIToken = opts.String("makecloud-token")
	d.APIBaseURL = opts.String("makecloud-base-url")
	d.APIInsecure = opts.Bool("makecloud-insecure")
	d.APICACert = opts.String("makecloud-ca-cert")
	d.APIClientCert = opts.String("makecloud-client-cert")
	d.APIClientKey = opts.String("makecloud-client-key")

	d.VDCID = opts.String("makecloud-vdc-id")
	d.TemplateID = opts.String("makecloud-template-id")
	d.TemplateName = opts.String("makecloud-template-name")
	d.NetworkID = opts.String("makecloud-network-id")
	d.FirewallTemplateIDs = opts.StringSlice("makecloud-firewall-template-id")
	d.FloatingIP = opts.String("makecloud-floating-ip")
	d.AllocateFloatingIP = opts.Bool("makecloud-allocate-floating-ip")

	d.StorageProfileID = opts.String("makecloud-storage-profile-id")
	d.DiskSizeGB = opts.Int("makecloud-disk-size")
	d.CPU = opts.Int("makecloud-cpu")
	ram, err := strconv.ParseFloat(strings.TrimSpace(opts.String("makecloud-ram")), 64)
	if err != nil {
		return fmt.Errorf("invalid makecloud-ram: %w", err)
	}
	d.RAMGB = ram

	d.Tags = opts.StringSlice("makecloud-tags")
	d.TemplateField = opts.StringSlice("makecloud-template-field")
	d.UserData = opts.String("makecloud-user-data")
	d.NoInjectSSHKey = opts.Bool("makecloud-no-inject-ssh-key")
	d.WaitTimeoutSecs = opts.Int("makecloud-wait-timeout")

	d.SSHUser = opts.String("makecloud-ssh-user")
	d.SSHPort = opts.Int("makecloud-ssh-port")
	if sshKeyPath := opts.String("makecloud-ssh-keypath"); sshKeyPath != "" {
		d.SSHKeyPath = sshKeyPath
	}

	d.SetSwarmConfigFromFlags(opts)
	return nil
}

func (d *Driver) PreCreateCheck() error {
	if d.BaseDriver == nil {
		d.BaseDriver = &drivers.BaseDriver{}
	}

	if strings.TrimSpace(d.APIToken) == "" {
		return errors.New("makecloud-token is required")
	}
	if strings.TrimSpace(d.VDCID) == "" {
		return errors.New("makecloud-vdc-id is required")
	}
	if strings.TrimSpace(d.TemplateID) == "" && strings.TrimSpace(d.TemplateName) == "" {
		return errors.New("makecloud-template-id or makecloud-template-name is required")
	}
	if d.CPU <= 0 {
		return errors.New("makecloud-cpu must be > 0")
	}
	if d.RAMGB <= 0 {
		return errors.New("makecloud-ram must be > 0")
	}
	if d.DiskSizeGB <= 0 {
		return errors.New("makecloud-disk-size must be > 0")
	}
	if d.WaitTimeoutSecs <= 0 {
		return errors.New("makecloud-wait-timeout must be > 0")
	}

	if d.APIBaseURL == "" {
		d.APIBaseURL = bcc.DefaultBaseURL
	}
	if _, err := url.Parse(d.APIBaseURL); err != nil {
		return fmt.Errorf("invalid makecloud-base-url: %w", err)
	}

	m, err := d.manager(context.Background())
	if err != nil {
		return err
	}

	if _, err := m.GetVdc(d.VDCID); err != nil {
		return fmt.Errorf("failed to access VDC %q: %w", d.VDCID, err)
	}

	return nil
}

func (d *Driver) Create() (err error) {
	if err := d.PreCreateCheck(); err != nil {
		return err
	}

	if d.BaseDriver == nil {
		d.BaseDriver = &drivers.BaseDriver{}
	}

	keyPath := d.GetSSHKeyPath()
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return fmt.Errorf("create ssh key dir: %w", err)
	}
	if err := ssh.GenerateSSHKey(keyPath); err != nil {
		return fmt.Errorf("generate ssh key: %w", err)
	}
	pubKeyBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return fmt.Errorf("read ssh public key: %w", err)
	}
	pubKey := strings.TrimSpace(string(pubKeyBytes))

	userData, err := buildFinalUserData(d.UserData, pubKey, d.GetSSHUsername(), d.NoInjectSSHKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.WaitTimeoutSecs)*time.Second)
	defer cancel()

	m, err := d.manager(ctx)
	if err != nil {
		return err
	}

	vdc, err := m.GetVdc(d.VDCID)
	if err != nil {
		return fmt.Errorf("get vdc %q: %w", d.VDCID, err)
	}

	template, err := d.resolveTemplate(vdc, m)
	if err != nil {
		return err
	}

	storageProfile, err := d.resolveStorageProfile(vdc)
	if err != nil {
		return err
	}

	network, err := d.resolveNetwork(vdc, m)
	if err != nil {
		return err
	}

	tags := toTags(d.Tags)

	fwTemplates, err := d.resolveFirewallTemplates(vdc, m)
	if err != nil {
		return err
	}

	port := &bcc.Port{
		Network:           &bcc.Network{ID: network.ID},
		FirewallTemplates: fwTemplates,
		Tags:              tags,
	}
	if err := vdc.CreateEmptyPort(port); err != nil {
		return fmt.Errorf("create port: %w", err)
	}
	d.PortID = port.ID

	defer func() {
		if err == nil {
			return
		}

		cleanupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cleanupManager, mErr := d.manager(cleanupCtx)
		if mErr != nil {
			log.Debugf("cleanup: failed to create manager: %v", mErr)
			return
		}

		if d.VMID != "" {
			vm, vmErr := cleanupManager.GetVm(d.VMID)
			if vmErr == nil {
				_ = vm.Delete()
			}
		}
		if d.PortID != "" {
			p, pErr := cleanupManager.GetPort(d.PortID)
			if pErr == nil {
				_ = p.Delete()
			}
		}
		if d.AllocatedFloatingID != "" {
			p, pErr := cleanupManager.GetPort(d.AllocatedFloatingID)
			if pErr == nil {
				_ = p.Delete()
			}
		}
	}()

	disk := &bcc.Disk{
		Name:           fmt.Sprintf("%s-root", d.GetMachineName()),
		Size:           d.DiskSizeGB,
		StorageProfile: storageProfile,
	}

	metadata, err := d.resolveMetadata(template, pubKey)
	if err != nil {
		return err
	}

	vm := &bcc.Vm{
		Name:     d.GetMachineName(),
		Cpu:      d.CPU,
		Ram:      d.RAMGB,
		Power:    true,
		Template: &bcc.Template{ID: template.ID},
		Metadata: metadata,
		Ports:    []*bcc.Port{port},
		Disks:    []*bcc.Disk{disk},
		Tags:     tags,
	}
	if userData != "" {
		vm.UserData = &userData
	}

	floatingRequested := strings.TrimSpace(d.FloatingIP)
	allocateFloating := d.AllocateFloatingIP && floatingRequested == ""
	if floatingRequested != "" {
		if net.ParseIP(floatingRequested) != nil {
			vm.Floating = &bcc.Port{IpAddress: &floatingRequested}
		} else {
			vm.Floating = &bcc.Port{ID: floatingRequested}
		}
	} else if allocateFloating {
		fport, ferr := d.allocateFloatingPort(vdc, m, tags)
		if ferr != nil {
			return ferr
		}
		d.AllocatedFloatingID = fport.ID
		vm.Floating = &bcc.Port{ID: fport.ID}
	}

	if err := vdc.CreateVm(vm); err != nil {
		return fmt.Errorf("create vm: %w", err)
	}
	d.VMID = vm.ID

	// If a floating IP was requested, ensure security templates are applied to
	// the floating port too (it controls inbound connectivity from the Internet).
	if floatingRequested != "" || allocateFloating {
		desired, derr := d.resolveFirewallTemplates(vdc, m)
		if derr != nil {
			return derr
		}

		vmReload, rerr := m.GetVm(vm.ID)
		if rerr != nil {
			return fmt.Errorf("reload vm %q after create: %w", vm.ID, rerr)
		}
		if vmReload.Floating != nil && vmReload.Floating.ID != "" {
			if err := ensurePortFirewallTemplates(vmReload.Floating, desired); err != nil {
				return err
			}
		}
	}

	ip, err := d.waitForIPv4(m, vm.ID)
	if err != nil {
		return err
	}
	d.IPAddress = ip

	return nil
}

func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

func (d *Driver) GetURL() (string, error) {
	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("tcp://%s:2376", ip), nil
}

func (d *Driver) GetState() (state.State, error) {
	if d.VMID == "" {
		return state.None, nil
	}

	m, err := d.manager(context.Background())
	if err != nil {
		return state.Error, err
	}

	vm, err := m.GetVm(d.VMID)
	if err != nil {
		if isNotFound(err) {
			return state.None, nil
		}
		return state.Error, err
	}

	if vm.Power {
		return state.Running, nil
	}
	return state.Stopped, nil
}

func (d *Driver) Start() error {
	if d.VMID == "" {
		return errors.New("vm id is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.WaitTimeoutSecs)*time.Second)
	defer cancel()
	m, err := d.manager(ctx)
	if err != nil {
		return err
	}

	vm, err := m.GetVm(d.VMID)
	if err != nil {
		return err
	}
	if err := vm.PowerOn(); err != nil {
		return err
	}

	ip, err := d.waitForIPv4(m, d.VMID)
	if err == nil {
		d.IPAddress = ip
	}
	return nil
}

func (d *Driver) Stop() error {
	if d.VMID == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.WaitTimeoutSecs)*time.Second)
	defer cancel()
	m, err := d.manager(ctx)
	if err != nil {
		return err
	}

	vm, err := m.GetVm(d.VMID)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}
	return vm.PowerOff()
}

func (d *Driver) Restart() error {
	if d.VMID == "" {
		return errors.New("vm id is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.WaitTimeoutSecs)*time.Second)
	defer cancel()
	m, err := d.manager(ctx)
	if err != nil {
		return err
	}

	vm, err := m.GetVm(d.VMID)
	if err != nil {
		return err
	}
	return vm.Reboot()
}

func (d *Driver) Kill() error {
	return d.Stop()
}

func (d *Driver) Remove() error {
	if d.VMID == "" && d.PortID == "" {
		return nil
	}

	timeoutSecs := d.WaitTimeoutSecs
	if timeoutSecs <= 0 {
		timeoutSecs = 600
	}
	minRemoveSecs := bcc.LockTimeout + 60
	if timeoutSecs < minRemoveSecs {
		timeoutSecs = minRemoveSecs
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
	defer cancel()
	m, err := d.manager(ctx)
	if err != nil {
		return err
	}

	if d.VMID != "" {
		if err := d.removeVM(ctx, m, d.VMID); err != nil {
			return err
		}
		d.VMID = ""
	}

	if d.PortID != "" {
		if err := d.removePort(ctx, m, d.PortID); err != nil {
			return err
		}
		d.PortID = ""
	}

	if d.AllocatedFloatingID != "" {
		if err := d.removePort(ctx, m, d.AllocatedFloatingID); err != nil {
			return err
		}
		d.AllocatedFloatingID = ""
	}

	return nil
}

func (d *Driver) removeVM(ctx context.Context, m *bcc.Manager, vmID string) error {
	vm, err := m.GetVm(vmID)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}

	if err := vm.Delete(); err != nil && !isNotFound(err) {
		return err
	}

	return waitForGone(ctx, 5*time.Second, func() error {
		_, err := m.GetVm(vmID)
		return err
	})
}

func (d *Driver) removePort(ctx context.Context, m *bcc.Manager, portID string) error {
	p, err := m.GetPort(portID)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}

	if err := p.Delete(); err != nil && !isNotFound(err) {
		return err
	}

	return waitForGone(ctx, 3*time.Second, func() error {
		_, err := m.GetPort(portID)
		return err
	})
}

func waitForGone(ctx context.Context, poll time.Duration, check func() error) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := check()
		if err != nil {
			if isNotFound(err) {
				return nil
			}
			return err
		}

		time.Sleep(poll)
	}
}

func (d *Driver) GetIP() (string, error) {
	if d.IPAddress != "" {
		return d.IPAddress, nil
	}
	if d.VMID == "" {
		return "", errors.New("ip address is not set")
	}

	m, err := d.manager(context.Background())
	if err != nil {
		return "", err
	}

	ip, err := d.waitForIPv4(m, d.VMID)
	if err != nil {
		return "", err
	}
	d.IPAddress = ip
	return ip, nil
}

type bccLogger struct{}

func (l *bccLogger) Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func (d *Driver) manager(ctx context.Context) (*bcc.Manager, error) {
	m, err := bcc.NewManager(d.APIToken, d.APICACert, d.APIClientCert, d.APIClientKey, d.APIInsecure)
	if err != nil {
		return nil, err
	}
	if d.APIBaseURL != "" {
		m.BaseURL = d.APIBaseURL
	}
	m.UserAgent = "docker-machine-driver-makecloud"
	m.Logger = &bccLogger{}
	return m.WithContext(ctx), nil
}

func (d *Driver) resolveTemplate(vdc *bcc.Vdc, m *bcc.Manager) (*bcc.Template, error) {
	if strings.TrimSpace(d.TemplateID) != "" {
		t, err := m.GetTemplate(strings.TrimSpace(d.TemplateID))
		if err != nil {
			return nil, fmt.Errorf("get template %q: %w", d.TemplateID, err)
		}
		return t, nil
	}

	templates, err := vdc.GetTemplates()
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	targetName := strings.TrimSpace(d.TemplateName)
	for _, t := range templates {
		if t != nil && t.Name == targetName {
			return t, nil
		}
	}

	return nil, fmt.Errorf("template %q not found in VDC %q (set makecloud-template-id instead)", targetName, d.VDCID)
}

func (d *Driver) resolveStorageProfile(vdc *bcc.Vdc) (*bcc.StorageProfile, error) {
	if strings.TrimSpace(d.StorageProfileID) != "" {
		sp, err := vdc.GetStorageProfile(strings.TrimSpace(d.StorageProfileID))
		if err != nil {
			return nil, fmt.Errorf("get storage profile %q: %w", d.StorageProfileID, err)
		}
		return sp, nil
	}

	sps, err := vdc.GetStorageProfiles()
	if err != nil {
		return nil, fmt.Errorf("list storage profiles: %w", err)
	}
	if len(sps) == 1 && sps[0] != nil {
		return sps[0], nil
	}
	return nil, errors.New("makecloud-storage-profile-id is required (VDC has multiple storage profiles)")
}

func (d *Driver) resolveNetwork(vdc *bcc.Vdc, m *bcc.Manager) (*bcc.Network, error) {
	if strings.TrimSpace(d.NetworkID) != "" {
		n, err := m.GetNetwork(strings.TrimSpace(d.NetworkID))
		if err != nil {
			return nil, fmt.Errorf("get network %q: %w", d.NetworkID, err)
		}
		return n, nil
	}

	networks, err := vdc.GetNetworks()
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	for _, n := range networks {
		if n != nil && n.IsDefault {
			return n, nil
		}
	}
	if len(networks) > 0 && networks[0] != nil {
		return networks[0], nil
	}
	return nil, errors.New("no networks found in VDC")
}

func (d *Driver) allocateFloatingPort(vdc *bcc.Vdc, m *bcc.Manager, tags []bcc.Tag) (*bcc.Port, error) {
	networkID, err := d.resolveFloatingNetworkID(vdc, m)
	if err != nil {
		return nil, err
	}

	fwTemplates, err := d.resolveFirewallTemplates(vdc, m)
	if err != nil {
		return nil, err
	}

	port := &bcc.Port{
		Network:           &bcc.Network{ID: networkID},
		FirewallTemplates: fwTemplates,
		Tags:              tags,
	}
	if err := vdc.CreateEmptyPort(port); err != nil {
		return nil, fmt.Errorf("create floating port: %w", err)
	}
	return port, nil
}

func (d *Driver) resolveFloatingNetworkID(vdc *bcc.Vdc, m *bcc.Manager) (string, error) {
	// Best-effort: if there are already external ports in this VDC, reuse their
	// network ID (it's the external network that has a public IP pool).
	{
		var ports []*bcc.Port
		args := bcc.Arguments{
			"vdc":         vdc.ID,
			"filter_type": "external",
		}
		if err := m.GetItems("v1/port", args, &ports); err == nil {
			for _, p := range ports {
				if p == nil || p.Network == nil {
					continue
				}
				id := strings.TrimSpace(p.Network.ID)
				if id != "" {
					return id, nil
				}
			}
		}
	}

	// Fallback: try to locate an external-looking network by name.
	networks, err := vdc.GetNetworks()
	if err != nil {
		return "", fmt.Errorf("list networks: %w", err)
	}
	for _, n := range networks {
		if n == nil {
			continue
		}
		if networkNameLooksExternal(n.Name) && strings.TrimSpace(n.ID) != "" {
			return strings.TrimSpace(n.ID), nil
		}
	}

	return "", errors.New("unable to auto-allocate floating IP: external network not found or not accessible; specify --makecloud-floating-ip to use an existing public IP")
}

func networkNameLooksExternal(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	hints := []string{"ext", "external", "public", "публич", "интернет", "internet"}
	for _, h := range hints {
		if strings.Contains(n, h) {
			return true
		}
	}
	return false
}

func (d *Driver) resolveFirewallTemplates(vdc *bcc.Vdc, m *bcc.Manager) ([]*bcc.FirewallTemplate, error) {
	if len(d.FirewallTemplateIDs) == 0 {
		return nil, nil
	}

	res := make([]*bcc.FirewallTemplate, 0, len(d.FirewallTemplateIDs))
	for _, id := range d.FirewallTemplateIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, err := m.GetFirewallTemplate(id); err != nil {
			return nil, fmt.Errorf("get firewall template %q: %w", id, err)
		}
		res = append(res, &bcc.FirewallTemplate{ID: id})
	}
	return dedupeFirewallTemplates(res), nil
}

func ensurePortFirewallTemplates(port *bcc.Port, desired []*bcc.FirewallTemplate) error {
	if port == nil || port.ID == "" || len(desired) == 0 {
		return nil
	}
	if hasAllFirewallTemplates(port.FirewallTemplates, desired) {
		return nil
	}
	merged := mergeFirewallTemplates(port.FirewallTemplates, desired)
	if err := port.UpdateFirewall(merged); err != nil {
		return fmt.Errorf("update firewall templates for port %q: %w", port.ID, err)
	}
	return nil
}

func hasAllFirewallTemplates(existing []*bcc.FirewallTemplate, desired []*bcc.FirewallTemplate) bool {
	if len(desired) == 0 {
		return true
	}
	existingIDs := map[string]bool{}
	for _, ft := range existing {
		if ft == nil {
			continue
		}
		id := strings.TrimSpace(ft.ID)
		if id == "" {
			continue
		}
		existingIDs[id] = true
	}
	for _, ft := range desired {
		if ft == nil {
			continue
		}
		id := strings.TrimSpace(ft.ID)
		if id == "" {
			continue
		}
		if !existingIDs[id] {
			return false
		}
	}
	return true
}

func mergeFirewallTemplates(existing []*bcc.FirewallTemplate, desired []*bcc.FirewallTemplate) []*bcc.FirewallTemplate {
	out := make([]*bcc.FirewallTemplate, 0, len(existing)+len(desired))
	seen := map[string]bool{}

	add := func(ft *bcc.FirewallTemplate) {
		if ft == nil {
			return
		}
		id := strings.TrimSpace(ft.ID)
		if id == "" || seen[id] {
			return
		}
		out = append(out, &bcc.FirewallTemplate{ID: id})
		seen[id] = true
	}

	for _, ft := range desired {
		add(ft)
	}
	for _, ft := range existing {
		add(ft)
	}

	return out
}

func dedupeFirewallTemplates(in []*bcc.FirewallTemplate) []*bcc.FirewallTemplate {
	out := make([]*bcc.FirewallTemplate, 0, len(in))
	seen := map[string]bool{}
	for _, ft := range in {
		if ft == nil {
			continue
		}
		id := strings.TrimSpace(ft.ID)
		if id == "" || seen[id] {
			continue
		}
		out = append(out, &bcc.FirewallTemplate{ID: id})
		seen[id] = true
	}
	return out
}

func (d *Driver) resolveMetadata(template *bcc.Template, sshPublicKey string) ([]*bcc.VmMetadata, error) {
	if len(d.TemplateField) == 0 {
		return nil, nil
	}

	fields, err := template.GetFields()
	if err != nil {
		return nil, fmt.Errorf("get template fields for %q: %w", template.ID, err)
	}

	_ = sshPublicKey

	values := map[string]string{}
	for _, pair := range d.TemplateField {
		k, v, err := splitKeyValue(pair)
		if err != nil {
			return nil, fmt.Errorf("invalid makecloud-template-field value %q: %w", pair, err)
		}
		values[k] = v
	}

	byID := map[string]*bcc.TemplateField{}
	byAlias := map[string]*bcc.TemplateField{}
	byName := map[string]*bcc.TemplateField{}
	for _, f := range fields {
		if f == nil {
			continue
		}
		byID[f.ID] = f
		if f.SystemAlias != "" {
			byAlias[f.SystemAlias] = f
		}
		if f.Name != "" {
			byName[f.Name] = f
		}
	}

	resolved := map[string]string{}
	for key, value := range values {
		f := byID[key]
		if f == nil {
			f = byAlias[key]
		}
		if f == nil {
			f = byName[key]
		}
		if f == nil {
			return nil, fmt.Errorf("unknown template field %q; use field ID, system_alias or name", key)
		}
		resolved[f.ID] = value
	}

	for _, f := range fields {
		if f == nil || !f.Required {
			continue
		}
		if _, ok := resolved[f.ID]; ok {
			continue
		}
		if f.Default == "" {
			return nil, fmt.Errorf("template field %q (%s) is required but no value provided", f.Name, f.ID)
		}
		resolved[f.ID] = f.Default
	}

	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i] == nil || fields[j] == nil {
			return false
		}
		return fields[i].Position < fields[j].Position
	})

	out := make([]*bcc.VmMetadata, 0, len(resolved))
	for _, f := range fields {
		if f == nil {
			continue
		}
		if v, ok := resolved[f.ID]; ok {
			out = append(out, &bcc.VmMetadata{
				Field: bcc.TemplateField{ID: f.ID},
				Value: v,
			})
		}
	}
	return out, nil
}

func (d *Driver) waitForIPv4(m *bcc.Manager, vmID string) (string, error) {
	deadline := time.Now().Add(time.Duration(d.WaitTimeoutSecs) * time.Second)

	for time.Now().Before(deadline) {
		vm, err := m.GetVm(vmID)
		if err != nil {
			if isNotFound(err) {
				return "", fmt.Errorf("vm %q not found", vmID)
			}
			log.Debugf("get vm %q failed (retrying): %v", vmID, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if vm.Floating != nil && vm.Floating.IpAddress != nil {
			ip := strings.TrimSpace(*vm.Floating.IpAddress)
			if ip != "" {
				return ip, nil
			}
		}

		for _, p := range vm.Ports {
			if p != nil && p.IpAddress != nil {
				ip := strings.TrimSpace(*p.IpAddress)
				if ip != "" {
					return ip, nil
				}
			}
		}

		time.Sleep(5 * time.Second)
	}

	return "", fmt.Errorf("timed out waiting for an IP address (vm %q)", vmID)
}

func isNotFound(err error) bool {
	var apiErr *bcc.ApiError
	if errors.As(err, &apiErr) && apiErr.Code() == 404 {
		return true
	}
	return false
}

func toTags(values []string) []bcc.Tag {
	tags := make([]bcc.Tag, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		tags = append(tags, bcc.Tag{Name: v})
	}
	return tags
}

func splitKeyValue(s string) (string, string, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", "", errors.New("expected key=value")
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", errors.New("key is empty")
	}
	return key, parts[1], nil
}
